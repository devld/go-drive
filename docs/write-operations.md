# 写入目标准备与执行

`Save`、`Upload`、`Copy`、`Move` 的执行接口只操作传入的目标。`override=false` 表示目标存在时拒绝，不再隐式自动改名。需要自动命名的调用方先调用 `driveutil.PrepareWrite`。

## 准备接口

`types.WritePreparer.PrepareWrite(ctx, path, mode)` 返回 `types.WritePlan`：

- `Path`：调用方命名空间中的最终虚拟路径。
- `Parents`：执行时需要创建的父目录，按从外到内排序。
- `Override`：执行时是否允许覆盖该确切目标。

`WriteReject` 拒绝已有目标；`WriteRename` 选择一个不存在的同级名称；`WriteOverwrite` 保留请求路径并允许覆盖。准备只查询，不创建目录、不删除或写入内容，也不预留名称。

Dispatcher 和挂载层在各自的虚拟视图中检查冲突。权限包装层通过这一接口获得最终目标，检查其有效权限、子路径规则及缺失父目录的权限。Chroot 转换输入和返回路径；监听包装层准备时不发布访问或修改事件。底层存储驱动可以使用 `driveutil.PrepareWrite` 的普通只读检查实现。

```go
plan, e := driveutil.PrepareWrite(ctx, userDrive, target, types.WriteRename)
if e != nil {
    return e
}
_, e = userDrive.Save(taskCtx, plan.Path, size, plan.Override, reader)
return e
```

准备结果不是授权凭据。执行接口仍检查目标和操作特有的权限：Copy 检查源读取范围，Move 检查源及其子树的读写权限。执行阶段不再分配另一个名称。若目标发生冲突，应返回错误；需要重新选名时，必须重新准备并授权。后端的条件写入能力决定了并发冲突能否原子拒绝，准备本身不提供事务或回滚。

## 调用入口

- `/copy`、`/move` 以及复制任务在非覆盖模式下先选择最终名称，再执行。
- 首次 `/upload` 请求选择目标并在响应 `path` 中返回；客户端后续写入、分片和完成回调使用该路径。
- `/write` 和分块上传的完成写入使用确切目标，非覆盖模式下冲突报错。直接调用执行接口的代码也需要遵守此约定。
- Upload 后续动作不重新选名；具体驱动只能为传入目标签发上传配置，不能自行重命名。

## WebDAV

WebDAV 文件系统适配器实现完整的 Copy/Move 操作。Handler 继续处理请求头、锁及状态码，但不先删除目标再触发源权限检查。禁止覆盖且目标已存在时返回 `412`，不会自动改名。

递归 Copy 通过 Drive.Copy 完成源和目标的子树检查。`Depth: 0` 的目录复制只读取源目录本身，目标替换仍检查将被删除的全部子树。权限拒绝发生在目录创建、目标删除和文件复制之前。具体存储错误或取消仍可能导致部分执行，本机制不提供跨存储事务。

目录复制的回退执行在覆盖时替换已有目标，避免保留旧目标中源目录没有的条目。原生复制与回退都使用同一个最终目标。

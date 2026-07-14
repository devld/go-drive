---
title: 自动任务
description: 使用 Cron 计划或文件事件触发 go-drive 的复制、移动、删除和 JavaScript 操作，并查看执行历史。
lang: zh-CN
translation_key: jobs
source_hash: 243ac6567c3d1f4c439a0e0e68aa39247492a14a92c629d7b4f9cefe470e47bc
---

# 自动任务

在“管理员 → 任务”创建后台任务。一个任务包含启用状态、一个或多个触发器，以及一个动作。

## 触发器

### 标准 Cron

当前使用标准 5 段 cron，不包含秒，也不支持 Quartz 的 `?`：

```text
┌──────── 分钟 0-59
│ ┌────── 小时 0-23
│ │ ┌──── 月中的日期 1-31
│ │ │ ┌── 月份 1-12
│ │ │ │ ┌ 星期 0-6
│ │ │ │ │
0 2 * * *
```

上例每天本地时间 02:00 执行。调度器使用 go-drive 进程的本地时区；容器中应显式确认时区设置。

### 文件事件

文件事件触发器包含：

- 路径模式，例如 `incoming/**/*.jpg`。
- 事件类型：`updated`、`deleted`，可多选。

更新事件也可能代表新建或覆盖。事件触发数据会传给脚本动作：

```js
log(JSON.stringify($event))
// {
//   type: "entry",
//   data: { path: "...", eventType: "updated", includeDescendants: "false" }
// }
```

任务对文件的修改可能再次匹配事件触发器。设计路径规则时避免任务自触发循环，例如把输入和输出目录分开并排除输出路径。

## 动作

当前支持四类动作。

### 复制/移动

- 源路径可以每行一个并使用通配符。
- 目标必须是已经存在的目录。
- 可选择移动和覆盖。
- 跨 Drive、目录或不支持原生复制的存储会通过服务器传输。

### 删除

每行一个路径模式。删除按匹配结果逆序执行，以便先删除子项。请先用低权限测试路径或脚本 `ls` 验证模式，避免过宽的 `**`。

### 组合（Flow）

组合动作可以将多个操作（复制、删除、脚本等）组合到一个任务中，按顺序依次执行。每一步可以单独开启"忽略错误"，使该步失败后不影响后续步骤的执行。

### JavaScript

常用函数：

```ts
cp(from, to, override)
mv(from, to, override)
rm(path)
ls(path)
mkdir(path)
log(message)
```

以及通用运行时中的 `http`、`newContext`、`newContextWithTimeout`、`sleep`、`pathUtils`、`encUtils`、错误构造与 Drive API。完整类型定义位于代码仓库：

- [`docs/scripts/global.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/global.d.ts)
- [`docs/scripts/env/jobs.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/env/jobs.d.ts)
- [`docs/scripts/libs`](https://github.com/devld/go-drive/tree/master/docs/scripts/libs)

示例：

```js
log('trigger: ' + JSON.stringify($event))

// 把 incoming 下所有 jpg 复制到 archive，允许覆盖
cp('incoming/**/*.jpg', 'archive', true)

// 调用外部 webhook
var ctx = newContextWithTimeout(newContext(), ms(10000))
try {
  var resp = http(ctx, 'POST', 'https://example.com/hook', {
    'content-type': 'application/json'
  }, JSON.stringify($event))
  try {
    log('webhook: ' + resp.Status)
  } finally {
    resp.Dispose()
  }
} finally {
  ctx.Cancel()
}
```

任务脚本属于受信任管理员代码，可访问根 Drive 和网络。不要运行来源不明的脚本，也不要把密钥直接写进会展示给其他管理员的日志。

## 执行、日志和中止

- 列表中的执行按钮可手动立即触发任务。
- 编辑脚本时可以在线试运行并查看日志。
- 执行历史记录开始/完成时间、状态、日志和错误。
- 运行中的任务可中止；是否能立即停止取决于底层远端请求是否响应上下文取消。
- 可以清空历史执行记录。

调试时先手动执行，再启用 Cron 或事件触发器。对移动、删除和递归模式使用专门的测试目录。

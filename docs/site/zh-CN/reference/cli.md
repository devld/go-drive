---
title: 命令行参考
description: 使用 go-drive 命令行参数选择配置文件、输出版本信息、控制启动并执行管理操作。
lang: zh-CN
translation_key: cli
source_hash: 0c1b026b35dcaa883a41f1dbd28e48f549c4471d59066ee8c1ffad4e588ac4f9
---

# 命令行参考

```text
-c <path>      指定 YAML 配置文件
-show-config   输出解析并补全默认值后的配置，然后退出
-v             输出版本、修订和构建时间，然后退出
```

示例：

```bash
./go-drive -c /etc/go-drive/config.yml
./go-drive -c ./config.yml -show-config
./go-drive -v
```

没有 `-c` 时，如果工作目录存在 `config.yml` 就自动读取；否则使用内置默认值。

## 环境变量

```text
GO_DRIVE_LOGGING_LEVEL=debug
```

覆盖配置文件中的 `logging.level`。仅在排障期间使用 `debug`，完成后移除该环境变量。

构建时常用 Make 变量：

```bash
BUILD_VERSION=dev BUILD_REV=$(git rev-parse HEAD) make all
```

`make all` 需要前端工具链和 CGO；普通 `go build` 不会生成包含 Web UI 的完整发布包。

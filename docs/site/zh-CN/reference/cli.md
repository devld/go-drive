---
title: 命令行参考
description: 使用 go-drive 命令行参数选择配置文件、输出版本信息、控制启动并执行管理操作。
lang: zh-CN
translation_key: cli
source_hash: 17ef0e9ae63515081d4f50424d23833034d049bad03039b502a2f1fbd39146fb
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
GO_DRIVE_DEBUG=1
```

启用额外调试行为和日志。仅在排障期间使用，完成后关闭。

构建时常用 Make 变量：

```bash
BUILD_VERSION=dev BUILD_REV=$(git rev-parse HEAD) make all
```

`make all` 需要前端工具链和 CGO；普通 `go build` 不会生成包含 Web UI 的完整发布包。

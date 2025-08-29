# Go-Drive OpenWrt 包

这个目录包含了为 OpenWrt 系统构建 Go-Drive 云盘服务的所有必要文件。

## 功能特性

- ✅ 通过 OpenWrt 服务管理 Go-Drive (启动/停止/重启)
- ✅ 现代化的 LuCI Web 界面配置
- ✅ UCI 配置系统集成
- ✅ 支持所有 Go-Drive 配置选项
- ✅ 多语言支持 (英文/中文)
- ✅ 自动包构建 (GitHub Actions)

## 文件结构

```
docs/openwrt/
├── Makefile                              # OpenWrt 包构建配置
├── README.md                             # 说明文档
├── files/                                # 系统文件
│   ├── etc/
│   │   ├── config/go-drive              # UCI 配置文件
│   │   ├── init.d/go-drive              # 系统初始化脚本
│   │   └── uci-defaults/go-drive        # 默认配置脚本
└── luci-app-go-drive/                   # LuCI 应用
    ├── htdocs/luci-static/resources/view/
    │   └── go-drive.js                  # 现代 LuCI JS 视图
    ├── luasrc/
    │   ├── controller/go-drive.lua      # LuCI 控制器
    │   └── model/cbi/go-drive.lua       # CBI 模型 (兼容性)
    ├── po/zh_Hans/go-drive.po           # 中文语言包
    └── root/usr/share/luci/menu.d/
        └── luci-app-go-drive.json       # LuCI 菜单配置
```

## 安装方法

### 方法一：使用预构建包

1. 从 [Releases](https://github.com/devld/go-drive/releases) 下载适合你路由器架构的包
2. 解压 tar.gz 文件获得 .ipk 文件
3. 通过 LuCI 界面或 SSH 安装：

```bash
# 通过 opkg 安装
opkg install go-drive_*.ipk
opkg install luci-app-go-drive_*.ipk

# 或通过 LuCI 界面：系统 -> 软件包 -> 上传软件包
```

### 方法二：从源码构建

1. 准备 OpenWrt 构建环境
2. 克隆此仓库到 OpenWrt 的 package 目录：

```bash
cd openwrt/package
git clone https://github.com/devld/go-drive.git
ln -sf go-drive/docs/openwrt go-drive-pkg
```

3. 配置并构建：

```bash
make menuconfig
# 选择: Network -> Cloud Manager -> go-drive
# 选择: LuCI -> Applications -> luci-app-go-drive

make package/go-drive-pkg/compile V=s
```

## 配置说明

安装完成后，你可以通过以下方式配置 Go-Drive：

### LuCI Web 界面

1. 登录到 OpenWrt 管理界面
2. 导航到：服务 (Services) -> Go-Drive
3. 配置各项参数：

#### 基本设置
- **启用**: 开启/关闭 Go-Drive 服务
- **监听地址**: 服务监听的地址和端口 (默认: :8089)
- **数据目录**: 存储应用数据的目录 (默认: /opt/go-drive)
- **默认语言**: 界面默认语言

#### 数据库设置
- **数据库类型**: SQLite 或 MySQL
- **数据库名称**: 数据库名或文件名
- **连接参数**: MySQL 连接参数 (仅 MySQL)

#### 高级设置
- **缩略图设置**: 缩略图缓存配置
- **认证设置**: 会话和认证配置
- **搜索设置**: 文件搜索功能
- **WebDAV设置**: WebDAV 服务配置

### 命令行配置

你也可以通过 UCI 命令行配置：

```bash
# 启用服务
uci set go-drive.config.enabled='1'
uci commit go-drive

# 修改监听端口
uci set go-drive.config.listen=':9000'
uci commit go-drive

# 启动服务
/etc/init.d/go-drive start
```

## 服务管理

```bash
# 启动服务
/etc/init.d/go-drive start

# 停止服务
/etc/init.d/go-drive stop

# 重启服务
/etc/init.d/go-drive restart

# 重载配置
/etc/init.d/go-drive reload

# 查看状态
/etc/init.d/go-drive status

# 开机自启
/etc/init.d/go-drive enable

# 禁用自启
/etc/init.d/go-drive disable
```

## 访问 Go-Drive

服务启动后，你可以通过以下地址访问 Go-Drive：

```
http://路由器IP:8089
```

例如：`http://192.168.1.1:8089`

## 故障排除

### 检查服务状态

```bash
# 检查进程是否运行
ps | grep go-drive

# 检查日志
logread | grep go-drive

# 检查配置文件
cat /opt/go-drive/config.yml
```

### 常见问题

1. **服务无法启动**
   - 检查端口是否被占用
   - 确保数据目录权限正确
   - 查看系统日志确定错误原因

2. **无法访问 Web 界面**
   - 确认防火墙设置
   - 检查监听地址配置
   - 验证服务是否正常运行

3. **配置更改不生效**
   - 重启 go-drive 服务
   - 检查 UCI 配置是否正确提交

## 开发说明

### 修改配置文件

如需修改配置项，请编辑以下文件：

- `files/etc/config/go-drive`: UCI 配置模板
- `files/etc/init.d/go-drive`: 初始化脚本
- `luci-app-go-drive/htdocs/luci-static/resources/view/go-drive.js`: LuCI 界面

### 添加语言支持

1. 在 `luci-app-go-drive/po/` 目录下创建新的语言目录
2. 翻译 `go-drive.po` 文件
3. 更新 LuCI 视图文件中的语言选项

## 自动构建

项目配置了 GitHub Actions 自动构建，支持以下架构：

- x86_64 (PC/虚拟机)
- aarch64_generic (64位ARM)
- arm_cortex-a9 (32位ARM)
- mips_24kc (MIPS大端)
- mipsel_24kc (MIPS小端)

构建包会自动发布到 GitHub Releases。

## 许可证

本项目与 Go-Drive 主项目使用相同的许可证。

## 支持

如有问题，请在 [GitHub Issues](https://github.com/devld/go-drive/issues) 中反馈。
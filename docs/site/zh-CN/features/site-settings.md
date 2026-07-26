---
title: 站点设置
description: 自定义 go-drive 品牌、匿名访问、文件处理、界面外观、注入样式及其他浏览器端站点行为。
lang: zh-CN
translation_key: site-settings
source_hash: 846c6c8f14f71eb721b9d871a6336d20ca950cbfb1b52c3aaf4413aa44342f94
---

# 站点设置

“管理员 → 站点”保存的是数据库选项，不是 `config.yml`。保存后刷新页面即可生效。

Web 界面当前内置 `en-US`、`zh-CN` 和 `ko-KR`，启动时按浏览器语言选择；无法加载匹配语言时回退到英文。发布二进制已嵌入这些语言资源，不再使用旧版 `lang-dir` 配置。

## 名称、样式和脚本

- “应用名称”修改页面标题和品牌文字。
- “全局 CSS”用于主题覆盖。支持的变量、选择器和完整示例见[自定义主题](./custom-themes.html)。
- “插入脚本”在所有访问者页面的 `<head>` 末尾、全局 CSS 之后执行 JavaScript。此时文档主体尚不存在；需要访问页面元素的脚本必须等待 `DOMContentLoaded`。

最小 CSS 示例：

```css
:root {
  --color-accent: #2563eb;
  --color-accent-strong: #2563eb;
  --radius-control: 10px;
}
```

文档列出的语义变量以及 `data-ui`、`data-surface` 选择器属于主题接口；内部 class 名只是实现细节。注入脚本拥有与站点相同的浏览器权限，能读取登录状态和页面内容，只能使用可信代码。

需要依赖 DOM 的逻辑可以这样延后执行：

```js
document.addEventListener('DOMContentLoaded', () => {
  // 在这里读取或修改页面元素。
})
```

## 文件预览

可分别配置文本、图片、音频、视频、Monaco 和外部预览器的扩展名。详细格式和第三方隐私影响见[文件预览与缩略图](./preview-thumbnail.html)。

## 匿名用户根路径

限制未登录访问者只能看到某个虚拟子目录。它不会自动授予读取权限，还需要为 `ANY` 配置路径权限。参见[用户、组、根路径和权限](../administration/access-control.html)。

## 下载选项

- “代理下载最大大小”限制需要服务器代理的文件大小。
- “ZIP 最大大小”限制打包下载规模。

限制值用于保护服务器带宽、内存和临时空间。云盘关闭代理下载时，浏览器直下可能不受同样路径影响；跨 Drive 和 ZIP 操作仍会消耗服务器资源。

## 缩略图映射

用 `tag1,tag2:<路径模式>` 为不同目录选择配置文件中的 handler。处理器本身在 `config.yml` 中定义，修改后需要重启；映射规则说明见[文件预览与缩略图](./preview-thumbnail.html)。

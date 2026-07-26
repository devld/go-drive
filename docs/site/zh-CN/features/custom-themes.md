---
title: 自定义主题
description: 使用受支持的 CSS 主题接口定制 go-drive 的颜色、毛玻璃、圆角、动画、背景和组件细节。
lang: zh-CN
translation_key: custom-themes
source_hash: 3f106313949b53b4a2bbfec3b5e087664057c87460a04e44782b02f460064993
---

# 自定义主题

在“**管理员 → 站点 → 全局 CSS**”中添加主题，保存后刷新页面。go-drive 会把这段 CSS 追加到应用内置样式之后，因此选择器优先级相同时通常不需要 `!important`。

主题接口分为两层：

1. 优先覆盖语义化自定义属性，调整颜色、表面、圆角、阴影、毛玻璃和动画。
2. 只有在某个组件需要单独处理时，才使用文档列出的 `data-ui` 和 `data-surface` 选择器。

内部 class 名属于实现细节，可能随版本变化。

## 配色模式

默认主题跟随 `prefers-color-scheme`。根元素没有 `data-theme` 属性时表示自动模式；在根元素上使用 `data-theme="light"` 或 `data-theme="dark"` 可以强制使用内置配色。

自动模式下的明亮主题覆盖可以这样写：

```css
@media (prefers-color-scheme: light) {
  :root:not([data-theme]) {
    --color-bg-glass: rgba(255, 255, 255, 0.62);
  }
}
```

当注入脚本或外部集成强制指定配色时，使用属性选择器：

```css
:root[data-theme='light'] {
  --color-bg-glass: rgba(255, 255, 255, 0.62);
}

:root[data-theme='dark'] {
  --color-bg-glass: rgba(20, 24, 31, 0.64);
}
```

两个配色共用的值可以直接放在 `:root` 上。

## 主题属性

### 背景和表面

| 属性 | 用途 |
| --- | --- |
| `--color-bg-canvas` | 页面画布的后备颜色 |
| `--color-bg-surface` | 主要内容表面 |
| `--color-bg-elevated` | 抬升的非毛玻璃内容 |
| `--color-bg-glass` | 半透明毛玻璃表面 |
| `--color-bg-hover` | 指针悬停状态 |
| `--color-bg-focus` | 控件聚焦状态 |
| `--color-bg-selected` | 选中项目状态 |
| `--color-bg-invalid` | 无效或拒绝状态 |
| `--color-bg-table-header` | 表头背景 |

### 文本、表单和分隔线

| 属性 | 用途 |
| --- | --- |
| `--color-text` | 主要文本 |
| `--color-text-muted` | 次要文本 |
| `--color-text-disabled` | 禁用或不可用内容 |
| `--color-border` | 通用分隔线 |
| `--color-field-bg` | 输入框背景 |
| `--color-field-bg-disabled` | 禁用输入框背景 |
| `--color-field-border` | 输入框边框 |
| `--color-glass-border` | 毛玻璃边缘高光 |

### 操作和状态

`--color-accent`、`--color-success`、`--color-warning` 和 `--color-danger` 适合文本和图标。对应的 `-strong` 变量用于实心控件，并配有前景色变量：

```text
--color-accent-strong       --color-on-accent
--color-info-strong         --color-on-info
--color-success-strong      --color-on-success
--color-warning-strong      --color-on-warning
--color-danger-strong       --color-on-danger
```

其他状态变量包括：

```text
--color-overlay
--color-loading-overlay
--color-focus-ring
--color-progress-track
--color-progress-track-paused
--color-progress-value
```

### 形状、层次、毛玻璃和动画

| 属性 | 用途 |
| --- | --- |
| `--radius-control` | 按钮和控件 |
| `--radius-popover` | 下拉框和菜单 |
| `--radius-dialog` | 对话框 |
| `--radius-glass` | 大型毛玻璃表面 |
| `--radius-glass-compact` | 紧凑毛玻璃表面 |
| `--backdrop-filter-field` | 输入框和编辑器字段模糊滤镜 |
| `--backdrop-filter-glass` | 毛玻璃模糊和饱和度滤镜 |
| `--shadow-control` | 控件悬停阴影 |
| `--shadow-elevated` | 对话框和浮层阴影 |
| `--motion-duration-fast` | 小型交互持续时间 |
| `--motion-duration-normal` | 进入和离开持续时间 |
| `--motion-easing-standard` | 标准动画曲线 |
| `--motion-easing-exit` | 离开动画曲线 |

把 `--backdrop-filter-field` 或 `--backdrop-filter-glass` 设为 `none` 可以关闭对应的滤镜。要让模糊清晰可见，背景颜色必须带透明度，而且表面后方需要有可见内容。

## 稳定组件选择器

以下属性是受支持的结构化主题 Hook。

| 选择器 | 目标 |
| --- | --- |
| `[data-ui="app-canvas"]` | 页面画布（`body`） |
| `[data-ui="app"]` | 已挂载的应用 |
| `[data-ui="app-header"]` | 全局页头 |
| `[data-ui="button"]` | 应用按钮或按钮式导航 |
| `[data-ui="explorer"]` | 文件浏览器 |
| `[data-ui="entry-panel"]` | 文件列表主表面 |
| `[data-ui="readme"]` | README 表面 |
| `[data-ui="readme-content"]` | README 内容包装 |
| `[data-ui="markdown"]` | 渲染后的 Markdown |
| `[data-ui="path-bar"]` | 面包屑路径 |
| `[data-ui="path-segment"]` | 路径分段 |
| `[data-ui="path-link"]` | 路径链接 |
| `[data-ui="search-panel"]` | 搜索表面 |
| `[data-ui="dropdown"]` | 下拉组件 |
| `[data-ui="dropdown-trigger"]` | 下拉触发器 |
| `[data-ui="dropdown-panel"]` | 浮动下拉面板 |
| `[data-ui="menu"]` | 浮动项目菜单表面 |
| `[data-ui="menu-content"]` | 项目菜单内容 |
| `[data-ui="menu-item"]` | 项目菜单操作 |
| `[data-ui="dialog-overlay"]` | 对话框遮罩 |
| `[data-ui="dialog"]` | 对话框表面 |
| `[data-ui="dialog-header"]` | 对话框页头 |
| `[data-ui="dialog-body"]` | 对话框主体 |
| `[data-ui="dialog-footer"]` | 对话框操作区 |
| `[data-ui="admin-page"]` | 管理页面布局 |
| `[data-ui="handler-header"]` | 预览或编辑器标题栏 |
| `[data-ui="preview"]` | 预览或编辑器根节点 |
| `[data-ui="entry-drag-status"]` | 拖放状态浮层 |
| `[data-ui="drop-zone"]` | 上传拖放目标 |
| `[data-surface="glass"]` | 任意内置毛玻璃表面 |

可以用状态和变体属性进一步限定选择器：

```css
[data-ui='button'][data-variant='primary'] {}
[data-ui='button'][data-variant='plain'] {}
[data-ui='button'][data-size='compact'] {}
[data-ui='path-segment'][data-root] {}
[data-ui='menu-item'][data-variant='danger'] {}
[data-ui='preview'][data-handler='video'] {}
[aria-disabled='true'] {}
[aria-expanded='true'] {}
```

## 完整毛玻璃主题示例

下面的示例增加固定装饰画布、半透明表面、更明显的毛玻璃、圆角控件和首页标记：

```css
@media (prefers-color-scheme: light) {
  :root:not([data-theme]) {
    --color-bg-canvas: transparent;
    --color-bg-surface: rgba(255, 255, 255, 0.4);
    --color-bg-elevated: rgba(255, 255, 255, 0.72);
    --color-bg-glass: rgba(255, 255, 255, 0.58);
    --color-bg-hover: rgba(255, 255, 255, 0.42);
    --color-text-muted: rgb(61, 61, 61);
    --color-text-disabled: #666;
    --color-field-bg: transparent;
    --color-glass-border: rgba(255, 255, 255, 0.82);
    --backdrop-filter-field: blur(14px) saturate(150%);
    --backdrop-filter-glass: blur(32px) saturate(180%);
    --radius-control: 8px;
    --radius-popover: 12px;
    --radius-dialog: 16px;
  }
}

[data-ui='app-canvas'] {
  background-color: rgb(0, 77, 101);
  background-image: radial-gradient(
    circle at left center,
    rgba(64, 158, 255, 0.2),
    rgba(103, 194, 58, 0.2)
  );
  background-repeat: no-repeat;
  background-position: center;
  background-size: cover;
  background-attachment: fixed;
}

@media (prefers-color-scheme: light) {
  [data-ui='handler-header'] {
    background-color: var(--color-bg-elevated);
  }

  [data-ui='admin-page'] {
    background-color: rgba(255, 255, 255, 0.8);
  }

}

[data-ui='path-segment'][data-root] [data-ui='path-link']::before {
  content: '🏠';
}
```

如果显式强制使用明亮主题，请把明亮主题变量块复制到 `:root[data-theme="light"]` 下。

## 故障排查

- 检查元素时，优先寻找可覆盖的文档变量，再考虑结构选择器。
- 避免使用 `!important`；通常只有覆盖行内样式或高优先级状态时才需要。
- 如果毛玻璃看起来不透明，降低 `--color-bg-glass` 的 Alpha 值。
- 如果背景仍然清晰，确认元素带有 `data-surface="glass"`，并检查 `--backdrop-filter-glass` 没有被设为 `none`。
- 固定背景可能增加移动浏览器的渲染开销；如果滚动不流畅，请移除 `background-attachment: fixed`。
- 自定义 CSS 可以请求外部图片和字体，只应使用可信资源地址。

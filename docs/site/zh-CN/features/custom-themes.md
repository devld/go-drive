---
title: 自定义主题
description: 使用受支持的 CSS 主题接口定制 go-drive 的颜色、毛玻璃、圆角、动画、背景和组件细节。
lang: zh-CN
translation_key: custom-themes
source_hash: 2a910c47a209cc92e7bed767f0e44de8db87120465ee02856350a5b64c581073
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
| `[data-ui="drop-zone-indicator"]` | 全屏上传拖放提示 |
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

## 可直接使用的主题预设

以下预设同时支持自动明暗模式和显式 `data-theme="light"` / `"dark"`。展开预设后点击代码块右上角的复制按钮，再把完整样式粘贴到“全局 CSS”中。

### 纸片风格

使用暖色纸张、横线画布、小圆角、衬线字体、墨水强调色和错位硬阴影，形成多层文具纸片效果。

::: details 展开完整纸片主题 CSS

```css
:root {
  --paper-canvas: #e8dec4;
  --paper-surface: #fffaf0;
  --paper-elevated: #fffdf7;
  --paper-hover: #f2e7ce;
  --paper-focus: #f7ecd3;
  --paper-selected: #ead9b4;
  --paper-invalid: #f2d4c8;
  --paper-text: #332d24;
  --paper-text-muted: #746957;
  --paper-text-disabled: #9a907f;
  --paper-border: #cbbd9c;
  --paper-accent: #8b3f32;
  --paper-accent-strong: #8b3f32;
  --paper-rule: rgba(91, 112, 127, 0.16);
  --paper-margin: rgba(173, 79, 70, 0.2);
  --paper-shadow: rgba(76, 61, 39, 0.26);

  --color-bg-canvas: var(--paper-canvas);
  --color-bg-surface: var(--paper-surface);
  --color-bg-elevated: var(--paper-elevated);
  --color-bg-glass: var(--paper-surface);
  --color-bg-hover: var(--paper-hover);
  --color-bg-focus: var(--paper-focus);
  --color-bg-selected: var(--paper-selected);
  --color-bg-invalid: var(--paper-invalid);
  --color-bg-table-header: var(--paper-hover);
  --color-text: var(--paper-text);
  --color-text-muted: var(--paper-text-muted);
  --color-text-disabled: var(--paper-text-disabled);
  --color-border: var(--paper-border);
  --color-field-bg: var(--paper-elevated);
  --color-field-bg-disabled: var(--paper-hover);
  --color-field-border: var(--paper-border);
  --color-glass-border: var(--paper-border);
  --color-accent: var(--paper-accent);
  --color-accent-strong: var(--paper-accent-strong);
  --color-overlay: rgba(51, 45, 36, 0.48);
  --color-loading-overlay: var(--paper-surface);
  --color-focus-ring: var(--paper-accent);
  --color-progress-track: var(--paper-hover);
  --color-progress-track-paused: var(--paper-selected);
  --color-progress-value: var(--paper-accent);

  --radius-control: 3px;
  --radius-popover: 4px;
  --radius-dialog: 6px;
  --radius-glass: 4px;
  --radius-glass-compact: 3px;
  --backdrop-filter-field: none;
  --backdrop-filter-glass: none;
  --shadow-control: 2px 2px 0 var(--paper-shadow);
  --shadow-elevated: 6px 7px 0 var(--paper-shadow);
}

@media (prefers-color-scheme: dark) {
  :root:not([data-theme]) {
    --paper-canvas: #1d1a16;
    --paper-surface: #29251f;
    --paper-elevated: #302b24;
    --paper-hover: #383127;
    --paper-focus: #40372b;
    --paper-selected: #51422f;
    --paper-invalid: #50322c;
    --paper-text: #eee2c8;
    --paper-text-muted: #b8aa90;
    --paper-text-disabled: #827866;
    --paper-border: #665a47;
    --paper-accent: #e48d72;
    --paper-accent-strong: #9f4e3d;
    --paper-rule: rgba(151, 174, 185, 0.1);
    --paper-margin: rgba(220, 119, 101, 0.16);
    --paper-shadow: rgba(0, 0, 0, 0.42);
  }
}

:root[data-theme='dark'] {
  --paper-canvas: #1d1a16;
  --paper-surface: #29251f;
  --paper-elevated: #302b24;
  --paper-hover: #383127;
  --paper-focus: #40372b;
  --paper-selected: #51422f;
  --paper-invalid: #50322c;
  --paper-text: #eee2c8;
  --paper-text-muted: #b8aa90;
  --paper-text-disabled: #827866;
  --paper-border: #665a47;
  --paper-accent: #e48d72;
  --paper-accent-strong: #9f4e3d;
  --paper-rule: rgba(151, 174, 185, 0.1);
  --paper-margin: rgba(220, 119, 101, 0.16);
  --paper-shadow: rgba(0, 0, 0, 0.42);
}

[data-ui='app-canvas'] {
  background-color: var(--color-bg-canvas);
  background-image:
    linear-gradient(
      to right,
      transparent 0,
      transparent 52px,
      var(--paper-margin) 52px,
      var(--paper-margin) 53px,
      transparent 53px
    ),
    repeating-linear-gradient(
      to bottom,
      transparent 0,
      transparent 27px,
      var(--paper-rule) 27px,
      var(--paper-rule) 28px
    );
  background-attachment: fixed;
}

[data-ui='app'] {
  font-family: Georgia, 'Noto Serif', 'Noto Serif CJK SC', serif;
}

[data-ui='app-header'] {
  border-bottom: 1px solid var(--paper-border);
}

[data-surface='glass'] {
  outline: 1px solid var(--paper-border);
  outline-offset: -1px;
  box-shadow: var(--shadow-elevated);
}

[data-ui='button'] {
  letter-spacing: 0.02em;
}

[data-ui='app-header'] [data-ui='button'] {
  padding: 4px 8px;
  border-bottom: 1px solid currentColor;
}

:is(
  [data-ui='app-header'] [data-ui='button'],
  [data-ui='button'][data-variant='primary'],
  [data-ui='button'][data-variant='info'],
  [data-ui='button'][data-variant='success'],
  [data-ui='button'][data-variant='warning'],
  [data-ui='button'][data-variant='danger']
):active {
  transform: translate(1px, 1px);
  box-shadow: none;
}
```

:::

### 复古终端

使用荧光屏配色、等宽字体、扫描线、方形面板和清晰边框，把文件管理器变成克制的命令终端显示界面。

::: details 展开完整复古终端主题 CSS

```css
:root {
  --terminal-canvas: #e7eadf;
  --terminal-surface: #f4f6ed;
  --terminal-elevated: #fbfcf6;
  --terminal-hover: #dce5d4;
  --terminal-focus: #d3e5cc;
  --terminal-selected: #c7ddbd;
  --terminal-invalid: #ead1cb;
  --terminal-text: #17251a;
  --terminal-muted: #526456;
  --terminal-disabled: #879188;
  --terminal-border: #66806a;
  --terminal-accent: #226b36;
  --terminal-accent-strong: #226b36;
  --terminal-scanline: rgba(34, 107, 54, 0.07);
  --terminal-glow: rgba(34, 107, 54, 0.18);

  --color-bg-canvas: var(--terminal-canvas);
  --color-bg-surface: var(--terminal-surface);
  --color-bg-elevated: var(--terminal-elevated);
  --color-bg-glass: var(--terminal-surface);
  --color-bg-hover: var(--terminal-hover);
  --color-bg-focus: var(--terminal-focus);
  --color-bg-selected: var(--terminal-selected);
  --color-bg-invalid: var(--terminal-invalid);
  --color-bg-table-header: var(--terminal-hover);
  --color-text: var(--terminal-text);
  --color-text-muted: var(--terminal-muted);
  --color-text-disabled: var(--terminal-disabled);
  --color-border: var(--terminal-border);
  --color-field-bg: var(--terminal-elevated);
  --color-field-bg-disabled: var(--terminal-hover);
  --color-field-border: var(--terminal-border);
  --color-glass-border: var(--terminal-border);
  --color-accent: var(--terminal-accent);
  --color-accent-strong: var(--terminal-accent-strong);
  --color-on-accent: #fff;
  --color-overlay: rgba(23, 37, 26, 0.56);
  --color-loading-overlay: var(--terminal-canvas);
  --color-focus-ring: var(--terminal-accent);
  --color-progress-track: var(--terminal-hover);
  --color-progress-track-paused: var(--terminal-selected);
  --color-progress-value: var(--terminal-accent);

  --radius-control: 0;
  --radius-popover: 0;
  --radius-dialog: 0;
  --radius-glass: 0;
  --radius-glass-compact: 0;
  --backdrop-filter-field: none;
  --backdrop-filter-glass: none;
  --shadow-control: 2px 2px 0 var(--terminal-border);
  --shadow-elevated: 4px 4px 0 var(--terminal-border);
}

@media (prefers-color-scheme: dark) {
  :root:not([data-theme]) {
    --terminal-canvas: #050b07;
    --terminal-surface: #09130c;
    --terminal-elevated: #0c1a10;
    --terminal-hover: #102719;
    --terminal-focus: #12321e;
    --terminal-selected: #164326;
    --terminal-invalid: #351718;
    --terminal-text: #9df7a9;
    --terminal-muted: #64ad70;
    --terminal-disabled: #37643f;
    --terminal-border: #2e8b45;
    --terminal-accent: #72f28a;
    --terminal-accent-strong: #287d3d;
    --terminal-scanline: rgba(114, 242, 138, 0.055);
    --terminal-glow: rgba(114, 242, 138, 0.28);
  }
}

:root[data-theme='dark'] {
  --terminal-canvas: #050b07;
  --terminal-surface: #09130c;
  --terminal-elevated: #0c1a10;
  --terminal-hover: #102719;
  --terminal-focus: #12321e;
  --terminal-selected: #164326;
  --terminal-invalid: #351718;
  --terminal-text: #9df7a9;
  --terminal-muted: #64ad70;
  --terminal-disabled: #37643f;
  --terminal-border: #2e8b45;
  --terminal-accent: #72f28a;
  --terminal-accent-strong: #287d3d;
  --terminal-scanline: rgba(114, 242, 138, 0.055);
  --terminal-glow: rgba(114, 242, 138, 0.28);
}

[data-ui='app-canvas'] {
  background-color: var(--color-bg-canvas);
  background-image: repeating-linear-gradient(
    to bottom,
    transparent 0,
    transparent 3px,
    var(--terminal-scanline) 3px,
    var(--terminal-scanline) 4px
  );
  background-attachment: fixed;
}

[data-ui='app'] {
  font-family: 'Cascadia Mono', 'SFMono-Regular', Consolas, monospace;
  text-shadow: 0 0 8px var(--terminal-glow);
}

[data-surface='glass'] {
  outline: 1px solid var(--terminal-border);
  outline-offset: -1px;
}

[data-ui='app-header'] {
  border-bottom: 3px double var(--terminal-border);
}

[data-ui='button'] {
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

[data-ui='app-header'] [data-ui='button'] {
  padding: 4px 8px;
  border: 1px solid currentColor;
  background-color: var(--color-bg-surface);
}

[data-ui='path-segment'][data-root] [data-ui='path-link']::before {
  content: '> ';
}
```

:::

## 故障排查

- 检查元素时，优先寻找可覆盖的文档变量，再考虑结构选择器。
- 避免使用 `!important`；通常只有覆盖行内样式或高优先级状态时才需要。
- 如果毛玻璃看起来不透明，降低 `--color-bg-glass` 的 Alpha 值。
- 如果背景仍然清晰，确认元素带有 `data-surface="glass"`，并检查 `--backdrop-filter-glass` 没有被设为 `none`。
- 固定背景可能增加移动浏览器的渲染开销；如果滚动不流畅，请移除 `background-attachment: fixed`。
- 自定义 CSS 可以请求外部图片和字体，只应使用可信资源地址。

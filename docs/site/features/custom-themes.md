---
title: Custom Themes
description: Customize go-drive colors, glass effects, radii, motion, backgrounds, and component details with the supported CSS theme API.
lang: en
translation_key: custom-themes
---

# Custom Themes

Add a theme in **Admin → Site → Global CSS**, save it, and refresh the page.
go-drive appends this CSS after the bundled application stylesheet, so a
documented selector with equal specificity normally does not need
`!important`.

The theme API has two layers:

1. Override semantic custom properties for colors, surfaces, radii, shadows,
   glass effects, and motion.
2. Use documented `data-ui` and `data-surface` selectors only when a particular
   component needs different treatment.

Internal class names are implementation details and may change.

## Color schemes

The default theme follows `prefers-color-scheme`. An absent `data-theme`
attribute means automatic mode; `data-theme="light"` or `data-theme="dark"` on
the root element forces a built-in scheme.

Use this pattern for automatic light-mode overrides:

```css
@media (prefers-color-scheme: light) {
  :root:not([data-theme]) {
    --color-bg-glass: rgba(255, 255, 255, 0.62);
  }
}
```

Use an attribute selector when a script or integration forces a scheme:

```css
:root[data-theme='light'] {
  --color-bg-glass: rgba(255, 255, 255, 0.62);
}

:root[data-theme='dark'] {
  --color-bg-glass: rgba(20, 24, 31, 0.64);
}
```

Place values shared by both schemes directly on `:root`.

## Theme properties

### Backgrounds and surfaces

| Property | Purpose |
| --- | --- |
| `--color-bg-canvas` | Page canvas fallback color |
| `--color-bg-surface` | Main content surface |
| `--color-bg-elevated` | Raised, non-glass content |
| `--color-bg-glass` | Translucent glass surface |
| `--color-bg-hover` | Pointer hover state |
| `--color-bg-focus` | Focused control state |
| `--color-bg-selected` | Selected item state |
| `--color-bg-invalid` | Invalid or rejected state |
| `--color-bg-table-header` | Table header surface |

### Text, fields, and separators

| Property | Purpose |
| --- | --- |
| `--color-text` | Primary text |
| `--color-text-muted` | Secondary text |
| `--color-text-disabled` | Disabled or unavailable content |
| `--color-border` | General separator |
| `--color-field-bg` | Input background |
| `--color-field-bg-disabled` | Disabled input background |
| `--color-field-border` | Input border |
| `--color-glass-border` | Glass edge highlight |

### Actions and status

`--color-accent`, `--color-success`, `--color-warning`, and
`--color-danger` are suitable for text and icons. Their `-strong` variants are
used for filled controls, with matching foreground properties:

```text
--color-accent-strong       --color-on-accent
--color-info-strong         --color-on-info
--color-success-strong      --color-on-success
--color-warning-strong      --color-on-warning
--color-danger-strong       --color-on-danger
```

Other state properties are:

```text
--color-overlay
--color-loading-overlay
--color-focus-ring
--color-progress-track
--color-progress-track-paused
--color-progress-value
```

### Shape, depth, glass, and motion

| Property | Purpose |
| --- | --- |
| `--radius-control` | Buttons and controls |
| `--radius-popover` | Dropdowns and menus |
| `--radius-dialog` | Dialog windows |
| `--radius-glass` | Large glass surfaces |
| `--radius-glass-compact` | Compact glass surfaces |
| `--backdrop-filter-field` | Input and editor field blur filter |
| `--backdrop-filter-glass` | Glass blur and saturation filter |
| `--shadow-control` | Hovered control shadow |
| `--shadow-elevated` | Dialog and popover shadow |
| `--motion-duration-fast` | Small interaction duration |
| `--motion-duration-normal` | Enter/leave duration |
| `--motion-easing-standard` | Standard motion curve |
| `--motion-easing-exit` | Exit motion curve |

Use `none` for `--backdrop-filter-field` or `--backdrop-filter-glass` to disable
the corresponding filter. A visible blur requires a translucent background
color and visual content behind the surface.

## Stable component selectors

The following attributes are the supported structural theme hooks.

| Selector | Target |
| --- | --- |
| `[data-ui="app-canvas"]` | Page canvas (`body`) |
| `[data-ui="app"]` | Mounted application |
| `[data-ui="app-header"]` | Global header |
| `[data-ui="button"]` | Application button or button-like navigation |
| `[data-ui="explorer"]` | File explorer |
| `[data-ui="entry-panel"]` | Main file-list surface |
| `[data-ui="readme"]` | README surface |
| `[data-ui="readme-content"]` | README content wrapper |
| `[data-ui="markdown"]` | Rendered Markdown |
| `[data-ui="path-bar"]` | Breadcrumb path |
| `[data-ui="path-segment"]` | Breadcrumb segment |
| `[data-ui="path-link"]` | Breadcrumb link |
| `[data-ui="search-panel"]` | Search surface |
| `[data-ui="dropdown"]` | Dropdown component |
| `[data-ui="dropdown-trigger"]` | Dropdown trigger |
| `[data-ui="dropdown-panel"]` | Floating dropdown panel |
| `[data-ui="menu"]` | Floating entry menu surface |
| `[data-ui="menu-content"]` | Entry menu contents |
| `[data-ui="menu-item"]` | Entry menu action |
| `[data-ui="dialog-overlay"]` | Dialog backdrop |
| `[data-ui="dialog"]` | Dialog surface |
| `[data-ui="dialog-header"]` | Dialog header |
| `[data-ui="dialog-body"]` | Dialog body |
| `[data-ui="dialog-footer"]` | Dialog actions |
| `[data-ui="admin-page"]` | Administration layout |
| `[data-ui="handler-header"]` | Preview or editor title bar |
| `[data-ui="preview"]` | Preview/editor root |
| `[data-ui="entry-drag-status"]` | Drag status popup |
| `[data-ui="drop-zone-indicator"]` | Full-screen upload drop indicator |
| `[data-ui="drop-zone"]` | Upload drop target |
| `[data-surface="glass"]` | Any built-in glass surface |

State and variant attributes refine these selectors:

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

## Ready-to-use presets

These presets support automatic light/dark mode and explicit
`data-theme="light"` / `"dark"`. Expand a preset and use the copy button in the
code block, then paste the complete stylesheet into **Global CSS**.

### Paper

Warm paper colors, ruled canvas lines, restrained radii, serif typography, ink
accents, and offset shadows create a layered stationery appearance.

::: details Show complete paper CSS

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

### Retro Terminal

Phosphor colors, monospaced text, scanlines, square panels, and crisp borders
turn the file manager into a restrained command-console display.

::: details Show complete retro terminal CSS

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

## Troubleshooting

- Inspect the element and prefer a documented custom property before adding a
  structural selector.
- Avoid `!important`; it is usually needed only when overriding an inline style
  or an unusually specific state.
- If glass looks opaque, reduce the alpha channel of `--color-bg-glass`.
- If the background remains sharp, confirm that the element has
  `data-surface="glass"` and that `--backdrop-filter-glass` is not `none`.
- A fixed background can cost additional rendering work on mobile browsers.
  Remove `background-attachment: fixed` if scrolling becomes uneven.
- Custom CSS can request external images and fonts. Only use asset URLs you
  trust.

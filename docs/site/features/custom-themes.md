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

## Complete glass theme example

This example adds a fixed decorative canvas, translucent surfaces, stronger
glass filtering, rounded controls, and a home marker:

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

For an explicitly forced light theme, repeat the light property block under
`:root[data-theme="light"]`.

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

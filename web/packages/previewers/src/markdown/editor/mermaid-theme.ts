import { isDarkMode, resolveCssColor } from '@go-drive/utils'
import type { MermaidConfig } from 'mermaid'

export function getMermaidTheme(
  scope: HTMLElement,
  isDark = isDarkMode(scope)
): MermaidConfig {
  const color = (value: string) => resolveCssColor(value, scope)
  const background = color('var(--color-bg-code-editor)')
  const text = color('var(--color-text)')
  const muted = color('var(--color-text-muted)')
  const accent = color('var(--color-accent)')
  const primary = color(
    'color-mix(in srgb, var(--color-accent) 12%, var(--color-bg-code-editor))'
  )
  const secondary = color('var(--color-bg-surface)')
  const tertiary = color('var(--color-bg-canvas)')

  return {
    theme: 'base',
    themeVariables: {
      darkMode: isDark,
      background,
      primaryColor: primary,
      primaryTextColor: text,
      primaryBorderColor: accent,
      secondaryColor: secondary,
      secondaryTextColor: text,
      secondaryBorderColor: muted,
      tertiaryColor: tertiary,
      tertiaryTextColor: text,
      tertiaryBorderColor: muted,
      lineColor: muted,
      textColor: text,
      edgeLabelBackground: background,
      noteBkgColor: primary,
      noteTextColor: text,
      noteBorderColor: accent,
      actorBkg: primary,
      actorTextColor: text,
      actorBorder: accent,
      actorLineColor: muted,
      signalColor: muted,
      signalTextColor: text,
      labelBoxBkgColor: secondary,
      labelBoxBorderColor: muted,
      labelTextColor: text,
      loopTextColor: text,
      activationBkgColor: secondary,
      activationBorderColor: accent,
    },
  }
}

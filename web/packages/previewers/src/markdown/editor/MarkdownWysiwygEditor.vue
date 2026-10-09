<template>
  <div
    ref="editorRoot"
    class="markdown-wysiwyg-editor"
    :class="{ 'is-readonly': disabled }"
  >
    <div v-if="failed" class="markdown-wysiwyg-editor__error">
      {{ t('preview.markdown.editor.error') }}
    </div>
  </div>
</template>
<script setup lang="ts">
import { languageDescriptions } from '@go-drive/code-mirror/languages'
import {
  LanguageDescription,
  LanguageSupport,
  StreamLanguage,
} from '@codemirror/language'
import { shift } from '@floating-ui/dom'
import { CrepeBuilder } from '@milkdown/crepe/builder'
import { blockEdit } from '@milkdown/crepe/feature/block-edit'
import { codeMirror } from '@milkdown/crepe/feature/code-mirror'
import { table } from '@milkdown/crepe/feature/table'
import { linkTooltip } from '@milkdown/crepe/feature/link-tooltip'
import { toolbar } from '@milkdown/crepe/feature/toolbar'
import { defaultUploader, uploadConfig } from '@milkdown/kit/plugin/upload'
import { blockService } from '@milkdown/kit/plugin/block'
import { replaceAll } from '@milkdown/kit/utils'
import { useI18n } from '@go-drive/i18n'
import { observeThemeChanges } from '@go-drive/utils'
import {
  onBeforeUnmount,
  onMounted,
  onWatcherCleanup,
  ref,
  shallowRef,
  watch,
} from 'vue'
import { installTouchControls } from './touch-controls'
import { createBlockService } from './block-controls'
import { getMermaidTheme } from './mermaid-theme'

import '@milkdown/crepe/theme/common/prosemirror.css'
import '@milkdown/crepe/theme/common/reset.css'
import '@milkdown/crepe/theme/common/code-mirror.css'
import '@milkdown/crepe/theme/common/block-edit.css'
import '@milkdown/crepe/theme/common/link-tooltip.css'
import '@milkdown/crepe/theme/common/table.css'
import '@milkdown/crepe/theme/common/toolbar.css'
import '@milkdown/crepe/theme/classic.css'

const props = withDefaults(
  defineProps<{
    modelValue?: string
    disabled?: boolean
  }>(),
  {
    modelValue: '',
    disabled: false,
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const { t } = useI18n({ useScope: 'global' })
const codeLanguages = [
  ...languageDescriptions,
  LanguageDescription.of({
    name: 'Mermaid',
    alias: ['mermaid'],
    extensions: ['mmd'],
    // Register Mermaid in the picker without syntax highlighting.
    support: new LanguageSupport(
      StreamLanguage.define({
        token: (stream) => {
          stream.skipToEnd()
          return null
        },
      })
    ),
  }),
]
const editorRoot = ref<HTMLElement | null>(null)
const failed = ref(false)
const ready = ref(false)
let disposed = false
let created = false
let builder: CrepeBuilder | undefined
let stopTouchControls: (() => void) | undefined
let stopThemeChanges: (() => void) | undefined
const mermaidTheme = shallowRef<ReturnType<typeof getMermaidTheme>>()
let mermaidModule: Promise<typeof import('mermaid').default> | undefined
let mermaidRenderId = 0

function loadMermaid() {
  mermaidModule ??= import('mermaid').then(({ default: mermaid }) => mermaid)
  return mermaidModule
}

async function renderMermaid(content: string) {
  const mermaid = await loadMermaid()
  mermaid.initialize({
    ...mermaidTheme.value,
    startOnLoad: false,
    securityLevel: 'strict',
    suppressErrorRendering: true,
  })
  // Mermaid's public render API already queues concurrent calls.
  const { svg } = await mermaid.render(
    `go-drive-mermaid-${++mermaidRenderId}`,
    content
  )
  return svg
}

onMounted(async () => {
  const updateTheme = () => {
    if (!editorRoot.value) return
    const theme = getMermaidTheme(editorRoot.value)
    if (JSON.stringify(theme) !== JSON.stringify(mermaidTheme.value)) {
      mermaidTheme.value = theme
    }
  }
  updateTheme()
  stopThemeChanges = observeThemeChanges(updateTheme)

  const editor = new CrepeBuilder({
    root: editorRoot.value,
    defaultValue: props.modelValue,
  })
    .addFeature(codeMirror, {
      languages: codeLanguages,
      previewLabel: t('preview.markdown.editor.mermaid_preview'),
      previewLoading: t('preview.markdown.editor.mermaid_loading'),
      previewToggleText: (previewOnlyMode) =>
        t(
          previewOnlyMode
            ? 'preview.markdown.editor.mermaid_show_code'
            : 'preview.markdown.editor.mermaid_hide_code'
        ),
      renderPreview: (language, content, applyPreview) => {
        if (language.trim().toLowerCase() !== 'mermaid' || !content.trim()) {
          return null
        }
        const stop = watch(
          mermaidTheme,
          () => {
            let active = true
            onWatcherCleanup(() => {
              active = false
            })
            void renderMermaid(content).then(
              (svg) => {
                if (active && !disposed) applyPreview(svg)
              },
              () => {
                if (!active || disposed) return
                const error = document.createElement('div')
                error.className = 'markdown-wysiwyg-editor__mermaid-error'
                error.textContent = t('preview.markdown.editor.mermaid_error')
                applyPreview(error)
              }
            )
          },
          { immediate: true }
        )
        // Milkdown's watcher owns this block's theme watcher and pending result.
        onWatcherCleanup(stop)
        return undefined
      },
    })
    .addFeature(linkTooltip)
    .addFeature(table)
    .addFeature(blockEdit, {
      textGroup: {
        label: t('preview.markdown.editor.text_blocks'),
        text: { label: t('preview.markdown.editor.paragraph') },
        h1: { label: t('preview.markdown.editor.heading_1') },
        h2: { label: t('preview.markdown.editor.heading_2') },
        h3: { label: t('preview.markdown.editor.heading_3') },
        h4: { label: t('preview.markdown.editor.heading_4') },
        h5: { label: t('preview.markdown.editor.heading_5') },
        h6: { label: t('preview.markdown.editor.heading_6') },
        quote: { label: t('preview.markdown.editor.quote') },
        divider: { label: t('preview.markdown.editor.divider') },
      },
      listGroup: {
        label: t('preview.markdown.editor.lists'),
        bulletList: { label: t('preview.markdown.editor.bullet_list') },
        orderedList: { label: t('preview.markdown.editor.ordered_list') },
        taskList: { label: t('preview.markdown.editor.task_list') },
      },
      advancedGroup: {
        label: t('preview.markdown.editor.advanced'),
        image: null,
        codeBlock: { label: t('preview.markdown.editor.code_block') },
        table: { label: t('preview.markdown.editor.table') },
        math: null,
      },
      blockHandle: {
        getOffset: () =>
          window.matchMedia('(max-width: 600px)').matches ? 6 : 16,
        getPosition: ({ active, editorDom }) => {
          const rect = active.el.getBoundingClientRect()
          // Lists indent their blocks; keep controls in the reserved outer gutter.
          const left =
            editorDom.getBoundingClientRect().left +
            parseFloat(getComputedStyle(editorDom).paddingLeft)
          return new DOMRect(left, rect.top, rect.right - left, rect.height)
        },
        middleware: [shift({ padding: 8, crossAxis: true })],
      },
      slashMenu: {
        middleware: [shift({ padding: 8, crossAxis: true })],
      },
    })
    .addFeature(toolbar, {
      boldLabel: t('preview.markdown.editor.bold'),
      italicLabel: t('preview.markdown.editor.italic'),
      strikethroughLabel: t('preview.markdown.editor.strikethrough'),
      codeLabel: t('preview.markdown.editor.inline_code'),
      linkLabel: t('preview.markdown.editor.link'),
    })

  editor.editor.config((ctx) => {
    ctx.set(blockService.key, createBlockService)
    ctx.update(uploadConfig.key, (value) => ({
      ...value,
      uploader: defaultUploader,
    }))
  })

  builder = editor
  editor.on((listener) => {
    listener.markdownUpdated((_ctx, markdown) => {
      if (ready.value) emit('update:modelValue', markdown)
    })
  })

  try {
    editor.setReadonly(props.disabled)
    await editor.create()
    created = true
    if (disposed) {
      await editor.destroy()
      return
    }
    if (editorRoot.value) {
      stopTouchControls = installTouchControls(
        editorRoot.value,
        () => props.disabled
      )
    }
    ready.value = true
  } catch {
    failed.value = true
  }
})

watch(
  () => props.modelValue,
  (value) => {
    if (!ready.value || !builder) return
    const markdown = value ?? ''
    if (builder.getMarkdown() !== markdown) {
      builder.editor.action(replaceAll(markdown))
    }
  }
)

watch(
  () => props.disabled,
  (value) => {
    builder?.setReadonly(value)
  }
)

onBeforeUnmount(() => {
  disposed = true
  stopTouchControls?.()
  stopThemeChanges?.()
  if (created) void builder?.destroy()
})
</script>
<style lang="scss">
.markdown-wysiwyg-editor {
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  overflow: auto;
  padding: 12px 0;
  background: var(--color-bg-elevated);

  .milkdown {
    height: 100%;
    width: 100%;
    min-height: 100%;
    --crepe-color-background: var(--color-bg-elevated);
    --crepe-color-on-background: var(--color-text);
    --crepe-color-surface: var(--color-bg-surface);
    --crepe-color-surface-low: var(--color-bg-hover);
    --crepe-color-on-surface: var(--color-text);
    --crepe-color-on-surface-variant: var(--color-text-muted);
    --crepe-color-outline: var(--color-border);
    --crepe-color-primary: var(--color-accent);
    --crepe-color-secondary: var(--color-bg-hover);
    --crepe-color-on-secondary: var(--color-text);
    --crepe-color-inverse: var(--color-text);
    --crepe-color-on-inverse: var(--color-bg-elevated);
    --crepe-color-inline-code: var(--color-danger);
    --crepe-color-error: var(--color-danger);
    --crepe-color-hover: var(--color-bg-hover);
    --crepe-color-selected: var(--color-bg-selected);
    --crepe-color-inline-area: var(--color-bg-hover);
    --crepe-shadow-1: var(--shadow-control);
    --crepe-shadow-2: var(--shadow-elevated);
  }

  .milkdown .ProseMirror {
    box-sizing: border-box;
    width: 100%;
    max-width: 960px;
    min-height: 100%;
    margin-inline: auto;
    // 10px outer margin + two 32px controls + 2px separator + 16px gap.
    padding: 28px 28px 28px 92px;
    outline: none;
  }

  &.is-readonly .milkdown .ProseMirror {
    padding-left: 28px;
  }

  .milkdown .milkdown-slash-menu {
    box-sizing: border-box;
    max-width: calc(100vw - 24px);
    max-height: min(60vh, 480px);
    max-height: min(60dvh, 480px);
    overflow: hidden;
  }

  .milkdown .milkdown-slash-menu > div {
    display: flex;
    flex-direction: column;
    min-height: 0;
    max-height: inherit;
  }

  .milkdown .milkdown-slash-menu .menu-groups {
    flex: 1 1 auto;
    min-height: 0;
    max-height: none;
    overflow-x: hidden;
    overflow-y: auto;
  }

  .milkdown .milkdown-slash-menu .menu-groups .menu-group li {
    min-width: 0;
  }

  .milkdown .milkdown-slash-menu .tab-group {
    padding: 8px 8px 0;
  }

  .milkdown .milkdown-slash-menu .tab-group ul {
    gap: 6px;
    padding: 4px 6px;
  }

  .milkdown .milkdown-slash-menu .tab-group ul li {
    padding: 4px 8px;
  }

  .milkdown .milkdown-slash-menu .menu-groups {
    padding: 0 8px 8px;
  }

  .milkdown .milkdown-slash-menu .menu-groups .menu-group h6 {
    padding: 8px 10px;
  }

  .milkdown .milkdown-slash-menu .menu-groups .menu-group li {
    gap: 12px;
    padding: 8px 10px;
  }

  .milkdown .milkdown-slash-menu .menu-groups .menu-group li svg {
    width: 20px;
    height: 20px;
  }

  .milkdown .milkdown-table-block .button-group svg {
    max-width: none;
  }

  .milkdown .milkdown-block-handle {
    // Position updates must follow scrolling immediately, without animating coordinates.
    transition: opacity 0.2s;
  }

  .milkdown .milkdown-code-block {
    --crepe-color-surface: var(--color-bg-code-editor);
  }

  .milkdown .milkdown-code-block .list-wrapper {
    background-color: var(--color-bg-elevated);
  }

  .milkdown .milkdown-code-block .tools .language-button {
    box-sizing: border-box;
    min-height: 24px;
    margin-bottom: 0;
    opacity: 1;
  }

  .milkdown .milkdown-code-block .tools .tools-button-group > button {
    opacity: 1;
  }

  &.is-readonly .milkdown .milkdown-code-block .tools .language-button {
    padding-right: 0;
    padding-left: 0;
    background: transparent;
    cursor: default;
    pointer-events: none;
  }

  &.is-readonly .milkdown .milkdown-code-block .tools .language-button:hover {
    background: transparent;
  }

  &.is-readonly
    .milkdown
    .milkdown-code-block
    .tools
    .language-button
    .expand-icon,
  &.is-readonly .milkdown .milkdown-code-block .language-picker {
    display: none !important;
  }

  &.is-readonly .milkdown .milkdown-block-handle {
    display: none !important;
  }

  &.is-readonly .milkdown .milkdown-table-block .handle {
    display: none !important;
  }

  .milkdown .milkdown-code-block .cm-editor {
    color: var(--color-text);

    .cm-content {
      caret-color: var(--color-text);
    }

    .cm-cursor,
    .cm-dropCursor {
      border-left-color: var(--color-text);
    }

    .cm-gutters,
    .cm-placeholder {
      color: var(--color-text-muted);
    }

    .cm-selectionBackground,
    &.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground {
      background-color: var(--color-bg-selected);
    }

    .cm-activeLine,
    .cm-activeLineGutter {
      background-color: var(--color-bg-focus);
    }

    .cm-selectionMatch,
    .cm-matchingBracket {
      background-color: var(--color-bg-selected);
    }

    .cm-nonmatchingBracket {
      background-color: var(--color-bg-invalid);
    }

    .cm-specialChar {
      color: var(--color-danger);
    }

    .cm-searchMatch {
      background-color: var(--color-bg-attention);
    }

    .cm-searchMatch-selected {
      background-color: var(--color-bg-selected);
    }

    .cm-searchMatch .cm-selectionMatch {
      background-color: transparent;
    }

    .cm-foldPlaceholder {
      color: var(--color-text-muted);
      background-color: var(--color-bg-hover);
      border-color: var(--color-border);
    }

    .cm-panels,
    .cm-tooltip {
      color: var(--color-text);
      background-color: var(--color-bg-elevated);
      border-color: var(--color-border);
    }

    .cm-tooltip-arrow::before {
      border-top-color: var(--color-border);
      border-bottom-color: var(--color-border);
    }

    .cm-tooltip-arrow::after {
      border-top-color: var(--color-bg-elevated);
      border-bottom-color: var(--color-bg-elevated);
    }

    .cm-tooltip-autocomplete > ul > li[aria-selected] {
      color: var(--color-text);
      background-color: var(--color-bg-selected);
    }

    .cm-textfield {
      color: var(--color-text);
      background-color: var(--color-field-bg);
      border-color: var(--color-field-border);
    }

    .cm-button {
      color: var(--color-text);
      background: var(--color-bg-hover);
      border-color: var(--color-border);
    }

    .cm-button:active {
      background: var(--color-bg-selected);
    }

    .cm-textfield:focus-visible,
    .cm-button:focus-visible {
      outline-color: var(--color-focus-ring);
    }

    &:not(.cm-focused) .cm-activeLine,
    &:not(.cm-focused) .cm-activeLineGutter {
      background-color: transparent;
    }
  }

  .preview-panel .preview {
    overflow-x: auto;
  }

  .milkdown .milkdown-code-block .codemirror-host.hidden + .preview-panel {
    margin-top: 12px;
  }

  .preview-panel .preview svg {
    max-width: 100%;
    height: auto;
  }

  @media (max-width: 600px) {
    .milkdown .ProseMirror {
      // 10px outer margin + 24px handle + 6px gap on the editable side.
      padding: 16px 10px 16px 40px;
    }

    &.is-readonly .milkdown .ProseMirror {
      padding-left: 10px;
    }

    .milkdown .ProseMirror ul {
      padding-left: 20px;
    }

    .milkdown .ProseMirror ol {
      padding-left: 24px;
    }

    .milkdown .milkdown-block-handle {
      width: 24px;
      flex-direction: column;
    }

    .milkdown .milkdown-block-handle .operation-item {
      width: 24px;
      height: 24px;
      padding: 2px;
    }

    .milkdown .milkdown-block-handle .operation-item svg {
      width: 20px;
      height: 20px;
    }

    .milkdown .milkdown-slash-menu {
      max-width: calc(100vw - 16px);
    }

    .milkdown .milkdown-slash-menu .tab-group ul {
      flex-wrap: wrap;
    }
  }

}

.markdown-wysiwyg-editor__error {
  padding: 16px;
  color: var(--color-danger);
}

.markdown-wysiwyg-editor__mermaid-error {
  padding: 12px;
  color: var(--color-danger);
}
</style>

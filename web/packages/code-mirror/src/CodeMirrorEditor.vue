<template>
  <div class="code-mirror-editor">
    <div ref="editorEl" class="code-mirror-editor__inner" />
    <div class="code-mirror-editor__languages">
      <select v-model="selectedLang" aria-label="Language">
        <option v-for="(_, language) in languages" :key="language" :value="language">
          {{ language }}
        </option>
      </select>
    </div>
  </div>
</template>
<script lang="ts">
export default { name: 'CodeMirrorEditor' }
</script>
<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { basicSetup, EditorView } from 'codemirror'
import { Compartment, EditorState, type Extension } from '@codemirror/state'
import { languages, getLang, getLangByFilename } from './languages'
import type { CodeMirrorTheme } from './types'
import themeLight from './theme-light'
import themeDark from './theme-dark'

const props = withDefaults(
  defineProps<{
    modelValue?: string
    filename?: string
    disabled?: boolean
    theme?: CodeMirrorTheme
  }>(),
  {
    modelValue: '',
    disabled: false,
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const editorEl = ref<HTMLDivElement | null>(null)
let editor: EditorView | undefined
let currentContent: string | undefined

const readOnlyCompartment = new Compartment()
const themeCompartment = new Compartment()
const langCompartment = new Compartment()
const selectedLang = ref<string>()

const prefersDark = () =>
  typeof window !== 'undefined' &&
  typeof window.matchMedia === 'function' &&
  window.matchMedia('(prefers-color-scheme: dark)').matches

const isDark = () => props.theme === 'dark' || (!props.theme && prefersDark())

const setEditorContent = (content: string) => {
  if (currentContent === content) return
  currentContent = content
  if (!editor) return
  editor.dispatch({
    changes: { from: 0, to: editor.state.doc.length, insert: content },
  })
}

const setTheme = () => {
  editor?.dispatch({
    effects: themeCompartment.reconfigure(isDark() ? themeDark : themeLight),
  })
}

const setLanguage = async () => {
  if (!editor) return
  let extension: Extension = []
  const language = selectedLang.value
  if (language) {
    try {
      extension = (await getLang(language)) ?? []
    } catch (error) {
      console.error('[CodeMirrorEditor] failed to load language', error)
    }
  }
  editor.dispatch({
    effects: langCompartment.reconfigure(extension),
  })
}

const setLanguageByFilename = () => {
  selectedLang.value = props.filename
    ? getLangByFilename(props.filename)
    : undefined
}

const initEditor = () => {
  editor = new EditorView({
    parent: editorEl.value!,
    extensions: [
      basicSetup,
      readOnlyCompartment.of(EditorState.readOnly.of(props.disabled)),
      themeCompartment.of([]),
      langCompartment.of([]),
      EditorView.updateListener.of((update) => {
        if (!update.docChanged) return
        currentContent = update.state.doc.toString()
        emit('update:modelValue', currentContent)
      }),
    ],
  })
}

const colorScheme =
  typeof window !== 'undefined' && typeof window.matchMedia === 'function'
    ? window.matchMedia('(prefers-color-scheme: dark)')
    : undefined

onMounted(() => {
  initEditor()
  setEditorContent(props.modelValue)
  setLanguageByFilename()
  setTheme()
  colorScheme?.addEventListener('change', setTheme)
})

onBeforeUnmount(() => {
  colorScheme?.removeEventListener('change', setTheme)
  editor?.destroy()
  editor = undefined
})

watch(
  () => props.disabled,
  (value) => {
    editor?.dispatch({
      effects: readOnlyCompartment.reconfigure(EditorState.readOnly.of(value)),
    })
  }
)
watch(() => props.theme, setTheme)
watch(selectedLang, setLanguage)
watch(() => props.filename, setLanguageByFilename)
watch(() => props.modelValue, (value) => setEditorContent(value ?? ''))
</script>
<style lang="scss">
.code-mirror-editor {
  position: relative;
  overflow: hidden;
  height: 100%;
}

.code-mirror-editor__languages {
  position: absolute;
  top: 10px;
  right: 10px;
  opacity: 0.4;

  &:hover {
    opacity: 1;
  }
}

.code-mirror-editor__languages select {
  background-color: var(--color-field-bg, #fff);
  color: var(--color-text, #222);
  border: solid 1px var(--color-field-border, #bbb);
  border-radius: var(--radius-control, 8px);
  padding: 4px 6px;
}

.code-mirror-editor__inner {
  overflow: hidden;
  height: 100%;

  .cm-editor {
    height: 100%;

    &.cm-focused {
      outline: none;
    }
  }

  .cm-scroller {
    overflow: auto;
    min-height: 300px;
  }
}
</style>

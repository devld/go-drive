<template>
  <div ref="editorEl" class="text-editor" />
</template>
<script lang="ts">
export default { name: 'TextEditor' }
</script>
<script setup lang="ts">
import { watch, onMounted, onBeforeUnmount, ref } from 'vue'
import {
  addPreferColorListener,
  isDarkMode,
  removePreferColorListener,
} from '@/utils/theme'

import { basicSetup, EditorView } from 'codemirror'
import { Compartment, EditorState } from '@codemirror/state'

import themeLight from './theme-light'
import themeDark from './theme-dark'

const props = defineProps({
  modelValue: {
    type: String,
  },
  filename: {
    type: String,
  },
  disabled: {
    type: Boolean,
  },
})

const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()

const editorEl = ref<HTMLDivElement | null>(null)

let editor: EditorView
const readOnlyCompartment = new Compartment()
const themeCompartment = new Compartment()

let currentContent: string

const setEditorContent = (content: string) => {
  if (currentContent === content) return
  currentContent = content
  if (editor) {
    editor.dispatch({
      changes: { from: 0, to: editor.state.doc.length, insert: content },
    })
  }
}

const prefersColorChanged = () => {
  const isDark = isDarkMode()

  editor.dispatch({
    effects: themeCompartment.reconfigure(isDark ? themeDark : themeLight),
  })
}

const initEditor = () => {
  editor = new EditorView({
    parent: editorEl.value!,
    extensions: [
      basicSetup,
      readOnlyCompartment.of(EditorState.readOnly.of(props.disabled)),
      themeCompartment.of([]),

      EditorView.updateListener.of(() => {
        currentContent = editor.state.doc.toString()
        emit('update:modelValue', currentContent)
      }),
    ],
  })
}

watch(
  () => props.disabled,
  (val) => {
    editor.dispatch({
      effects: readOnlyCompartment.reconfigure(EditorState.readOnly.of(val)),
    })
  }
)

onMounted(() => {
  initEditor()
  addPreferColorListener(prefersColorChanged)

  watch(
    () => props.modelValue,
    (val) => setEditorContent(val ?? ''),
    { immediate: true }
  )
  prefersColorChanged()
})
onBeforeUnmount(() => {
  removePreferColorListener(prefersColorChanged)
  editor.destroy()
})
</script>
<style lang="scss">
.text-editor {
  overflow: hidden;

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

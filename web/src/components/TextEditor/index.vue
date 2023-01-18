<template>
  <div class="text-editor">
    <div ref="editorEl" class="text-editor__inner" />
    <div class="text-editor__languages">
      <select v-model="selectedLang">
        <option v-for="(_, l) in languages" :key="l" :value="l">{{ l }}</option>
      </select>
    </div>
  </div>
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
import { getLang, getLangByEntry, languages } from './languages'

import { basicSetup, EditorView } from 'codemirror'
import { Compartment, EditorState, Extension } from '@codemirror/state'

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
const langCompartment = new Compartment()

let currentContent: string
const selectedLang = ref<string>()

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

const setLanguage = async () => {
  let extension: Extension = []
  let loadedLang: string | undefined
  if (selectedLang.value) {
    try {
      loadedLang = selectedLang.value
      const lang = await getLang(selectedLang.value)
      if (lang) extension = lang
    } catch (e: any) {
      console.error('[TextEditor] failed to load language', e)
    }
  }
  if (
    (Array.isArray(extension) && extension.length === 0) ||
    loadedLang === selectedLang.value
  ) {
    editor.dispatch({
      effects: langCompartment.reconfigure(extension),
    })
  }
}

const setLanageByFilename = () => {
  selectedLang.value = props.filename
    ? getLangByEntry(props.filename)
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

watch(selectedLang, setLanguage)
watch(() => props.filename, setLanageByFilename)
watch(
  () => props.modelValue,
  (val) => setEditorContent(val ?? '')
)

onMounted(() => {
  initEditor()
  addPreferColorListener(prefersColorChanged)
  setEditorContent(props.modelValue ?? '')
  setLanageByFilename()
  prefersColorChanged()
})
onBeforeUnmount(() => {
  removePreferColorListener(prefersColorChanged)
  editor.destroy()
})
</script>
<style lang="scss">
.text-editor {
  position: relative;
  overflow: hidden;
}

.text-editor__languages {
  position: absolute;
  top: 10px;
  right: 10px;
  opacity: 0.4;

  &:hover {
    opacity: 1;
  }
}

.text-editor__inner {
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

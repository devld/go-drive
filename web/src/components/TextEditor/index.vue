<template>
  <div ref="editorEl" class="text-editor" />
</template>
<script>
export default { name: 'TextEditor' }
</script>
<script setup>
import { watch, onMounted, onBeforeUnmount, ref } from 'vue'
import CodeMirror, { loadMode } from './codemirror'
import { filenameExt } from '@/utils'
import {
  addPreferColorListener,
  isDarkMode,
  removePreferColorListener,
} from '@/utils/theme'

function getThemeName() {
  return isDarkMode() ? 'material-darker' : 'github-light'
}

const props = defineProps({
  modelValue: {
    type: String,
  },
  filename: {
    type: String,
  },
  lineNumbers: {
    type: Boolean,
  },
  disabled: {
    type: Boolean,
  },
})

const emit = defineEmits(['update:modelValue'])

const editorEl = ref(null)
let editor
let currentContent

const setEditorContent = (content) => {
  if (currentContent === content) return
  currentContent = content
  if (editor) {
    editor.setValue(currentContent)
  }
}

const setEditorOption = (name, value) => {
  if (editor) editor.setOption(name, value)
}

const prefersColorChanged = () => setEditorOption('theme', getThemeName())

const initEditor = () => {
  editor = CodeMirror(editorEl.value, {
    theme: getThemeName(),
    value: currentContent || '',
    lineNumbers: props.lineNumbers,
    readOnly: props.disabled ? 'nocursor' : false,
  })
  setEditorMode()
  editor.on('change', () => {
    currentContent = editor.getValue()
    emit('update:modelValue', currentContent)
  })
}

const setEditorMode = async () => {
  const ext = filenameExt(props.filename)
  try {
    const mode = CodeMirror.findModeByExtension(ext)
    if (!mode) throw new Error(`mode ${mode.mode} not found`)
    await loadMode(mode)
    setEditorOption('mode', mode.mode)
  } catch (e) {
    console.warn(`[CodeMirror] failed to load language mode of '${ext}'`, e)
  }
}

watch(
  () => props.filename,
  (val) => {
    if (val) setEditorMode()
  }
)

watch(
  () => props.modelValue,
  (val) => setEditorContent(val),
  { immediate: true }
)
watch(
  () => props.lineNumbers,
  (val) => setEditorOption('lineNumbers', val)
)
watch(
  () => props.disabled,
  (val) => setEditorOption('readOnly', val ? 'nocursor' : false)
)

onMounted(() => {
  initEditor()
  addPreferColorListener(prefersColorChanged)
})
onBeforeUnmount(() => {
  removePreferColorListener(prefersColorChanged)
})
</script>
<style lang="scss">
@import url('codemirror/lib/codemirror.css');

@import url('codemirror-github-light/lib/codemirror-github-light-theme.css');
@import 'codemirror/theme/material-darker.css';

.text-editor {
  .CodeMirror {
    height: unset;
  }

  .CodeMirror-scroll {
    min-height: 300px;
  }
}
</style>

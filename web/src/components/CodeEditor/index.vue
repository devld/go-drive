<template>
  <div class="code-editor">
    <iframe ref="el" class="code-editor__inner"></iframe>
    <div v-if="loading" class="code-editor__loading">Editor loading...</div>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import {
  EditorOutMessageHandlers,
  JavaScriptSetupOptions,
} from '../../../monaco-editor/src/types'
import { getEnv } from './js-script-env'
import { useEditorSetup, useEditorTheme } from './utils'

const props = defineProps({
  modelValue: {
    type: String,
  },
  type: {
    type: String,
  },
  disabled: {
    type: Boolean,
  },
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'save'): void
}>()

const id = (Math.random() * 10000000).toFixed(0)

const loading = ref(false)
const el = ref<HTMLIFrameElement>()

let lastValue: string | undefined

const mode = computed(() => {
  if (!props.type) return
  const [lang, env] = props.type.split('-', 2)
  return { lang, env }
})
const language = computed(() => mode.value?.lang)

const url = computed(
  () =>
    `./code-editor/index.html?id=${id}&lang=${encodeURIComponent(
      mode.value?.lang ?? ''
    )}`
)

const initEditor = () => {
  loading.value = true
  el.value!.src = url.value
}
const prepareEditor = () => {
  let jsEnv: JavaScriptSetupOptions | undefined
  switch (language.value) {
    case 'javascript':
      jsEnv = getEnv(mode.value?.env)
      if (jsEnv) editorEmit('setupJs', jsEnv)
      break
  }
}

const messageHandlers: EditorOutMessageHandlers = {
  ready: () => {
    prepareEditor()
    setValue()
    setDisabled()
    setTheme()
    loading.value = false
  },
  change: (v) => {
    lastValue = v
    emit('update:modelValue', v)
  },
  save: () => emit('save'),
}

const [editorEmit] = useEditorSetup(id, el, messageHandlers)
const [setTheme] = useEditorTheme(editorEmit)

const setValue = () => {
  if (lastValue === props.modelValue) return
  editorEmit('setValue', props.modelValue ?? '')
}
const setDisabled = () => {
  editorEmit('setDisabled', props.disabled)
}

watch(() => props.modelValue, setValue)
watch(() => props.disabled, setDisabled)
watch(language, initEditor)
onMounted(initEditor)
</script>
<style lang="scss">
.code-editor {
  height: 0;
  min-height: 300px;
  position: relative;
}

.code-editor__inner {
  border: none;
  width: 100%;
  height: 100%;
}

.code-editor__loading {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  justify-content: center;
  align-items: center;
}
</style>

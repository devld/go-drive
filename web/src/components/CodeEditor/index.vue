<template>
  <div class="code-editor" :style="{ 'min-height': height }">
    <iframe ref="el" class="code-editor__inner"></iframe>
    <div v-if="typeSelectable" class="code-editor__languages">
      <select
        v-model="selectedLanguage"
        class="code-editor__language-select"
      >
        <option v-for="(_, l) in languages" :key="l" :value="l">{{ l }}</option>
      </select>
    </div>
    <LoadingState
      v-if="loading"
      variant="overlay"
      :text="$t('app.loading')"
    />
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { EditorOutMessageHandlers } from '../../../monaco-editor/src/types'
import { getEnv } from './js-script-env'
import { useEditorSetup, useEditorTheme } from './utils'
import { languages } from './mapping'
import { stringSplitN } from '@/utils'
import LoadingState from '@/components/LoadingState.vue'

const props = defineProps({
  modelValue: {
    type: String,
  },
  type: {
    type: String,
  },
  typeSelectable: {
    type: Boolean,
    default: true,
  },
  height: {
    type: String,
    default: '500px',
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
  const [lang, env] = stringSplitN(props.type, '-', 2)
  return { lang, env }
})
const selectedLanguage = ref<string>()
watch(
  mode,
  (m) => {
    selectedLanguage.value = m?.lang
  },
  { immediate: true }
)

const url = computed(
  () =>
    `./code-editor/index.html?id=${id}&lang=${encodeURIComponent(
      selectedLanguage.value ?? ''
    )}`
)

const initEditor = () => {
  loading.value = true
  el.value!.src = url.value
  lastValue = undefined
}
const prepareEditor = () => {
  switch (mode.value?.lang) {
    case 'javascript': {
      editorEmit('setupJs', (mode.value.env && getEnv(mode.value.env)) || {})
      break
    }
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
watch(selectedLanguage, initEditor)
onMounted(initEditor)
</script>
<style lang="scss">
.code-editor {
  height: 0;
  position: relative;
}

.code-editor__inner {
  border: none;
  width: 100%;
  height: 100%;
}

.code-editor__languages {
  position: absolute;
  top: 10px;
  right: 10px;
  opacity: 0.4;

  &:hover {
    opacity: 1;
  }
}

.code-editor__language-select {
  background-color: var(--color-field-bg);
  -webkit-backdrop-filter: var(--backdrop-filter-field);
  backdrop-filter: var(--backdrop-filter-field);
  border: solid 1px var(--color-field-border);
  border-radius: var(--radius-control);
  color: var(--color-text);
  outline: none;
  padding: 4px 6px;

  &:focus {
    border-color: var(--color-focus-ring);
    box-shadow: inset 0 0 0 1px var(--color-focus-ring);
  }

  option {
    background-color: var(--color-bg-elevated);
    color: var(--color-text);
  }
}

</style>

<template>
  <div ref="el" class="text-edit-view" @keydown="onKeyDown">
    <HandlerTitleBar :title="filename" @close="emit('close')">
      <template #actions>
        <SimpleButton v-if="!readonly" :loading="saving" @click="saveFile">
          {{ $t('hv.text_edit.save') }}
        </SimpleButton>
      </template>
    </HandlerTitleBar>
    <template v-if="!error">
      <CodeEditor
        v-if="useMonacoEditor"
        v-model="content"
        :type="monacoEditorType"
        :disabled="readonly"
        @save="!readonly && saveFile()"
      />
      <TextEditor
        v-else
        v-model="content"
        :filename="filename"
        :disabled="readonly"
      />
    </template>
    <ErrorView v-else :status="error.status" :message="error.message" />
    <div v-if="!inited" class="loading-tips">Loading...</div>
  </div>
</template>
<script setup lang="ts">
import { getContent } from '@/api'
import uploadManager from '@/api/upload-manager'
import CodeEditor from '@/components/CodeEditor/index.vue'
import { getLang } from '@/components/CodeEditor/mapping'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import TextEditor from '@/components/TextEditor/index.vue'
import { Entry } from '@/types'
import { entryMatches, filename as filenameFn, filenameExt } from '@/utils'
import { ApiError } from '@/utils/http'
import { alert } from '@/utils/ui-utils'
import { computed, nextTick, ref, watch } from 'vue'
import { EntryHandlerContext } from '../types'

const props = defineProps({
  entry: {
    type: Object as PropType<Entry>,
    required: true,
  },
  entries: { type: Array as PropType<Entry[]> },
  ctx: {
    type: Object as PropType<EntryHandlerContext>,
    required: true,
  },
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save-state', v: boolean): void
}>()

const error = ref<ApiError | null>(null)
const inited = ref(false)

const content = ref('')

const saving = ref(false)

const path = computed(() => props.entry.path)

const filename = computed(() => filenameFn(path.value))

const readonly = computed(() => !props.entry.meta.writable)

const el = ref<HTMLElement | null>(null)

const useMonacoEditor = computed(() => {
  const ext = props.ctx.config.options['web.monacoEditorExts']
  return ext && ext.length > 0 && entryMatches(props.entry, ext)
})
const monacoEditorType = computed(() => {
  const ext = filenameExt(filename.value)
  return getLang(ext)
})

const loadFile = async () => {
  inited.value = false
  try {
    return await loadFileContent()
  } catch (e: any) {
    error.value = e
  } finally {
    inited.value = true
  }
}

const loadFileContent = async () => {
  content.value = await getContent(path.value, props.entry.meta, {
    noCache: true,
  })
  nextTick(() => {
    changeSaveState(true)
  })
  return content.value
}

const saveFile = async () => {
  if (readonly.value) return
  if (saving.value) {
    return
  }
  saving.value = true
  try {
    await uploadManager.upload(
      {
        path: path.value,
        file: new Blob([content.value]),
        override: true,
      },
      true
    )
    changeSaveState(true)
  } catch (e: any) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}

const changeSaveState = (saved: boolean) => {
  emit('save-state', saved)
}

const onKeyDown = (e: KeyboardEvent) => {
  if (e.key === 's' && (e.ctrlKey || e.metaKey) && !readonly.value) {
    e.preventDefault()
    saveFile()
  }
}

watch(
  () => content.value,
  () => {
    changeSaveState(false)
  }
)

loadFile()
</script>
<style lang="scss">
.text-edit-view {
  position: relative;
  width: 100vw;
  height: 100%;
  padding-top: 48px;
  background-color: var(--secondary-bg-color);
  overflow: hidden;
  box-sizing: border-box;
  box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);

  .handler-title-bar {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
  }

  .text-editor {
    height: 100%;

    .CodeMirror {
      height: 100%;
    }
  }

  .code-editor {
    height: 100%;
  }

  .loading-tips {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    display: flex;
    justify-content: center;
    align-items: center;
    width: 100%;
    height: 300px;
    font-weight: bold;
    font-size: 24px;
    text-transform: uppercase;
    -webkit-user-select: none;
    user-select: none;
  }
}
</style>

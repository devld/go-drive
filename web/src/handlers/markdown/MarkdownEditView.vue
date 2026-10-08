<template>
  <div
    ref="el"
    class="markdown-edit-view"
    data-ui="preview"
    data-handler="markdown"
    @keydown="onKeyDown"
  >
    <HandlerTitleBar :title="filename" @close="emit('close')">
      <template #actions>
        <SimpleButton v-if="!readonly" :loading="saving" @click="saveFile">
          {{ $t('hv.text_edit.save') }}
        </SimpleButton>
      </template>
    </HandlerTitleBar>
    <MarkdownWysiwygEditor
      v-if="inited && !error"
      :key="path"
      v-model="content"
      :disabled="readonly"
    />
    <ErrorView
      v-else-if="error"
      :status="error.status"
      :message="error.message"
    />
    <LoadingState
      v-if="!inited"
      variant="overlay"
      :text="$t('app.loading')"
    />
  </div>
</template>
<script setup lang="ts">
import { getContent } from '@/api'
import uploadManager from '@/api/upload-manager'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import { Entry } from '@/types'
import { filename as filenameFn } from '@/utils'
import { HttpError } from '@/utils/http'
import { MarkdownWysiwygEditor } from '@go-drive/previewers/markdown/editor'
import { isPrimaryModifierPressed, LoadingState } from '@go-drive/utils'
import { alert } from '@/utils/ui-utils'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
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

const error = ref<HttpError | null>(null)
const inited = ref(false)
const content = ref('')
const saving = ref(false)
const path = computed(() => props.entry.path)
const filename = computed(() => filenameFn(path.value))
const readonly = computed(() => !props.entry.meta.writable)
const el = ref<HTMLElement | null>(null)

let loadId = 0
const loadFile = async () => {
  const id = ++loadId
  inited.value = false
  error.value = null
  content.value = ''
  try {
    const value = await getContent(path.value, props.entry.meta, {
      noCache: true,
    })
    if (id !== loadId) return
    content.value = value
    changeSaveState(true)
  } catch (e: any) {
    if (id === loadId) error.value = e
  } finally {
    if (id === loadId) inited.value = true
  }
}

const saveFile = async () => {
  if (readonly.value || saving.value) return
  const savePath = path.value
  const saveContent = content.value
  saving.value = true
  try {
    await uploadManager.upload(
      {
        path: savePath,
        file: new Blob([saveContent]),
        override: true,
      },
      true
    )
    if (path.value === savePath && content.value === saveContent) {
      changeSaveState(true)
    }
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
  if (
    e.key === 's' &&
    isPrimaryModifierPressed(e) &&
    !e.altKey &&
    !e.shiftKey &&
    !readonly.value
  ) {
    e.preventDefault()
    saveFile()
  }
}

watch(
  content,
  () => {
    if (inited.value) changeSaveState(false)
  },
  { flush: 'sync' }
)

watch(path, loadFile, { immediate: true })
onBeforeUnmount(() => {
  loadId++
})
</script>
<style lang="scss">
.markdown-edit-view {
  position: relative;
  width: 100vw;
  height: 100%;
  padding-top: 48px;
  background-color: var(--color-bg-elevated);
  overflow: hidden;
  box-sizing: border-box;
  box-shadow: var(--shadow-elevated);

  .handler-title-bar {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
  }

  > .loading-state {
    top: 48px;
  }
}
</style>

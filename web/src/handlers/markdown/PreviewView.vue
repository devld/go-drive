<template>
  <div
    ref="el"
    class="markdown-view"
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
    <MarkdownSplitView v-if="!error" :content="content">
      <template #editor>
        <CodeMirrorEditor
          v-model="content"
          :filename="filename"
          :disabled="readonly"
        />
      </template>
    </MarkdownSplitView>
    <ErrorView v-else :status="error.status" :message="error.message" />
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
import CodeMirrorEditor from '@go-drive/code-mirror'
import MarkdownSplitView from '@go-drive/previewers/markdown'
import { isPrimaryModifierPressed, LoadingState } from '@go-drive/utils'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import ErrorView from '@/components/ErrorView.vue'
import SimpleButton from '@/components/SimpleButton'
import type { Entry } from '@/types'
import { filename as filenameFn } from '@/utils'
import { HttpError } from '@/utils/http'
import { computed, nextTick, ref, watch } from 'vue'
import type { EntryHandlerContext } from '../types'

const props = defineProps({
  entry: {
    type: Object as PropType<Entry>,
    required: true,
  },
  ctx: {
    type: Object as PropType<EntryHandlerContext>,
    required: true,
  },
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save-state', value: boolean): void
}>()

const error = ref<HttpError | null>(null)
const inited = ref(false)
const content = ref('')
const saving = ref(false)
const el = ref<HTMLElement | null>(null)

const path = computed(() => props.entry.path)
const filename = computed(() => filenameFn(path.value))
const readonly = computed(() => !props.entry.meta.writable)

const loadFile = async () => {
  inited.value = false
  error.value = null
  try {
    content.value = await getContent(path.value, props.entry.meta, {
      noCache: true,
    })
    nextTick(() => changeSaveState(true))
  } catch (e: any) {
    error.value = e
  } finally {
    inited.value = true
  }
}

const saveFile = async () => {
  if (readonly.value || saving.value) return
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

const onKeyDown = (event: KeyboardEvent) => {
  if (
    event.key === 's' &&
    isPrimaryModifierPressed(event) &&
    !event.altKey &&
    !event.shiftKey &&
    !readonly.value
  ) {
    event.preventDefault()
    void saveFile()
  }
}

watch(
  () => content.value,
  () => changeSaveState(false)
)

loadFile()
</script>
<style lang="scss">
.markdown-view {
  position: relative;
  width: 100vw;
  height: 100%;
  padding-top: 48px;
  overflow: hidden;
  box-sizing: border-box;
  background-color: var(--color-bg-elevated);
  box-shadow: var(--shadow-elevated);

  .handler-title-bar {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
  }

  .markdown-split-view {
    height: 100%;
  }

  .code-mirror-editor {
    height: 100%;
  }

  > .loading-state {
    top: 48px;
  }
}
</style>

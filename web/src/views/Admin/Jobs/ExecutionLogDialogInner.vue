<template>
  <div ref="scroller" class="execution-log-dialog__inner">
    <div class="execution-log-dialog__content">
      {{ logContent }}
    </div>
    <div v-if="executing" class="execution-log-dialog__executing">
      <LoadingIndicator class="execution-log-dialog__executing-icon" />
    </div>
  </div>
</template>
<script setup lang="ts">
import {
  ref,
  nextTick,
  type PropType,
  onBeforeUnmount,
  onMounted,
  watch,
} from 'vue'
import { useI18n } from 'vue-i18n'
import { deleteTask, getTask } from '@/api'
import { s } from '@/i18n'
import type { ExecutionLogDialogOptions } from './execution-log-dialog'
import type { StreamHttpResponse } from '@/api/http'
import type { RequestTask } from '@/utils/http'
import type { BaseDialogOptionsData } from '@/utils/ui-utils/base-dialog'
import type { Task } from '@/types'
import LoadingIndicator from '@/components/LoadingIndicator.vue'
import { formatProgressPercent } from '@/utils'

const { t } = useI18n()

const props = defineProps({
  opts: {
    type: Object as PropType<ExecutionLogDialogOptions>,
    required: true,
  },
})

const emit = defineEmits<{
  (e: 'options', v: Partial<BaseDialogOptionsData>): void
}>()

const scroller = ref<HTMLDivElement>()

const logContent = ref('')
const executing = ref(false)
let requestTask: RequestTask<StreamHttpResponse<Task>> | undefined
let executionTask: Task | undefined
let aborted = false
let progressPollTimer: ReturnType<typeof setTimeout> | undefined
let progressPolling = false
const baseTitle = s(props.opts.title) || ''

const stopProgressPolling = () => {
  progressPolling = false
  if (progressPollTimer) clearTimeout(progressPollTimer)
  progressPollTimer = undefined
}

const pollProgress = async () => {
  if (!progressPolling || !executionTask) return
  try {
    const task = await getTask(executionTask.id)
    if (!progressPolling) return
    executionTask = task
    const progress = formatProgressPercent(task.progress)
    emit('options', {
      title: progress ? `${baseTitle} (${progress})` : baseTitle,
    })
    if (
      progressPolling &&
      (task.status === 'pending' || task.status === 'running')
    ) {
      progressPollTimer = setTimeout(pollProgress, 1000)
    }
  } catch {
    // The streaming request remains authoritative if polling briefly fails.
    if (progressPolling) progressPollTimer = setTimeout(pollProgress, 1000)
  }
}

const emitExecuting = () => {
  emit('options', {
    confirmText: t('p.admin.jobs.abort'),
    confirmType: 'danger',
  })
  executing.value = true
}

const emitNormal = () => {
  stopProgressPolling()
  emit('options', {
    title: baseTitle,
    confirmText: '',
    confirmType: undefined,
  })
  executing.value = false
}

const scrollToBottom = async () => {
  await nextTick()
  scroller.value?.scrollTo({
    top: scroller.value.scrollHeight,
    behavior: 'smooth',
  })
}

const doExecute = async () => {
  emitExecuting()

  try {
    requestTask = props.opts.execute()
    const resp = await requestTask
    if (resp.status !== 200) {
      throw new Error(`Request failed with status: ${resp.status}`)
    }
    executionTask = resp.data
    progressPolling = true
    void pollProgress()
    const reader = resp.stream.getReader()
    if (!reader) throw new Error('reader is undefined')

    const textDecoder = new TextDecoder()
    // eslint-disable-next-line no-constant-condition
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      logContent.value += textDecoder.decode(value)
    }
    logContent.value += textDecoder.decode()
  } catch (e: any) {
    if (!aborted) logContent.value += '\nError' + (e?.message || '') + '\n'
  } finally {
    emitNormal()
  }
}

onMounted(() => {
  doExecute()
})

onBeforeUnmount(stopProgressPolling)

defineExpose({
  beforeConfirm: async () => {
    if (!executing.value) return
    aborted = true
    if (requestTask) requestTask.cancel()
    if (executionTask) await deleteTask(executionTask.id)
    return false
  },
  beforeCancel: async () => {
    if (!executing.value) return
    aborted = true
    if (requestTask) requestTask.cancel()
  },
})

watch(logContent, scrollToBottom)
</script>
<style lang="scss">
.execution-log-dialog {
  &__inner {
    width: 500px;
    max-width: 90vw;
    height: 260px;
    max-height: 80vh;
    text-align: left;
    overflow: auto;
  }

  &__content {
    width: 100%;
    font-size: 12px;
    outline: none;
    border: 0;
    resize: none;
    white-space: pre;
  }

  &__executing-icon {
    width: 1em;
    height: 1em;
    color: var(--color-text-muted);
  }
}
</style>

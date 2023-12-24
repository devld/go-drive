<template>
  <div ref="scroller" class="execution-log-dialog__inner">
    <div class="execution-log-dialog__content">
      {{ logContent }}
    </div>
    <div v-if="executing" class="execution-log-dialog__executing">
      <Icon class="execution-log-dialog__executing-icon" svg="#icon-loading" />
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, nextTick, type PropType, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { deleteTask } from '@/api'
import type { ExecutionLogDialogOptions } from './execution-log-dialog'
import type { StreamHttpResponse } from '@/api/http'
import type { RequestTask } from '@/utils/http'
import type { BaseDialogOptionsData } from '@/utils/ui-utils/base-dialog'
import type { Task } from '@/types'

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

const emitExecuting = () => {
  emit('options', {
    confirmText: t('p.admin.jobs.abort'),
    confirmType: 'danger',
  })
  executing.value = true
}

const emitNormal = () => {
  emit('options', {
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
    logContent.value += '\nError' + (e?.message || '') + '\n'
  } finally {
    emitNormal()
  }
}

onMounted(() => {
  doExecute()
})

defineExpose({
  beforeConfirm: async () => {
    if (!executing.value) return
    if (requestTask) requestTask.cancel()
    if (executionTask) deleteTask(executionTask.id)
  },
  beforeCancel: async () => {
    if (!executing.value) return
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
    animation: spinning 1s linear infinite;
    color: var(--secondary-text-color);
  }
}
</style>

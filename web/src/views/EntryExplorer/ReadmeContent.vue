<template>
  <div v-if="readmeLoading || readmeContent" data-ui="readme-content">
    <LoadingState
      v-if="readmeLoading"
      variant="panel"
      :surface="false"
      :text="$t('p.home.readme_loading')"
    />
    <div
      v-else
      v-markdown="readmeContent"
      class="markdown-body"
      data-ui="markdown"
    ></div>
  </div>
</template>
<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { RequestTask } from '@/utils/http'
import { Entry } from '@/types'
import { getContent } from '@/api'
import { dir } from '@/utils'
import { useI18n } from 'vue-i18n'
import LoadingState from '@/components/LoadingState.vue'

const README_FILENAME = 'readme.md'

const { t } = useI18n()

const props = defineProps({
  path: {
    type: String,
    required: true,
  },
  entries: {
    type: Array as PropType<Entry[]>,
  },
})

const readmeEntry = computed<Entry | undefined>(() => {
  if (!props.entries) return
  let readmeFound
  for (const e of props.entries) {
    if (e.type !== 'file') continue
    if (README_FILENAME.toLowerCase() === e.name.toLowerCase()) {
      readmeFound = e
      break
    }
  }
  return readmeFound
})

let readmeTask: RequestTask<any> | undefined

const readmeContent = ref('')
const readmeLoading = ref(false)

const loadReadme = async (entry: Entry) => {
  readmeTask?.cancel()
  const task = getContent(entry.path, entry.meta)
  readmeTask = task

  let content
  readmeLoading.value = true
  try {
    content = await task
  } catch (e: any) {
    if (e.isCancel) return
    content = `<p style="text-align: center;">${t('p.home.readme_failed')}</p>`
  } finally {
    if (readmeTask === task) {
      readmeTask = undefined
      readmeLoading.value = false
    }
  }
  if (props.path === dir(entry.path)) {
    readmeContent.value = content
  }
}

watch(
  () => props.entries,
  () => {
    if (readmeEntry.value) {
      loadReadme(readmeEntry.value)
    } else {
      readmeContent.value = ''
      readmeLoading.value = false
      readmeTask?.cancel()
      readmeTask = undefined
    }
  }
)
</script>

<template>
  <div v-if="readmeContent">
    <div v-markdown="readmeContent" class="markdown-body"></div>
  </div>
</template>
<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { RequestTask } from '@/utils/http'
import { Entry } from '@/types'
import { getContent } from '@/api'
import { dir } from '@/utils'
import { useI18n } from 'vue-i18n'

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

let readmeTask: RequestTask | undefined

const readmeContent = ref('')

const loadReadme = async (entry: Entry) => {
  readmeTask?.cancel()
  readmeTask = getContent(entry.path, entry.meta)

  let content
  readmeContent.value = `<p style="text-align: center">${t(
    'p.home.readme_loading'
  )}</p>`
  try {
    content = await readmeTask
  } catch (e: any) {
    if (e.isCancel) return
    content = `<p style="text-align: center;">${t('p.home.readme_failed')}</p>`
  } finally {
    readmeTask = undefined
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
      readmeTask?.cancel()
      readmeTask = undefined
    }
  }
)
</script>

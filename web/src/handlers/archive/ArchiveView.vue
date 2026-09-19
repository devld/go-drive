<template>
  <div
    class="archive-view"
    data-ui="preview"
    data-handler="archive"
  >
    <HandlerTitleBar :title="entry.name" @close="emit('close')">
      <template #actions>
        <SimpleButton
          v-if="currentDir"
          variant="plain"
          small
          @click="goParent"
        >
          {{ $t('handler.archive.parent') }}
        </SimpleButton>
      </template>
    </HandlerTitleBar>

    <div class="archive-view__body">
      <div class="archive-view__location" :title="currentDir || '/'">
        {{ currentDir || '/' }}
      </div>

      <LoadingState v-if="loading" variant="panel">
        <progress
          v-if="loadingProgress.total > 0"
          class="archive-view__progress"
          :value="loadingProgress.loaded"
          :max="loadingProgress.total"
        />
        <span v-if="loadingPercent !== undefined">
          {{ loadingPercent }}%
        </span>
      </LoadingState>
      <ErrorView
        v-else-if="error"
        :status="error.status"
        :message="error.message"
      />
      <div v-else class="archive-view__content">
        <ErrorView
          v-if="downloadError"
          :status="downloadError.status"
          :message="downloadError.message"
        />
        <div class="archive-view__entries">
          <div
            v-for="item in items"
            :key="item.path"
            class="archive-view__entry"
          >
            <button
              v-if="item.type === 'dir'"
              class="archive-view__entry-main"
              type="button"
              @click="openDir(item)"
            >
              <span class="archive-view__entry-icon" aria-hidden="true">
                📁
              </span>
              <span class="archive-view__entry-name">{{ item.name }}</span>
            </button>
            <div v-else class="archive-view__entry-main">
              <span class="archive-view__entry-icon" aria-hidden="true">
                📄
              </span>
              <span class="archive-view__entry-name">{{ item.name }}</span>
              <span class="archive-view__entry-size">
                {{ formatBytes(item.size) }}
              </span>
            </div>
            <SimpleButton
              v-if="item.type === 'file'"
              class="archive-view__download"
              variant="plain"
              small
              icon="download"
              :loading="downloadingPath === item.path"
              :title="$t('handler.archive.download')"
              :aria-label="$t('handler.archive.download')"
              @click="download(item)"
            />
          </div>
          <span v-if="items.length === 0" class="archive-view__empty">
            {{ $t('handler.archive.empty') }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  ArchiveEntry,
  ArtifactInfo,
  artifactUrl,
  getArchiveIndex,
  prepareArtifact,
} from '@/api/artifact'
import ErrorView from '@/components/ErrorView.vue'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import { Entry, TaskProgress } from '@/types'
import { formatBytes, TASK_CANCELLED, taskDone } from '@/utils'
import { LoadingState } from '@go-drive/utils'
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

const emit = defineEmits<{ (e: 'close'): void }>()

const currentDir = ref('')
const allItems = ref<ArchiveEntry[]>([])
const loading = ref(false)
const error = ref<any>()
const downloadError = ref<any>()
const downloadingPath = ref('')
const loadingProgress = ref<TaskProgress>({ loaded: 0, total: 0 })
const loadingPercent = computed(() => {
  const { loaded, total } = loadingProgress.value
  if (total <= 0) return undefined
  return Math.min(100, Math.round((loaded / total) * 100))
})
let loadRequest = 0
let downloadRequest = 0

const items = computed(() => {
  const result = allItems.value.filter((item) => {
    const index = item.path.lastIndexOf('/')
    const parent = index < 0 ? '' : item.path.substring(0, index)
    return parent === currentDir.value
  })
  return result.sort((a, b) => {
    if (a.type !== b.type) return a.type === 'dir' ? -1 : 1
    return a.name.localeCompare(b.name)
  })
})

const waitReady = async (
  type: string,
  args: string | undefined,
  request: number,
  current: () => number,
  onProgress?: (progress: TaskProgress) => void
) => {
  const prepared = await prepareArtifact(
    props.entry.path,
    props.entry.meta,
    type,
    args
  )
  if ('info' in prepared) {
    return
  }
  await taskDone<ArtifactInfo>(prepared.task, (task) => {
    if (request !== current()) return false
    onProgress?.(task.progress ?? { loaded: 0, total: 0 })
  })
}

const load = async () => {
  const request = ++loadRequest
  loading.value = true
  error.value = undefined
  downloadError.value = undefined
  downloadRequest++
  downloadingPath.value = ''
  loadingProgress.value = { loaded: 0, total: 0 }
  try {
    await waitReady('archive-index', undefined, request, () => loadRequest, (progress) => {
      loadingProgress.value = progress
    })
    if (request !== loadRequest) return
    const result = await getArchiveIndex(props.entry.path, props.entry.meta)
    if (request === loadRequest) allItems.value = result ?? []
  } catch (e: any) {
    if (request === loadRequest && e !== TASK_CANCELLED) error.value = e
  } finally {
    if (request === loadRequest) loading.value = false
  }
}

const openDir = (item: ArchiveEntry) => {
  if (item.type !== 'dir') return
  currentDir.value = item.path
  downloadError.value = undefined
}

const download = async (item: ArchiveEntry) => {
  const request = ++downloadRequest
  downloadingPath.value = item.path
  downloadError.value = undefined
  try {
    await waitReady('archive-content', item.path, request, () => downloadRequest)
    if (request !== downloadRequest) return
    const link = document.createElement('a')
    link.href = artifactUrl(
      props.entry.path,
      props.entry.meta,
      'archive-content',
      item.path
    )
    link.download = item.name
    link.target = '_blank'
    link.rel = 'noreferrer noopener nofollow'
    link.click()
  } catch (e: any) {
    if (request === downloadRequest && e !== TASK_CANCELLED) {
      downloadError.value = e
    }
  } finally {
    if (request === downloadRequest) downloadingPath.value = ''
  }
}

const goParent = () => {
  const index = currentDir.value.lastIndexOf('/')
  currentDir.value = index < 0 ? '' : currentDir.value.substring(0, index)
}

watch(
  () => props.entry.path,
  () => {
    if (!props.entry.path) return
    currentDir.value = ''
    allItems.value = []
    load()
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  // Stop polling when the view closes, but leave the server task running so
  // its index can still populate the shared cache.
  loadRequest++
  downloadRequest++
})
</script>

<style lang="scss">
.archive-view {
  position: relative;
  display: flex;
  flex-direction: column;
  width: min(520px, calc(100vw - 48px));
  height: min(520px, calc(100vh - 96px));
  overflow: hidden;
  background-color: var(--color-bg-elevated);
}

.archive-view__body {
  display: flex;
  flex: 1;
  min-height: 0;
  flex-direction: column;
}

.archive-view__location {
  flex: none;
  padding: 8px 16px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-text-muted);
  border-bottom: 1px solid var(--color-border);
}

.archive-view__progress {
  width: min(260px, 70vw);
}

.archive-view__content {
  display: flex;
  flex: 1;
  min-height: 0;
  flex-direction: column;
}

.archive-view__entries {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 8px;
}

.archive-view__entry {
  display: flex;
  align-items: center;
  gap: 4px;
  border-radius: var(--radius-control);

  &:hover {
    background: var(--color-bg-hover);
  }
}

.archive-view__entry-main {
  display: flex;
  flex: 1;
  min-width: 0;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  color: var(--color-text);
  text-align: left;
  border: 0;
  background: transparent;
}

button.archive-view__entry-main {
  cursor: pointer;
}

.archive-view__entry-icon {
  flex: none;
}

.archive-view__entry-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.archive-view__entry-size {
  margin-left: auto;
  color: var(--color-text-muted);
  font-size: 12px;
}

.archive-view__download.simple-button {
  flex: none;
  margin-right: 4px;
}

.archive-view__empty {
  display: block;
  padding: 24px 10px;
  color: var(--color-text-muted);
  text-align: center;
}
</style>

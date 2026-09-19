<template>
  <div
    class="archive-view glass-surface"
    data-ui="preview"
    data-handler="archive"
    data-surface="glass"
  >
    <HandlerTitleBar :title="entry.name" @close="emit('close')" />

    <div class="archive-view__body">
      <PathBar
        class="archive-view__location"
        :path="currentDir"
        root-name="/"
        @update:path="onPathChange"
      />

      <LoadingState
        v-if="loading"
        class="archive-view__status"
        variant="panel"
        :surface="false"
      >
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
        class="archive-view__status"
        :status="error.status"
        :message="error.message"
        :show-back="false"
      />
      <div v-else class="archive-view__content">
        <div class="archive-view__entries">
          <div
            v-for="item in displayItems"
            :key="item.name === '..' ? '..' : item.path"
            class="archive-view__entry"
          >
            <button
              v-if="item.type === 'dir'"
              class="archive-view__entry-main"
              type="button"
              @click="openDir(item)"
            >
              <EntryIcon
                class="archive-view__entry-icon"
                :entry="toIconEntry(item)"
                :show-thumbnail="false"
              />
              <span class="archive-view__entry-name">{{ item.name }}</span>
            </button>
            <div v-else class="archive-view__entry-main">
              <EntryIcon
                class="archive-view__entry-icon"
                :entry="toIconEntry(item)"
                :show-thumbnail="false"
              />
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
  ARCHIVE_ARGS_INDEX,
  ARTIFACT_ARCHIVE,
  archiveContentArgs,
  ArchiveEntry,
  ArtifactInfo,
  artifactUrl,
  getArchiveIndex,
  prepareArtifact,
} from '@/api/artifact'
import type { EntryEventData } from '@/components/entry'
import ErrorView from '@/components/ErrorView.vue'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import { Entry, TaskProgress } from '@/types'
import { formatBytes, TASK_CANCELLED, taskDone } from '@/utils'
import { alert } from '@/utils/ui-utils'
import { LoadingState } from '@go-drive/utils'
import { computed, onBeforeUnmount, ref, shallowRef, watch } from 'vue'
import { EntryHandlerContext } from '../types'
import { buildArchiveTree } from './tree'

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
const childrenByParent = shallowRef<Map<string, ArchiveEntry[]>>(new Map())
const loading = ref(false)
const error = ref<any>()
const downloadingPath = ref('')
const loadingProgress = ref<TaskProgress>({ loaded: 0, total: 0 })
const loadingPercent = computed(() => {
  const { loaded, total } = loadingProgress.value
  if (total <= 0) return undefined
  return Math.min(100, Math.round((loaded / total) * 100))
})
let loadRequest = 0
let downloadRequest = 0

const items = computed(
  () => childrenByParent.value.get(currentDir.value) ?? []
)

const parentItem = computed<ArchiveEntry | undefined>(() => {
  if (!currentDir.value) return undefined
  const index = currentDir.value.lastIndexOf('/')
  return {
    path: index < 0 ? '' : currentDir.value.substring(0, index),
    name: '..',
    type: 'dir',
    size: -1,
    modTime: -1,
  }
})

const displayItems = computed(() =>
  parentItem.value ? [parentItem.value, ...items.value] : items.value
)

const toIconEntry = (item: ArchiveEntry): Entry => ({
  type: item.type,
  name: item.name,
  path: item.path,
  size: item.size,
  modTime: item.modTime,
  meta: {},
})

const waitReady = async (
  args: string | undefined,
  request: number,
  current: () => number,
  onProgress?: (progress: TaskProgress) => void
) => {
  const prepared = await prepareArtifact(
    props.entry.path,
    props.entry.meta,
    ARTIFACT_ARCHIVE,
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
  downloadRequest++
  downloadingPath.value = ''
  loadingProgress.value = { loaded: 0, total: 0 }
  try {
    await waitReady(ARCHIVE_ARGS_INDEX, request, () => loadRequest, (progress) => {
      loadingProgress.value = progress
    })
    if (request !== loadRequest) return
    const result = await getArchiveIndex(props.entry.path, props.entry.meta)
    if (request === loadRequest) {
      childrenByParent.value = buildArchiveTree(result ?? [])
    }
  } catch (e: any) {
    if (request === loadRequest && e !== TASK_CANCELLED) error.value = e
  } finally {
    if (request === loadRequest) loading.value = false
  }
}

const openDir = (item: ArchiveEntry) => {
  if (item.type !== 'dir') return
  currentDir.value = item.path
}

const onPathChange = ({ path, event }: EntryEventData) => {
  event?.preventDefault()
  if (typeof path !== 'string') return
  currentDir.value = path
}

const download = async (item: ArchiveEntry) => {
  const request = ++downloadRequest
  downloadingPath.value = item.path
  try {
    await waitReady(archiveContentArgs(item.path), request, () => downloadRequest)
    if (request !== downloadRequest) return
    const link = document.createElement('a')
    link.href = artifactUrl(
      props.entry.path,
      props.entry.meta,
      ARTIFACT_ARCHIVE,
      archiveContentArgs(item.path)
    )
    link.download = item.name
    link.target = '_blank'
    link.rel = 'noreferrer noopener nofollow'
    link.click()
  } catch (e: any) {
    if (request === downloadRequest && e !== TASK_CANCELLED) {
      alert(e.message)
    }
  } finally {
    if (request === downloadRequest) downloadingPath.value = ''
  }
}

watch(
  () => props.entry.path,
  () => {
    if (!props.entry.path) return
    currentDir.value = ''
    childrenByParent.value = new Map()
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
  background-color: var(--color-bg-glass);
  box-shadow: var(--shadow-elevated);
}

.archive-view__body {
  display: flex;
  flex: 1;
  min-height: 0;
  flex-direction: column;
}

.archive-view__location.path-bar {
  flex: none;
  padding: 8px 16px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--color-text-muted);
  border-bottom: 1px solid var(--color-border);

  .path-bar__path {
    padding: 2px 4px;

    &:hover {
      background: var(--color-bg-hover);
    }
  }

  .path-bar__segment:last-child .path-bar__path {
    color: var(--color-text);
  }
}

.archive-view__progress {
  width: min(260px, 70vw);
}

.archive-view__status.loading-state,
.archive-view__status.error-view {
  flex: 1;
  min-height: 0;
}

.archive-view__status.error-view {
  display: flex;
  flex-direction: column;
  justify-content: center;
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
  font: inherit;
  color: var(--color-text);
  text-align: left;
  border: 0;
  background: transparent;
}

button.archive-view__entry-main {
  cursor: pointer;
}

.archive-view__entry-icon.entry-icon {
  flex: none;
  width: 28px;
  height: 28px;
  border-radius: 6px;
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

.archive-view__download.simple-button.plain.small {
  flex: none;
  margin-right: 4px;
  font-size: 22px;
}

.archive-view__empty {
  display: block;
  padding: 24px 10px;
  color: var(--color-text-muted);
  text-align: center;
}
</style>

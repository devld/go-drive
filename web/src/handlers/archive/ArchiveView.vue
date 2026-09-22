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
            <label
              v-if="item.name !== '..'"
              class="archive-view__check"
            >
              <input
                type="checkbox"
                :checked="isChecked(item)"
                :indeterminate.prop="isIndeterminate(item)"
                @click.stop
                @change="toggleItem(item)"
              />
            </label>
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
        <div v-if="selectedCount > 0" class="archive-view__footer">
          <span>{{ $t('handler.archive.n_selected', { n: selectedCount }) }}</span>
          <span class="archive-view__actions">
            <SimpleButton
              small
              icon="download"
              :loading="packaging"
              @click="downloadSelected"
            >
              {{ $t('handler.archive.download') }}
            </SimpleButton>
            <SimpleButton
              small
              icon="copy"
              :loading="extracting"
              @click="extractTo"
            >
              {{ $t('handler.archive.extract_to') }}
            </SimpleButton>
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
  archivePackArgs,
  ArchiveEntry,
  ArtifactInfo,
  artifactRefUrl,
  getArchiveIndex,
  prepareArtifact,
} from '@/api/artifact'
import { deleteTask, extractArchive } from '@/api'
import type { EntryEventData } from '@/components/entry'
import ErrorView from '@/components/ErrorView.vue'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import { Entry, Task, TaskProgress } from '@/types'
import { dir, formatBytes, TASK_CANCELLED, taskDone } from '@/utils'
import { alert, loading as showTaskLoading, open } from '@/utils/ui-utils'
import { T } from '@go-drive/i18n'
import { LoadingState } from '@go-drive/utils'
import { computed, onBeforeUnmount, ref, shallowRef, watch } from 'vue'
import { EntryHandlerContext } from '../types'
import {
  buildArchiveTree,
  hasArchiveDescendantSelected,
  isArchiveDirFullySelected,
  isArchivePathCovered,
  mergeArchiveSelection,
  toggledArchiveSelection,
} from './tree'

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

const emit = defineEmits<{ (e: 'close'): void; (e: 'refresh'): void }>()

const currentDir = ref('')
const childrenByParent = shallowRef<Map<string, ArchiveEntry[]>>(new Map())
const selected = ref<Set<string>>(new Set())
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

const selectedMembers = computed(() =>
  mergeArchiveSelection(selected.value, childrenByParent.value)
)
const selectedCount = computed(() => selectedMembers.value.length)
const extracting = ref(false)
const packaging = ref(false)

const isChecked = (item: ArchiveEntry) =>
  item.type === 'dir'
    ? isArchiveDirFullySelected(
        item.path,
        selected.value,
        childrenByParent.value
      )
    : isArchivePathCovered(item.path, selected.value)

const isIndeterminate = (item: ArchiveEntry) =>
  item.type === 'dir' &&
  !isChecked(item) &&
  hasArchiveDescendantSelected(
    item.path,
    selected.value,
    childrenByParent.value
  )

const toggleItem = (item: ArchiveEntry) => {
  selected.value = toggledArchiveSelection(
    item,
    selected.value,
    childrenByParent.value
  )
}

const toIconEntry = (item: ArchiveEntry): Entry => ({
  type: item.type,
  name: item.name,
  path: item.path,
  size: item.size,
  modTime: item.modTime,
  meta: {},
})

const waitReady = async (
  args: string,
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
    return prepared.info
  }
  return taskDone<ArtifactInfo>(prepared.task, (task) => {
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
    const info = await waitReady(ARCHIVE_ARGS_INDEX, request, () => loadRequest, (progress) => {
      loadingProgress.value = progress
    })
    if (request !== loadRequest) return
    if (!info || !info.ref) return
    const result = await getArchiveIndex(props.entry.path, props.entry.meta, info.ref)
    if (request === loadRequest) {
      childrenByParent.value = buildArchiveTree(result ?? [])
      selected.value = new Set()
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
    const info = await waitReady(archiveContentArgs(item.path), request, () => downloadRequest)
    if (request !== downloadRequest || !info || !info.ref) return
    const link = document.createElement('a')
    link.href = artifactRefUrl(
      props.entry.path,
      props.entry.meta,
      ARTIFACT_ARCHIVE,
      info.ref
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

const downloadSelected = async () => {
  const members = selectedMembers.value
  if (members.length === 0 || packaging.value) return
  packaging.value = true
  let canceled = false
  let task: Task<ArtifactInfo> | undefined
  const onCancel = () => {
    canceled = true
    return task && deleteTask(task.id)
  }
  try {
    showTaskLoading({
      text: T('handler.archive.packaging'),
      onCancel,
    })
    const prepared = await prepareArtifact(
      props.entry.path,
      props.entry.meta,
      ARTIFACT_ARCHIVE,
      archivePackArgs(members)
    )
    const info = 'info' in prepared
      ? prepared.info
      : await taskDone(prepared.task, (running) => {
          if (canceled) return false
          task = running
          showTaskLoading({
            text: T('handler.archive.packaging_progress', {
              p: running.progress
                ? `${formatBytes(running.progress.loaded)}/${formatBytes(
                    running.progress.total
                  )}`
                : '',
            }),
            onCancel,
          })
        })
    if (!info || !info.ref) {
      throw new Error(String(T('handler.archive.pack_expired')))
    }
    const link = document.createElement('a')
    link.href = artifactRefUrl(
      props.entry.path,
      props.entry.meta,
      ARTIFACT_ARCHIVE,
      info.ref
    )
    link.target = '_blank'
    link.rel = 'noreferrer noopener nofollow'
    link.click()
  } catch (e: any) {
    if (e !== TASK_CANCELLED) alert(e.message)
  } finally {
    packaging.value = false
    showTaskLoading()
  }
}

const extractTo = async () => {
  const members = selectedMembers.value
  if (members.length === 0) return
  let destPath = ''
  try {
    const dest = await open({
      title: T('handler.archive.extract_open_title'),
      type: 'dir',
      filter: 'write',
      path: dir(props.entry.path),
    })
    destPath = dest.path
  } catch {
    return
  }

  extracting.value = true
  let canceled = false
  let task: Task<void> | undefined
  const onCancel = () => {
    canceled = true
    return task && deleteTask(task.id)
  }
  try {
    showTaskLoading({
      text: T('handler.archive.extracting'),
      onCancel,
    })
    await taskDone(extractArchive(props.entry.path, destPath, members), (running) => {
      if (canceled) return false
      task = running
      showTaskLoading({
        text: T('handler.archive.extracting_progress', {
          p: running.progress
            ? `${formatBytes(running.progress.loaded)}/${formatBytes(
                running.progress.total
              )}`
            : '',
        }),
        onCancel,
      })
    })
    emit('refresh')
  } catch (e: any) {
    if (e !== TASK_CANCELLED) alert(e.message)
  } finally {
    extracting.value = false
    showTaskLoading()
  }
}

watch(
  () => props.entry.path,
  () => {
    if (!props.entry.path) return
    currentDir.value = ''
    childrenByParent.value = new Map()
    selected.value = new Set()
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

.archive-view__check {
  display: flex;
  flex: none;
  align-items: center;
  padding-left: 8px;
}

.archive-view__footer {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  border-top: 1px solid var(--color-border);
  color: var(--color-text-muted);
  font-size: 13px;
}

.archive-view__actions {
  display: flex;
  flex: none;
  gap: 8px;
}
</style>

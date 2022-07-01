<template>
  <div class="entry-list-view">
    <EntryList
      v-if="!error"
      ref="entryListEl"
      :path="loadedPath"
      :entries="filteredEntries"
      :sort="sort"
      :selection="selection"
      :selectable="selectable"
      :view-mode="viewMode"
      :show-toggles="showToggles"
      :get-link="getLink"
      @entry-click="emit('entry-click', $event)"
      @entry-menu="emit('entry-menu', $event)"
      @update:path="emit('update:path', $event)"
      @update:sort="emit('update:sort', $event)"
      @update:selection="emit('update:selection', $event)"
      @update:view-mode="emit('update:viewMode', $event)"
    />
    <ErrorView v-else :status="error.status" :message="error.message" />
  </div>
</template>
<script lang="ts">
export default { name: 'EntryListView' }
</script>
<script setup lang="ts">
import { listEntries } from '@/api'
import { ApiError, RequestTask } from '@/utils/http'
import { EntryEventData, GetLinkFn, ListViewMode } from '@/components/entry'
import { Entry } from '@/types'
import { ref, computed, watch } from 'vue'
import { EntriesLoadData } from './types'

const props = defineProps({
  path: {
    type: String,
  },
  filter: {
    type: Function as PropType<Fn1<Entry, boolean>>,
  },
  sort: {
    type: String,
  },
  selection: {
    type: Array as PropType<Entry[]>,
  },
  selectable: {
    type: [Boolean, Function] as PropType<boolean | Fn1<Entry, boolean>>,
    default: true,
  },
  viewMode: {
    type: String as PropType<ListViewMode>,
  },
  showToggles: {
    type: Boolean,
  },
  getLink: {
    type: Function as PropType<GetLinkFn>,
  },
})

const emit = defineEmits<{
  (e: 'entry-click', v: EntryEventData): void
  (e: 'entry-menu', v: EntryEventData): void
  (e: 'update:path', v: EntryEventData): void
  (e: 'update:sort', v: string): void
  (e: 'update:selection', v: Entry[]): void
  (e: 'update:viewMode', v: ListViewMode): void
  (e: 'loading', v: boolean): void
  (e: 'entries-load', v: EntriesLoadData): void
  (e: 'error', v: any): void
}>()

const currentPath = ref<string | null>(null)
const loadedPath = ref('')
const entries = ref<Entry[]>([])
const error = ref<ApiError | null>(null)
const entryListEl = ref<InstanceType<EntryListType> | null>(null)
let task: RequestTask<Entry[]>
let lastEntry: string | undefined

const filteredEntries = computed(() =>
  props.filter ? entries.value.filter(props.filter) : entries.value
)

const focusOnEntry = (name: string, later?: boolean) => {
  if (later) {
    lastEntry = name
    return
  }
  entryListEl.value!.focusOnEntry(name)
}
const setViewMode = (mode: ListViewMode) => entryListEl.value!.setViewMode(mode)
const toggleViewMode = () => entryListEl.value!.toggleViewMode()
const setSortBy = (sort: string) => entryListEl.value!.setSortBy(sort)

const loadEntries = async () => {
  if (task) task.cancel()
  error.value = null
  emit('loading', true)
  try {
    const path = currentPath.value
    task = listEntries(path!)
    const loadedEntries = await task
    const thisEntry = loadedEntries[0]
    entries.value = loadedEntries.slice(1)
    loadedPath.value = path!
    emit('entries-load', {
      entries: entries.value,
      entry: thisEntry,
      path: loadedPath.value,
    })

    if (lastEntry) {
      focusOnEntry(lastEntry)
      lastEntry = undefined
    }
  } catch (e: any) {
    if (e.isCancel) return
    error.value = e
    emit('error', e)
  } finally {
    emit('loading', false)
  }
}

const commitPathChange = (path = '') => {
  if (currentPath.value === path) return
  currentPath.value = path
  loadEntries()
}

const tryRecoverState = (newPath: string, oldPath: string) => {
  if (!oldPath.startsWith(newPath)) return
  // navigate back
  // entry name
  const path = oldPath.substring(newPath ? newPath.length + 1 : newPath.length)
  focusOnEntry(path, true)
}

const reload = () => {
  loadEntries()
}

defineExpose({
  reload,
  focusOnEntry,
  setViewMode,
  toggleViewMode,
  setSortBy,
})

watch(
  () => props.path,
  (path, oldPath) => {
    tryRecoverState(path!, oldPath!)
    commitPathChange(path)
  }
)

commitPathChange(props.path)
</script>

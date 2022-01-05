<template>
  <div class="entry-list-view">
    <entry-list
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
    <error-view v-else :status="error.status" :message="error.message" />
  </div>
</template>
<script>
export default { name: 'EntryListView' }
</script>
<script setup>
import { listEntries } from '@/api'
import { ref, computed, watch, nextTick } from 'vue'

const props = defineProps({
  path: {
    type: String,
  },
  filter: {
    type: Function,
  },
  sort: {
    type: String,
  },
  selection: {
    type: Array,
  },
  selectable: {
    type: [Boolean, Function],
    default: true,
  },
  viewMode: {
    type: String,
  },
  showToggles: {
    type: Boolean,
  },
  getLink: {
    type: Function,
  },
})

const emit = defineEmits([
  'entry-click',
  'entry-menu',
  'update:path',
  'update:sort',
  'update:selection',
  'update:viewMode',
  'loading',
  'entries-load',
  'error',
])

const currentPath = ref(null)
const loadedPath = ref('')
const entries = ref([])
const error = ref(null)
const entryListEl = ref(null)
let task
let lastEntry

const filteredEntries = computed(() =>
  props.filter ? entries.value.filter(props.filter) : entries.value
)

const focusOnEntry = (name) => {
  entryListEl.value.focusOnEntry(name)
}

const loadEntries = async () => {
  if (task) task.cancel()
  error.value = null
  emit('loading', true)
  try {
    const path = currentPath.value
    task = listEntries(path)
    const loadedEntries = await task
    const thisEntry = loadedEntries[0]
    entries.value = loadedEntries.slice(1)
    loadedPath.value = path
    emit('entries-load', {
      entries: entries.value,
      entry: thisEntry,
      path: loadedPath.value,
    })

    await nextTick()
    if (lastEntry) {
      focusOnEntry(lastEntry)
      lastEntry = null
    }
  } catch (e) {
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

const tryRecoverState = (newPath, oldPath) => {
  if (!oldPath.startsWith(newPath)) return
  const path = oldPath.substr(newPath ? newPath.length + 1 : newPath.length)
  lastEntry = path
}

const reload = () => {
  loadEntries()
}

defineExpose({ reload, focusOnEntry })

watch(
  () => props.path,
  (path, oldPath) => {
    tryRecoverState(path, oldPath)
    commitPathChange(path)
  }
)

commitPathChange(props.path)
</script>

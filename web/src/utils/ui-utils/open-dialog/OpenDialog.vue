<template>
  <div class="open-dialog__inner">
    <entry-list-view
      v-model:selection="selection"
      :path="path"
      :filter="filterEntries"
      :selectable="dirMode ? false : isEntrySelectable"
      view-mode="list"
      @entry-click="entryClicked"
      @update:path="pathChanged"
      @entries-load="entriesLoaded"
    />
    <div v-if="!dirMode" class="open-dialog__selected-count">
      <span v-if="max > 0">{{ $t('dialog.open.max_items', { n: max }) }}</span>
      <span>{{ $t('dialog.open.n_selected', { n: selection.length }) }}</span>
      <a href="javascript:;" @click="clearSelection">
        {{ $t('dialog.open.clear') }}
      </a>
    </div>
  </div>
</template>
<script setup>
import EntryListView from '@/views/EntryListView/index.vue'
import { filenameExt } from '@/utils'
import { ref, unref, watch } from 'vue'

/// file,dir,<1024,.js,write
function createFilter(filter) {
  if (typeof filter !== 'string') return () => true
  const filters = filter
    .split(',')
    .map((f) => f.trim())
    .filter(Boolean)
  if (!filters.length) return () => true
  let allowFile, allowDir
  let maxSize = Number.POSITIVE_INFINITY
  let allowedExt = {}
  let writable
  filters.forEach((f) => {
    if (f === 'file') allowFile = true
    if (f === 'dir') allowDir = true
    if (f === 'write') writable = true
    if (f.startsWith('.')) allowedExt[f.substring(1).toLowerCase()] = true
    if (f.startsWith('<')) maxSize = parseInt(f.substring(1))
  })
  if (!allowDir && !allowFile) {
    allowDir = true
    allowFile = true
  }
  if (Object.keys(allowedExt).length === 0) allowedExt = null
  return (entry) => {
    if (!allowFile && entry.type === 'file') return false
    if (!allowDir && entry.type === 'dir') return false
    if (allowedExt && !allowedExt[filenameExt(entry.name)]) return false
    if (entry.size > maxSize) return false
    if (writable && !entry.meta.writable) return false
    return true
  }
}

const props = defineProps({
  loading: {
    type: String,
    required: true,
  },
  opts: {
    type: Object,
    required: true,
  },
})

const emit = defineEmits(['confirm-disabled'])

const dirMode = ref(false)
const path = ref('')
const message = ref('')
const selection = ref([])
const max = ref(0)
let filter

const beforeConfirm = () => {
  if (dirMode.value) return path.value
  return [...selection.value]
}

const selectionChanged = () => {
  confirmDisabled(!selection.value.length)
}

const isEntrySelectable = (entry) => {
  if (max.value > 0 && selection.value.length >= max.value) return false
  return filter(entry)
}

const entriesLoaded = ({ entry }) => {
  if (!dirMode.value) return
  confirmDisabled(!filter(entry))
}

const entryClicked = ({ entry, event }) => {
  event.preventDefault()
  if (!dirMode.value) {
    if (entry.type === 'file') {
      if (selection.value.findIndex((e) => e.path === entry.path) === -1) {
        selection.value.push(entry)
      }
      return
    }
  }
  path.value = entry.path
  confirmDisabled(true)
}

const pathChanged = ({ path: path_, event }) => {
  event.preventDefault()
  path.value = path_
  confirmDisabled(true)
}

const filterEntries = (entry) => {
  if (entry.type === 'dir') return true
  if (dirMode.value) return false
  return filter(entry)
}

const clearSelection = () => {
  selection.value.splice(0)
}

const confirmDisabled = (disabled) => {
  emit('confirm-disabled', disabled)
}

watch(
  () => selection.value,
  () => selectionChanged()
)

if (props.opts.type === 'dir') {
  dirMode.value = true
}

// filter selectable entries
if (typeof props.opts.filter === 'function') {
  filter = unref(props.opts.filter)
} else {
  filter = createFilter(props.opts.filter)
}
// max selection
let tempMax = +props.opts.max
if (tempMax <= 0) tempMax = 0
max.value = tempMax

message.value = props.opts.message || ''

confirmDisabled(true)

defineExpose({ beforeConfirm })
</script>
<style lang="scss">
.open-dialog__inner {
  position: relative;
  width: 320px;
  height: 50vh;
  display: flex;
  flex-direction: column;

  .entry-list-view {
    flex: 1;
    position: relative;
    overflow-x: hidden;
    overflow-y: auto;
    text-align: left;

    .entry-list__head {
      padding-top: 16px;
      padding-bottom: 10px;
      margin-bottom: 0;
      position: sticky;
      top: 0;
      background-color: var(--secondary-bg-color);
      box-shadow: 0 0 6px rgba(0, 0, 0, 0.2);
      z-index: 1;
    }
  }

  .open-dialog__selected-count {
    text-align: right;
    font-size: 12px;
    color: #999;

    a {
      color: #999;
      text-emphasis: none;
      margin-left: 1em;
    }
  }

  .entry-item__modified-time {
    display: none;
  }

  .entry-item__size {
    display: none;
  }
}
</style>

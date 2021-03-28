<template>
  <div class="open-dialog__inner">
    <entry-list-view
      :path="path"
      :filter="filterEntries"
      @entry-click="entryClicked"
      @path-change="pathChanged"
      @entries-load="entriesLoaded"
      :selection.sync="selection"
      :selectable="dirMode ? false : isEntrySelectable"
      view-mode="list"
    />
    <div class="open-dialog__selected-count" v-if="!dirMode">
      <span v-if="max > 0">{{ $t('dialog.open.max_items', { n: max }) }}</span>
      <span>{{ $t('dialog.open.n_selected', { n: selection.length }) }}</span>
      <a href="javascript:;" @click="clearSelection">
        {{ $t('dialog.open.clear') }}
      </a>
    </div>
  </div>
</template>
<script>
import EntryListView from '@/views/EntryListView'
import { getEntry } from '@/api'
import { filenameExt } from '@/utils'

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
    if (writable && !entry.meta.can_write) return false
    return true
  }
}

export default {
  name: 'OpenDialogInner',
  components: { EntryListView },
  props: {
    loading: {
      type: String,
      required: true,
    },
    opts: {
      type: Object,
      required: true,
    },
  },
  data() {
    return {
      dirMode: false,

      path: '',

      message: '',

      selection: [],
      max: 0,
    }
  },
  watch: {
    selection() {
      this.selectionChanged()
    },
  },
  created() {
    if (this.opts.type === 'dir') {
      this.dirMode = true
    }

    // filter selectable entries
    if (typeof this.opts.filter === 'function') {
      this._filter = this.opts.filter
    } else {
      this._filter = createFilter(this.opts.filter)
    }
    // max selection
    let max = +this.opts.max
    if (max <= 0) max = 0
    this.max = max

    this.message = this.opts.message || ''

    this.confirmDisabled(true)
  },
  methods: {
    beforeConfirm() {
      if (this.dirMode) return this.path
      return [...this.selection]
    },
    selectionChanged() {
      this.confirmDisabled(!this.selection.length)
    },
    isEntrySelectable(entry) {
      if (this.max > 0 && this.selection.length >= this.max) return false
      return this._filter(entry)
    },
    entriesLoaded(entries) {
      if (!this.dirMode) return
      this.cancelGetEntry()
      this._getEntryTask = getEntry(this.path).then(
        (entry) => {
          this.confirmDisabled(!this._filter(entry))
        },
        () => {}
      )
    },
    entryClicked({ entry, event }) {
      event.preventDefault()
      if (!this.dirMode) {
        if (entry.type === 'file') {
          if (this.selection.findIndex((e) => e.path === entry.path) === -1) {
            this.selection.push(entry)
          }
          return
        }
      }
      this.path = entry.path
      this.confirmDisabled(true)
      this.cancelGetEntry()
    },
    pathChanged({ path, event }) {
      event.preventDefault()
      this.path = path
      this.confirmDisabled(true)
      this.cancelGetEntry()
    },
    filterEntries(entry) {
      if (entry.type === 'dir') return true
      if (this.dirMode) return false
      return this._filter(entry)
    },
    cancelGetEntry() {
      if (this._getEntryTask) {
        this._getEntryTask.cancel()
        this._getEntryTask = null
      }
    },
    clearSelection() {
      this.selection.splice(0)
    },
    confirmDisabled(disabled) {
      this.$emit('confirm-disabled', disabled)
    },
  },
}
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
      @include var(background-color, secondary-bg-color);
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

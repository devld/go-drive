<template>
  <div class="entry-list-view">
    <entry-list
      v-if="!error"
      ref="entryList"
      :path="loadedPath"
      :entries="filteredEntries"
      @entry-click="$emit('entry-click', $event)"
      @entry-menu="$emit('entry-menu', $event)"
      @path-change="$emit('path-change', $event)"
      :selection="selection"
      @update:selection="$emit('update:selection', $event)"
      :selectable="selectable"
      :view-mode="viewMode"
    />
    <error-view v-else :status="error.status" :message="error.message" />
  </div>
</template>
<script>
import { listEntries } from '@/api'

export default {
  name: 'EntryListView',
  model: {
    prop: 'path',
    event: 'path-change'
  },
  props: {
    path: {
      type: String
    },
    filter: {
      type: Function
    },
    selection: {
      type: Array
    },
    selectable: {
      type: [Boolean, Function],
      default: true
    },
    viewMode: {
      type: String
    }
  },
  data () {
    return {
      currentPath: null,
      loadedPath: '',
      entries: [],

      error: null
    }
  },
  computed: {
    filteredEntries () {
      return this.filter ? this.entries.filter(this.filter) : this.entries
    }
  },
  watch: {
    path (path, oldPath) {
      this.tryRecoverState(path, oldPath)
      this.commitPathChange(path)
    }
  },
  created () {
    this.commitPathChange(this.path)
  },
  methods: {
    commitPathChange (path = '') {
      if (this.currentPath === path) return
      this.currentPath = path
      this.loadEntries()
    },
    tryRecoverState (newPath, oldPath) {
      if (!oldPath.startsWith(newPath)) return
      const path = oldPath.substr(newPath ? (newPath.length + 1) : newPath.length)
      this._lastEntry = path
    },
    focusOnEntry (name) {
      this.$refs.entryList.focusOnEntry(name)
    },
    async loadEntries () {
      if (this._task) this._task.cancel()
      this.error = null
      this.$emit('loading', true)
      try {
        const path = this.currentPath
        this._task = listEntries(path)
        this.entries = await this._task
        this.loadedPath = path
        this.$emit('entries-load', { entries: this.entries, path: this.loadedPath })

        await this.$nextTick()
        if (this._lastEntry) {
          this.focusOnEntry(this._lastEntry)
          this._lastEntry = null
        }
      } catch (e) {
        if (e.isCancel) return
        this.error = e
        this.$emit('error', e)
      } finally {
        this.$emit('loading', false)
      }
    },
    reload () {
      this.loadEntries()
    }
  }
}
</script>

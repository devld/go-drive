<template>
  <div class="entry-list-view">
    <entry-list
      v-if="!error"
      ref="entryList"
      :path="loadedPath"
      :entries="entries"
      @entry-click="$emit('entry-click', $event)"
      @entry-menu="$emit('entry-menu', $event)"
    />
    <error-view v-else :status="error.status" :message="error.message" />
  </div>
</template>
<script>
import { listEntries as listEntries_ } from '@/api'

const entriesCache = {}
async function listEntries (path, force) {
  let entries
  if (!force && entriesCache[path]) {
    entries = entriesCache[path]
    delete entriesCache[path]
    return entries
  }
  entries = await listEntries_(path)
  entriesCache[path] = entries
  return entries
}

export default {
  name: 'EntryListView',
  model: {
    prop: 'path',
    event: 'path-change'
  },
  props: {
    path: {
      type: String
    }
  },
  data () {
    return {
      currentPath: null,
      loadedPath: '',
      entries: [],

      error: null,

      errorMessages: {
        403: 'Operation Not Allowed',
        404: 'Resource Not Found',
        500: 'Server Error'
      }
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
      const path = oldPath.substr(newPath.length + 1)
      this._lastEntry = path
    },
    focusOnEntry (name) {
      this.$refs.entryList.focusOnEntry(name)
    },
    async loadEntries (force) {
      this.error = null
      this.$emit('loading', true)
      try {
        const path = this.currentPath
        this.entries = await listEntries(path, force)
        this.loadedPath = path
        this.$emit('entries-load', { entries: this.entries, path: this.loadedPath })

        await this.$nextTick()
        if (this._lastEntry) {
          this.focusOnEntry(this._lastEntry)
          this._lastEntry = null
        }
      } catch (e) {
        this.error = e
        this.$emit('error', e)
      } finally {
        this.$emit('loading', false)
      }
    },
    reload (force) {
      this.loadEntries(force)
    }
  }
}
</script>

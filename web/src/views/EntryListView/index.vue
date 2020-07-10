<template>
  <div class="entry-list-view">
    <entry-list
      v-if="!error"
      ref="entryList"
      :path="loadedPath"
      :entries="entries"
      :entry-link="entryLink"
      @entry-click="entryClicked"
    />
    <error-view v-else :status="error.status" :message="error.message" />
  </div>
</template>
<script>
import { listEntries } from '@/api'

const entriesCache = {}
async function listEntriesWithCache (path) {
  let entries
  if (entriesCache[path]) {
    entries = entriesCache[path]
    delete entriesCache[path]
    return entries
  }
  entries = await listEntries(path)
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
    },
    entryLink: {
      type: Function
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
    entryClicked (e) {
      if (e.entry.type === 'dir') {
        this.$emit('path-change', e)
      } else {
        this.$emit('open-file', e)
      }
    },
    commitPathChange (path = '') {
      if (this.currentPath === path) return
      this.currentPath = path
      this.loadEntries()
    },
    tryRecoverState (newPath, oldPath) {
      if (!oldPath.startsWith(newPath)) return
      let path = oldPath.substr(newPath.length)
      const i = path.indexOf('/')
      if (i >= 0) path = path.substr(0, i)
      this._lastEntry = path
    },
    focusOnEntry (name) {
      this.$refs.entryList.focusOnEntry(name)
    },
    async loadEntries () {
      this.error = null
      this.$emit('loading', true)
      try {
        const path = this.currentPath
        this.entries = await listEntriesWithCache(path)
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
    }
  }
}
</script>

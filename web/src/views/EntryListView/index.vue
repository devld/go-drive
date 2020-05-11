<template>
  <div class="entry-list-view">
    <entry-list
      v-if="!error"
      :path="loadedPath"
      :entries="entries"
      @path-change="pathChange"
      @open-file="openFile"
      :entry-link="entryLink"
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
    path () {
      this.commitPathChange(this.path)
    }
  },
  created () {
    this.commitPathChange(this.path)
  },
  methods: {
    openFile (e) {
      this.$emit('open-file', e)
    },
    pathChange (path) {
      this.commitPathChange(path)
    },
    commitPathChange (path = '/') {
      if (this.currentPath === path) return
      this.currentPath = path
      this.loadEntries()
      this.$emit('path-change', this.currentPath)
    },
    async loadEntries () {
      this.error = null
      this.$emit('loading', true)
      try {
        const path = this.currentPath
        this.entries = await listEntries(path)
        this.loadedPath = path
        this.$emit('entries-load', { entries: this.entries, path: this.loadedPath })
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

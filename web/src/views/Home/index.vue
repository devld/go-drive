<template>
  <div class="home">
    <main class="files-list">
      <entry-list-view
        :path="path"
        :entry-link="makeEntryRouteLink"
        @entries-load="entriesLoaded"
        @open-file="openFile"
      />
    </main>
    <footer class="page-footer" v-if="readmeContent">
      <div class="markdown-body" v-markdown="readmeContent">
        <p style="text-align: center;">Loading README...</p>
      </div>
    </footer>
    <div class="file-viewer-dialog" v-if="fileView">
      <component
        :is="fileView.component"
        :path="fileView.path"
        :key="fileView.path"
        @close="closeFileView"
        @file-change="fileViewChange"
      />
    </div>
  </div>
</template>
<script>
import EntryListView from '@/views/EntryListView'

import { getContent } from '@/api'
import { pathJoin, pathClean } from '@/utils'

import { resolveEntryHandler, HANDLER_COMPONENTS, getHandler } from './file-handlers'

const README_FILENAME = 'readme.md'
const README_FAILED_CONTENT = '<p style="text-align: center;">Failed to load README.md</p>'

export default {
  name: 'Home',
  components: { EntryListView, ...HANDLER_COMPONENTS },
  data () {
    return {
      path: '/',
      readmeContent: '',

      fileView: null
    }
  },
  beforeRouteUpdate (to, from, next) {
    this.path = '/' + to.params.path
    next()
    this.resolveFileView()
  },
  created () {
    const path = this.$route.params.path
    this.path = '/' + (path || '')
    this.resolveFileView()
  },
  methods: {
    openFile ({ entry, path }) {
      const routePath = this.makeEntryRouteLink(entry, path)
      if (routePath) {
        location.href = routePath
      }
    },
    entriesLoaded ({ entries, path }) {
      if (path !== this.path) {
        this.$router.push(`/files${path}`)
      }
      this.tryLoadReadme(entries)
    },
    async tryLoadReadme (entries) {
      let readmeFound
      for (const e of entries) {
        if (e.type !== 'file') continue
        if (README_FILENAME.toLowerCase() === e.name.toLowerCase()) {
          readmeFound = e
          break
        }
      }
      if (readmeFound) {
        await this.loadReadme(readmeFound.name)
      } else {
        this.readmeContent = ''
      }
    },
    async loadReadme (name) {
      try {
        this.readmeContent = await getContent(pathJoin(this.path, name))
      } catch (e) {
        this.readmeContent = README_FAILED_CONTENT
      }
    },
    makeEntryRouteLink (entryOrPath, path) {
      const basePath = '#/files'
      if (typeof (entryOrPath) === 'string') {
        return `${basePath}${entryOrPath}`
      }
      const entry = entryOrPath
      if (entry.type === 'drive' || entry.type === 'dir') {
        return `${basePath}${pathClean(pathJoin(path, entry.name))}`
      }
      const handlers = resolveEntryHandler(entry, path)
      if (handlers.length > 0) {
        return this.makeEntryHandlerLink(handlers[0], entry)
      }
    },
    makeEntryHandlerLink (handler, entry) {
      return `#/files${this.path}?` +
        `v=${handler.name}&` +
        `f=${encodeURIComponent(entry.name || '')}`
    },
    resolveFileView () {
      const handler = getHandler(this.$route.query.v)
      const file = this.$route.query.f
      if (!handler || !file) {
        this.fileView = null
      } else {
        this.fileView = {
          component: handler.view.name,
          path: pathClean(pathJoin(this.path, file))
        }
      }
    },
    closeFileView () {
      if (document.referrer && location.href.startsWith(document.referrer)) {
        this.$router.go(-1)
      } else {
        this.$router.replace(`/files${this.path}`)
      }
    },
    fileViewChange () {

    }
  }
}
</script>
<style lang="scss">
.files-list {
  max-width: 880px;
  margin: 42px auto 0;
  background-color: #fff;
  padding: 16px 0;
  border-radius: 16px;
}

.page-footer {
  box-sizing: border-box;
  max-width: 880px;
  margin: 42px auto;
  background-color: #fff;
  padding: 16px;
  border-radius: 16px;
}

.file-viewer-dialog {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
}

@media screen and (max-width: 900px) {
  .home {
    margin: 16px;
  }
}
</style>

<template>
  <div class="home">
    <main class="files-list">
      <entry-list-view
        ref="entryList"
        :path="path"
        @path-change="pathChanged"
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
    <div class="enter-handler-dialog" v-if="entryHandlerView && entries">
      <div class="entry-handler-view">
        <component
          :is="entryHandlerView.component"
          :path="entryHandlerView.path"
          :entries="entries"
          @close="closeEntryHandlerView"
          @entry-change="entryHandlerViewChange"
        />
      </div>
    </div>
  </div>
</template>
<script>
import EntryListView from '@/views/EntryListView'

import { getContent } from '@/api'
import { pathJoin, pathClean } from '@/utils'

import { resolveEntryHandler, HANDLER_COMPONENTS, getHandler } from './entry-handlers'

const README_FILENAME = 'readme.md'
const README_FAILED_CONTENT = '<p style="text-align: center;">Failed to load README.md</p>'

export default {
  name: 'Home',
  components: { EntryListView, ...HANDLER_COMPONENTS },
  data () {
    return {
      path: '/',
      readmeContent: '',

      entryHandlerView: null,
      entries: null
    }
  },
  beforeRouteUpdate (to, from, next) {
    this.path = '/' + to.params.path
    next()
    this.resolveEntryHandlerView()
  },
  created () {
    const path = this.$route.params.path
    this.path = '/' + (path || '')
    this.resolveEntryHandlerView(true)
  },
  methods: {
    openFile ({ entry, path }) {
      const routePath = this.makeEntryRouteLink(entry, path)
      if (routePath) {
        location.href = routePath
      }
    },
    pathChanged () {
      this.entries = null
    },
    entriesLoaded ({ entries, path }) {
      if (path !== this.path) {
        this.$router.push(`/files${path}`)
      }
      this.entries = entries
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
      if (entry.type === 'dir') {
        return `${basePath}${pathClean(pathJoin(path, entry.name))}`
      }
      const handlers = resolveEntryHandler(entry, path)
      if (handlers.length > 0) {
        return this.makeEntryHandlerLink(handlers[0].name, entry.name)
      }
    },
    makeEntryHandlerLink (handlerName, entryName) {
      return `#/files${this.path}?` +
        `handler=${handlerName}&` +
        `entry=${encodeURIComponent(entryName || '')}`
    },
    resolveEntryHandlerView (init) {
      const handler = getHandler(this.$route.query.handler)
      const entry = this.$route.query.entry
      if (!handler || !entry) {
        this.entryHandlerView = null
      } else {
        this.noHistory = init
        this.entryHandlerView = {
          handler: handler.name,
          component: handler.view.name,
          path: pathClean(pathJoin(this.path, entry)),
          entryName: entry
        }
      }
    },
    closeEntryHandlerView () {
      this.focusOnEntry(this.entryHandlerView.entryName)
      if (this.noHistory) {
        this.$router.replace(`/files${this.path}`)
      } else {
        this.$router.go(-1)
      }
    },
    entryHandlerViewChange (name) {
      this.focusOnEntry(name)
      if (this.entries.findIndex(e => e.name === name) === -1) return
      location.replace(this.makeEntryHandlerLink(this.entryHandlerView.handler, name))
    },
    focusOnEntry (name) {
      this.$refs.entryList.focusOnEntry(name)
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

.enter-handler-dialog {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  overflow: hidden;
  background-color: rgba(0, 0, 0, 0.1);
}

.entry-handler-view {
  width: 100%;
  height: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
  animation: scale-in 0.4s;
}

@media screen and (max-width: 900px) {
  .home {
    margin: 16px;
  }
}
</style>

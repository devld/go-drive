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
    <div
      class="enter-handler-dialog"
      v-if="entryHandlerView && entries && findEntry(entryHandlerView.entryName)"
    >
      <div class="entry-handler-view">
        <component
          :is="entryHandlerView.component"
          :entry="findEntry(entryHandlerView.entryName)"
          :entries="entries"
          @close="closeEntryHandlerView"
          @entry-change="entryHandlerViewChange"
          @save-state="entryHandlerViewSaveStateChange"
        />
      </div>
    </div>
  </div>
</template>
<script>
import EntryListView from '@/views/EntryListView'

import { getContent } from '@/api'
import { pathJoin, filename, dir } from '@/utils'

import { resolveEntryHandler, HANDLER_COMPONENTS, getHandler } from './entry-handlers'

const README_FILENAME = 'readme.md'
const README_FAILED_CONTENT = '<p style="text-align: center;">Failed to load README.md</p>'

const THIS_PATH_NAME = 'files'

export default {
  name: 'Home',
  components: { EntryListView, ...HANDLER_COMPONENTS },
  props: {
    path: {
      type: String,
      required: true
    }
  },
  data () {
    return {
      readmeContent: '',

      entryHandlerView: null,
      entries: null
    }
  },
  beforeRouteUpdate (to, from, next) {
    this.confirmUnsavedState().then(() => {
      next()
      this.resolveEntryHandlerView()
    }, () => {
      next(false)
    })
  },
  created () {
    this.resolveEntryHandlerView()
  },
  methods: {
    openFile ({ entry, path }) {
      const routePath = this.makeEntryRouteLink(path, entry)
      if (routePath) {
        location.href = routePath
      }
    },
    pathChanged () {
      this.entries = null
    },
    entriesLoaded ({ entries, path }) {
      if (path !== this.path) {
        this.$router.push(`/${THIS_PATH_NAME}/${path}`)
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
    makeEntryRouteLink (path, entry) {
      if (!entry || entry.type === 'dir') {
        return `#/${THIS_PATH_NAME}/${path}`
      }
      const handlers = resolveEntryHandler(entry, path)
      if (handlers.length > 0) {
        return this.makeEntryHandlerLink(handlers[0].name, entry.name)
      }
    },
    makeEntryHandlerLink (handlerName, entryName) {
      return `#${this.$route.path}?handler=${handlerName}&entry=${encodeURIComponent(entryName)}`
    },
    resolveEntryHandlerView () {
      const handler = getHandler(this.$route.query.handler)
      const entry = this.$route.query.entry
      if (!handler || !entry) {
        this.entryHandlerView = null
      } else {
        this.entryHandlerView = {
          handler: handler.name,
          component: handler.view.name,
          entryName: entry,
          savedState: true
        }
      }
    },
    async closeEntryHandlerView () {
      try { await this.confirmUnsavedState() } catch { return }
      this.focusOnEntry(this.entryHandlerView.entryName)
      this.$router.replace(this.$route.path)
    },
    async entryHandlerViewChange (path) {
      try { await this.confirmUnsavedState() } catch { return }
      const dirPath = dir(path)
      const name = filename(path)
      this.focusOnEntry(name)
      const newPath = `/${THIS_PATH_NAME}/${dirPath}` +
        `?handler=${this.entryHandlerView.handler}&entry=${encodeURIComponent(name)}`
      if (decodeURIComponent(this.$route.fullPath) !== decodeURIComponent(newPath)) {
        this.$router.replace(newPath)
      }
    },
    entryHandlerViewSaveStateChange (saved) {
      this.entryHandlerView.savedState = saved
    },
    confirmUnsavedState () {
      if (!this.entryHandlerView || this.entryHandlerView.savedState) return Promise.resolve()
      return new Promise((resolve, reject) => {
        if (confirm('You have some unsaved changes, are you sure to leave?')) {
          resolve()
        } else {
          // eslint-disable-next-line prefer-promise-reject-errors
          reject()
        }
      })
    },
    focusOnEntry (name) {
      this.$refs.entryList.focusOnEntry(name)
    },
    findEntry (name) {
      return this.entries.find(e => e.name === name)
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
}

@media screen and (max-width: 900px) {
  .home {
    margin: 16px;
  }
}
</style>

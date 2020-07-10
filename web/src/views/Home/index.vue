<template>
  <div class="home">
    <main class="files-list">
      <entry-list-view
        ref="entryList"
        :path="path"
        :entry-link="makeEntryLink"
        @path-change="pathChanged"
        @entries-load="entriesLoaded"
        @open-file="openFile"
      />
    </main>
    <footer class="page-footer" v-if="readmeContent">
      <div class="markdown-body" v-markdown="readmeContent">
        <p style="text-align: center;">Loading README...</p>
      </div>
    </footer>
    <!-- entry handler view dialog -->
    <div class="enter-handler-dialog" v-if="handlerViewDialogShowing">
      <transition name="top-fade" @after-leave="afterHandlerViewLeave">
        <div
          class="entry-handler-view"
          v-if="entryHandlerView && entries && entryHandlerView.entry"
        >
          <component
            :is="entryHandlerView.component"
            :entry="entryHandlerView.entry"
            :entries="entries"
            @close="closeEntryHandlerView"
            @entry-change="entryHandlerViewChange"
            @save-state="entryHandlerViewSaveStateChange"
          />
        </div>
      </transition>
    </div>
    <!-- entry handler view dialog -->
  </div>
</template>
<script>
import EntryListView from '@/views/EntryListView'

import { getContent } from '@/api'
import { filename, dir } from '@/utils'

import { resolveEntryHandler, HANDLER_COMPONENTS, getHandler } from './entry-handlers'

const README_FILENAME = 'readme.md'
const README_FAILED_CONTENT = '<p style="text-align: center;">Failed to load README.md</p>'

const THIS_PATH_NAME = 'files'

const HISTORY_FLAG = '_h'

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
      entries: null,

      handlerViewDialogShowing: false
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
    openFile ({ entry, event }) {
      sessionStorage.setItem(HISTORY_FLAG, '1')
    },
    pathChanged () {
      this.entries = null
    },
    entriesLoaded ({ entries, path }) {
      if (path !== this.path) {
        this.$router.push(this.makeRoutePath(path))
      }
      this.entries = entries
      if (this.entryHandlerView) {
        this.entryHandlerView.entry =
          entries.find(e => e.name === this.entryHandlerView.entryName)
      }
      this.tryLoadReadme(entries)
    },
    resolveEntryHandlerView () {
      const handler = getHandler(this.$route.query.handler)
      const entryName = this.$route.query.entry
      if (!handler || !entryName) {
        this.entryHandlerView = null
        sessionStorage.removeItem(HISTORY_FLAG)
      } else {
        this.handlerViewDialogShowing = true
        this.$nextTick(() => {
          this.entryHandlerView = {
            handler: handler.name,
            component: handler.view.name,
            entryName,
            savedState: true,
            entry: this.entries && this.entries.find(e => e.name === entryName)
          }
        })
      }
    },
    async closeEntryHandlerView () {
      try { await this.confirmUnsavedState() } catch { return }
      this.focusOnEntry(this.entryHandlerView.entryName)
      if (sessionStorage.getItem(HISTORY_FLAG)) {
        this.$router.go(-1)
      } else {
        this.$router.replace(this.$route.path)
      }
    },
    async entryHandlerViewChange (path) {
      try { await this.confirmUnsavedState() } catch { return }
      const dirPath = dir(path)
      const name = filename(path)
      this.focusOnEntry(name)
      const newPath = this.makeEntryHandlerPath(this.entryHandlerView.handler, name, dirPath)
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
    makeEntryLink (path, entry) {
      if (!entry || entry.type === 'dir') {
        return '#' + this.makeRoutePath(path)
      }
      const handlers = resolveEntryHandler(entry, path)
      if (handlers.length > 0) {
        return '#' + this.makeEntryHandlerPath(handlers[0].name, entry.name)
      }
    },
    makeEntryHandlerPath (handlerName, entryName, path) {
      return this.makeRoutePath(encodeURI(path || this.path)) +
        `?handler=${handlerName}&entry=${encodeURIComponent(entryName)}`
    },
    makeRoutePath (path) {
      return `/${THIS_PATH_NAME}/${path}`
    },
    afterHandlerViewLeave () {
      this.handlerViewDialogShowing = false
    },
    focusOnEntry (name) {
      this.$refs.entryList.focusOnEntry(name)
    },
    getHandlerViewEntry (name) {
      return this.entries.find(e => e.name === name)
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
        await this.loadReadme(readmeFound.path)
      } else {
        this.readmeContent = ''
      }
    },
    async loadReadme (path) {
      try {
        this.readmeContent = await getContent(path)
      } catch (e) {
        this.readmeContent = README_FAILED_CONTENT
      }
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

<template>
  <div class="entry-explorer">
    <!-- file list main area -->
    <div class="files-list">
      <entry-list-view
        ref="entryListEl"
        v-model:selection="selectedEntries"
        v-model:sort="sortBy"
        v-model:view-mode="viewMode"
        :path="path"
        show-toggles
        :get-link="getLink"
        @entries-load="entriesLoaded"
        @entry-click="entryClicked"
        @entry-menu="showEntryMenu"
        @loading="progressBar($event)"
      />
    </div>
    <!-- file list main area -->

    <!-- README -->
    <div v-if="readmeContent" class="page-footer">
      <div v-markdown="readmeContent" class="markdown-body"></div>
    </div>
    <!-- README -->

    <!-- entry handler view dialog -->
    <dialog-view class="entry-handler-dialog" :show="entryHandlerViewShowing">
      <component
        :is="HANDLER_COMPONENTS[entryHandlerView.component]"
        v-if="entryHandlerViewShowing"
        :entry="entryHandlerView.entry"
        :entries="entries"
        @update="reloadEntryList"
        @close="closeEntryHandlerView"
        @entry-change="entryHandlerViewChange"
        @save-state="entryHandlerViewSaveStateChange"
      />
    </dialog-view>
    <!-- entry handler view dialog -->

    <!-- entry menu -->
    <dialog-view
      v-model:show="entryMenuShowing"
      overlay-close
      esc-close
      transition="top-fade"
    >
      <entry-menu
        v-if="entryMenuData"
        :menus="entryMenuData.menus"
        :entry="entryMenuData.entry"
        @click="menuClicked"
      />
    </dialog-view>
    <!-- entry menu -->

    <!-- new entry menu -->
    <new-entry-area
      ref="newEntryAreaEl"
      :path="path"
      :entries="entries"
      :readonly="isCurrentDirReadonly"
      @update="reloadEntryList"
    />
    <!-- new entry menu -->
  </div>
</template>
<script>
export default { name: 'EntryExplorer' }
</script>
<script setup>
import EntryListView from '@/views/EntryListView/index.vue'
import EntryMenu from './EntryMenu.vue'
import NewEntryArea from './NewEntryArea.vue'

import { getContent } from '@/api'
import { filename, dir, debounce, setTitle } from '@/utils'

import {
  resolveEntryHandler,
  HANDLER_COMPONENTS,
  getHandler,
} from '@/utils/handlers'
import { useStore } from 'vuex'
import uiUtils, { confirm } from '@/utils/ui-utils'
import { computed, onBeforeUnmount, ref } from 'vue'
import { onBeforeRouteLeave, onBeforeRouteUpdate, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useEntryExplorer } from '@/utils/explorer'

const README_FILENAME = 'readme.md'

const HISTORY_FLAG = '_h'
const setHistoryFlag = () => {
  sessionStorage.setItem(HISTORY_FLAG, '1')
}
const getHistoryFlag = () => {
  const val = sessionStorage.getItem(HISTORY_FLAG)
  sessionStorage.removeItem(HISTORY_FLAG)
  return !!val
}

const { t } = useI18n()

const store = useStore()
const router = useRouter()

const props = defineProps({
  basePath: {
    type: String,
    required: true,
  },
})

const {
  getDirLink,
  getHandlerLink,
  getLink,
  resolveHandlerByRoute,
  resolvePath,
} = useEntryExplorer(props.basePath)

const path = computed(() => resolvePath(router.currentRoute.value))

const readmeContent = ref('')

const entryHandlerView = ref(null)

const entryMenuData = ref(null)
const entryMenuShowing = ref(false)

const entries = ref(null)
const selectedEntries = ref([])

const currentDirEntry = ref(null)

const viewMode = ref('list')
const sortBy = ref(undefined)

const entryListEl = ref(null)
const newEntryAreaEl = ref(null)

const user = computed(() => store.state.user)

let readmeTask

const entryHandlerViewShowing = computed(
  () =>
    !!(entryHandlerView.value && entries.value && entryHandlerView.value.entry)
)

const isCurrentDirReadonly = computed(
  () => !currentDirEntry.value || !currentDirEntry.value.meta.writable
)

const progressBar = (v) => store.commit('progressBar', v)

const entryClicked = ({ entry, event }) => {
  if (selectedEntries.value.length > 0) {
    selectedEntries.value.splice(0)
    event.preventDefault()
    return
  }
  if (entry.type === 'dir') {
    // path changed
    entries.value = null
    currentDirEntry.value = null
    return
  }
  const handlers = resolveEntryHandler(entry, currentDirEntry.value, user.value)
  if (handlers.length > 0) {
    executeEntryHandler(handlers[0], entry)
  }
}

const menuClicked = ({ entry, menu }) => {
  entryMenuShowing.value = false
  const handler = getHandler(menu.name)
  if (!handler) return

  if (handler.view) {
    if (Array.isArray(entry)) {
      // selected entries
      entryHandlerView.value = {
        handler: handler.name,
        component: handler.view?.name,
        entry,
        savedState: true,
      }
    } else {
      router.push(getHandlerLink(handler.name, entry.name, path.value))
    }
    return
  }
  // execute handler
  executeEntryHandler(handler, entry)
}

const executeEntryHandler = async (handler, entry) => {
  if (typeof handler.handler === 'function' && !handler.view) {
    try {
      const r = await handler.handler(entry, uiUtils)
      if (r && r.update) reloadEntryList()
    } catch (e) {
      console.error('entry handler error', e)
    }
  }
}

const showEntryMenu = ({ entry, event }) => {
  if (selectedEntries.value.length > 0) {
    entry = [...selectedEntries.value] // selected entries
  }
  const handlers = resolveEntryHandler(entry, currentDirEntry.value, user.value)
  if (handlers.length === 0) return

  event && event.preventDefault()

  entryMenuData.value = {
    entry,
    menus: handlers
      .filter((h) => h.display)
      .map((h) => ({
        name: h.name,
        display: typeof h.display === 'function' ? h.display(entry) : h.display,
      })),
  }
  entryMenuShowing.value = true
}

const entriesLoaded = ({
  entries: entries_,
  path: path_,
  entry: thisEntry,
}) => {
  setTitle(path_)

  if (path_ !== path.value) {
    router.push(getDirLink(path_))
  }
  tryLoadReadme(entries_)

  entries.value = entries_
  currentDirEntry.value = thisEntry

  selectedEntries.value.splice(0)

  if (entryHandlerView.value?.entryName) {
    entryHandlerView.value.entry = entries_.find(
      (e) => e.name === entryHandlerView.value.entryName
    )

    setTitle(`${entryHandlerView.value.entryName}`)
  }
}

const resolveRouteAndHandleEntry = (to) => {
  to = to || router.currentRoute.value
  const matched = resolveHandlerByRoute(to)
  if (!matched) {
    entryHandlerView.value = null
    return false
  }
  const { handler, entryName } = matched
  const entry = entries.value?.find((e) => e.name === entryName)

  if (handler.view) {
    // handler view dialog
    entryHandlerView.value = {
      handler: handler.name,
      component: handler.view?.name,
      entryName,
      entry,
      savedState: true,
    }

    setTitle(`${entryName}`)
  }
}
const closeEntryHandlerView = () => {
  setTitle(path.value)

  if (!entryHandlerView.value) return
  if (entryHandlerView.value.entryName) {
    focusOnEntry(entryHandlerView.value.entryName)
  }
  if (!replaceHandlerRoute()) {
    entryHandlerView.value = null
  }
}
const entryHandlerViewChange = async (path) => {
  try {
    await confirmUnsavedState()
  } catch {
    return
  }
  const dirPath = dir(path)
  const name = filename(path)
  focusOnEntry(name)
  const newPath = getHandlerLink(entryHandlerView.value.handler, name, dirPath)
  if (
    decodeURIComponent(router.currentRoute.value.fullPath) !==
    decodeURIComponent(newPath)
  ) {
    router.replace(newPath)
  }
}

const entryHandlerViewSaveStateChange = (saved) => {
  entryHandlerView.value.savedState = saved
}

const confirmUnsavedState = () => {
  if (!entryHandlerView.value || entryHandlerView.value.savedState) {
    return Promise.resolve()
  }
  return confirm(t('p.home.unsaved_confirm'))
}

const onWindowUnload = (e) => {
  if (!entryHandlerView.value || entryHandlerView.value.savedState) return
  e.preventDefault()
  e.returnValue = ''
}

const replaceHandlerRoute = () => {
  if (getHistoryFlag()) {
    router.go(-1)
    return true
  } else {
    const route = router.currentRoute.value
    if (route.fullPath !== route.path) {
      router.replace(route.path)
      return true
    }
  }
}

const focusOnEntry = (name) => {
  entryListEl.value.focusOnEntry(name)
}

const tryLoadReadme = async (entries) => {
  let readmeFound
  for (const e of entries) {
    if (e.type !== 'file') continue
    if (README_FILENAME.toLowerCase() === e.name.toLowerCase()) {
      readmeFound = e
      break
    }
  }
  if (readmeFound) {
    await loadReadme(readmeFound)
  } else {
    readmeContent.value = ''
  }
}

const loadReadme = async (entry) => {
  readmeTask?.cancel()
  let content
  readmeContent.value = `<p style="text-align: center">${t(
    'p.home.readme_loading'
  )}</p>`
  readmeTask = getContent(entry.path, entry.meta.accessKey)
  try {
    content = await readmeTask
  } catch (e) {
    if (e.isCancel) return
    content = `<p style="text-align: center;">${t('p.home.readme_failed')}</p>`
  }
  if (path.value === dir(entry.path)) {
    readmeContent.value = content
  }
}

const reloadEntryList = debounce(() => {
  selectedEntries.value.splice(0)
  entryListEl.value.reload()
}, 500)

const onKeyDown = (e) => {
  if (e.key === 'Escape') {
    closeEntryHandlerView()
    e.stopPropagation()
    e.preventDefault()
  }
}

onBeforeRouteUpdate((to, from, next) => {
  if (!resolveHandlerByRoute(from) && resolveHandlerByRoute(to)) {
    setHistoryFlag()
  }
  confirmUnsavedState().then(
    () => {
      next()
      resolveRouteAndHandleEntry(to)
    },
    () => {
      next(false)
    }
  )
})

onBeforeRouteLeave((to, from, next) => {
  progressBar(false)
  next()
})

window.addEventListener('beforeunload', onWindowUnload)
window.addEventListener('keydown', onKeyDown)
resolveRouteAndHandleEntry()

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('beforeunload', onWindowUnload)
})
</script>
<style lang="scss">
.entry-explorer {
  margin-bottom: 40px;
}

.files-list {
  max-width: 880px;
  margin: 16px auto 0;
  background-color: var(--primary-bg-color);
  padding: 16px 0;
  border-radius: 16px;
}

.page-footer {
  box-sizing: border-box;
  max-width: 880px;
  margin: 42px auto;
  background-color: var(--primary-bg-color);
  padding: 16px;
  border-radius: 16px;
}

.entry-handler-dialog {
  .dialog-view__content {
    background-color: transparent;
  }
}

@media screen and (max-width: 900px) {
  .entry-explorer {
    margin: 16px;

    .entry-item--view-list {
      .entry-item__info {
        flex-direction: column;
        justify-content: center;
        align-items: stretch;
      }

      .entry-item__name {
        flex: unset;
      }

      .entry-item__meta {
        display: flex;
        font-size: 12px;
        color: var(--secondary-text-color);
        justify-content: space-between;
        margin-top: 4px;
      }
    }
  }
}
</style>

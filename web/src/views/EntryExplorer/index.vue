<template>
  <div
    class="entry-explorer"
    :class="{ 'search-disabled': !searchConfig?.enabled }"
  >
    <!-- search panel -->
    <div v-if="searchConfig?.enabled" class="search-panel-wrapper">
      <SearchPanel :path="path" @navigate="navigateToEntry" />
    </div>
    <!-- search panel -->

    <!-- file list main area -->
    <div class="files-list">
      <EntryListView
        ref="entryListEl"
        v-model:selection="selectedEntries"
        v-model:sort="sortBy"
        :view-mode="viewMode"
        :path="path"
        show-toggles
        draggable
        :get-link="getLink"
        @update:view-mode="onViewModeChanged"
        @entries-load="entriesLoaded"
        @entry-click="onEntryClicked"
        @entry-menu="showEntryMenu"
        @loading="progressBar($event)"
        @drag-action="onEntriesDragAction"
      />
    </div>
    <!-- file list main area -->

    <!-- README -->
    <ReadmeContent class="page-footer" :path="path" :entries="entries" />
    <!-- README -->

    <!-- entry menu -->
    <DialogView
      v-model:show="entryMenuShowing"
      overlay-close
      esc-close
      transition="top-fade"
    >
      <EntryMenu
        v-if="entryMenuData"
        :menus="entryMenuData.menus"
        :entry="entryMenuData.entry"
        @click="onEntryMenuClicked"
      />
    </DialogView>
    <!-- entry menu -->

    <!-- new entry menu -->
    <NewEntryArea
      ref="newEntryAreaEl"
      :path="path"
      :entries="entries"
      :readonly="isCurrentDirReadonly"
      @update="reloadEntryList"
    />
    <!-- new entry menu -->
  </div>
</template>
<script lang="ts">
export default { name: 'EntryExplorer' }
</script>
<script setup lang="ts">
import { mountPaths } from '@/api/admin'
import { EntryEventData, ListViewMode } from '@/components/entry'
import { EntryDragData } from '@/components/entry/useDrag'
import { EntryHandler, EntryHandlersMenu } from '@/handlers/types'
import { useAppStore } from '@/store'
import { Entry } from '@/types'
import { debounce, dir, filename, setTitle } from '@/utils'
import { copyOrMove } from '@/utils/entry'
import { confirm, loading } from '@/utils/ui-utils'
import EntryListView from '@/views/EntryListView/index.vue'
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  onBeforeRouteLeave,
  onBeforeRouteUpdate,
  RouteLocationNormalized,
  useRouter,
} from 'vue-router'
import { EntriesLoadData } from '../EntryListView/types'
import EntryMenu from './EntryMenu.vue'
import { useEntryExplorer, useEntryHandler } from './explorer'
import NewEntryArea from './NewEntryArea.vue'
import ReadmeContent from './ReadmeContent.vue'
import SearchPanel from './SearchPanel/index.vue'
import { EntryMenuClickData } from './types'

const VIEW_MODE_STORAGE_KEY = 'entries-list-view-mode'

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

const store = useAppStore()
const router = useRouter()

const props = defineProps({
  basePath: {
    type: String,
    required: true,
  },
})

/** current path */
const path = computed(() => resolvePath(router.currentRoute.value)!)

const currentDirEntry = ref<Entry | undefined>()
const entries = ref<Entry[] | undefined>()

const selectedEntries = ref<Entry[]>([])

const entryMenuData = ref<EntryHandlersMenu | undefined>()
const entryMenuShowing = ref(false)

const viewMode = ref<ListViewMode>(
  (localStorage.getItem(VIEW_MODE_STORAGE_KEY) as ListViewMode) || 'list'
)
const onViewModeChanged = debounce((mode) => {
  viewMode.value = mode
  localStorage.setItem(VIEW_MODE_STORAGE_KEY, mode)
}, 500)

const sortBy = ref(undefined)

const entryListEl = ref<InstanceType<typeof EntryListView> | null>(null)
const newEntryAreaEl = ref<InstanceType<typeof NewEntryArea> | null>(null)

const searchConfig = computed(() => store.config?.search)

const isCurrentDirReadonly = computed(
  () => !currentDirEntry.value || !currentDirEntry.value.meta.writable
)

let currentTempRoute: RouteLocationNormalized | undefined

const {
  handlerCtx,
  getDirLink,
  getHandlerLink,
  getLink,
  resolveHandlerByRoute,
  resolvePath,
  isRouteForHandlerView,
} = useEntryExplorer(props.basePath)

onBeforeRouteUpdate((to, from, next) => {
  if (!resolveHandlerByRoute(from) && resolveHandlerByRoute(to)) {
    setHistoryFlag()
  }
  confirmUnsavedState().then(
    () => {
      next()
      currentTempRoute = to
      onRouteChanged(currentTempRoute)
    },
    () => {
      next(false)
    }
  )
})

onBeforeRouteLeave((_to, _from, next) => {
  progressBar(false)
  next()
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('beforeunload', onWindowUnload)
})

const reloadEntryList = debounce(() => {
  selectedEntries.value.splice(0)
  entryListEl.value!.reload()
}, 500)

const onHandlerBeingExecute = (
  handler: EntryHandler,
  entry: Entry | Entry[]
) => {
  if (handler.view && !Array.isArray(entry)) {
    if (
      currentTempRoute &&
      isRouteForHandlerView(currentTempRoute, handler.name, entry.name)
    ) {
      return
    }

    router.push(getHandlerLink(path.value, handler.name, entry.name))
  }
}

const onHandlerBeingHide = async (entry: Entry | Entry[]) => {
  await confirmUnsavedState()

  setTitle(path.value)
  if (!Array.isArray(entry)) {
    focusOnEntry(entry.name)
  }
  removeHandlerRoute()
}

const onHandlerEntryBeingChanged = async (path: string, handler: string) => {
  await confirmUnsavedState()

  const dirPath = dir(path)
  const name = filename(path)

  if (
    !isRouteForHandlerView(router.currentRoute.value, handler, name, dirPath)
  ) {
    router.replace(getHandlerLink(dirPath, handler, name))
  }
}

const {
  getEntryMenus,
  executeHandler,
  onRouteChanged,
  getViewHandlerSavedState,
  getViewHandlerShowing,
  hideViewHandler,
} = useEntryHandler(
  currentDirEntry,
  entries,
  handlerCtx,
  resolveHandlerByRoute,
  reloadEntryList,
  onHandlerBeingExecute,
  onHandlerBeingHide,
  onHandlerEntryBeingChanged
)

const progressBar = (v?: number | boolean) => store.setProgressBar(v)

const onEntryClicked = ({ entry }: EntryEventData) => {
  if (!entry) return
  if (entry.type === 'dir') {
    // path changed
    entries.value = undefined
    currentDirEntry.value = undefined
    return
  }
  // route change
}

const onEntryMenuClicked = ({ entry, menu }: EntryMenuClickData) => {
  entryMenuShowing.value = false
  executeHandler(menu.name, entry)
}

const showEntryMenu = ({ entry, event }: EntryEventData) => {
  const menu = getEntryMenus(
    selectedEntries.value.length > 0 ? [...selectedEntries.value] : entry!
  )
  if (!menu) return

  event && event.preventDefault()
  entryMenuData.value = menu
  entryMenuShowing.value = true
}

const entriesLoaded = ({
  entries: entries_,
  path: path_,
  entry: thisEntry,
}: EntriesLoadData) => {
  setTitle(path_)

  if (path_ !== path.value) {
    router.push(getDirLink(path_))
  }

  entries.value = entries_
  currentDirEntry.value = thisEntry

  selectedEntries.value.splice(0)
  onRouteChanged(router.currentRoute.value)
}

const onEntriesDragAction = async (data: EntryDragData) => {
  const to = typeof data.to === 'string' ? data.to : data.to.path

  if (data.action === 'link') {
    try {
      loading(true)
      await mountPaths(
        to,
        data.from.map((e) => ({
          path: e.path,
          name: e.name,
        }))
      )
    } catch (e: any) {
      alert(e.message)
      return
    } finally {
      loading()
    }
  } else {
    try {
      const executed = await copyOrMove(data.action === 'move', data.from, to)
      if (executed.length === 0) return
    } catch {
      return
    }
  }

  reloadEntryList()
}

const confirmUnsavedState = () => {
  if (getViewHandlerSavedState()) return Promise.resolve()
  return confirm(t('p.home.unsaved_confirm'))
}

const onWindowUnload = (e: BeforeUnloadEvent) => {
  if (getViewHandlerSavedState()) return
  e.preventDefault()
  e.returnValue = ''
}

const removeHandlerRoute = () => {
  if (getHistoryFlag()) {
    router.go(-1)
    return true
  } else {
    const route = router.currentRoute.value
    const dirRoute = getDirLink(path.value)
    if (route.fullPath !== dirRoute) {
      router.replace(dirRoute)
      return true
    }
  }
}

const focusOnEntry = (name: string, later?: boolean) => {
  entryListEl.value!.focusOnEntry(name, later)
}

const navigateToEntry = (entry: Entry) => {
  const targetPath = dir(entry.path)
  if (targetPath !== path.value) {
    router.push(getDirLink(targetPath))
    focusOnEntry(entry.name, true)
  } else {
    focusOnEntry(entry.name)
  }
}

const onKeyDown = async (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    if (!getViewHandlerShowing()) return
    e.stopPropagation()
    e.preventDefault()

    if (await hideViewHandler()) {
      removeHandlerRoute()
    }
  }
}

window.addEventListener('beforeunload', onWindowUnload)
window.addEventListener('keydown', onKeyDown)
</script>
<style lang="scss">
.entry-explorer {
  position: relative;
  margin: 0 auto 40px;
  max-width: 900px;
  padding-top: 72px;

  &.search-disabled {
    padding-top: 0;
  }
}

.search-panel-wrapper {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  z-index: 10;
}

.files-list {
  background-color: var(--primary-bg-color);
  padding: 16px 0;
  border-radius: 16px;
}

.page-footer {
  box-sizing: border-box;
  margin: 42px 0;
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

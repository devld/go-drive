<template>
  <div
    class="entry-list"
    :class="[validViewMode ? `entry-list--view-${validViewMode}` : '']"
  >
    <div class="entry-list__head">
      <PathBar
        :path="path"
        :get-link="getLink"
        @path-change="emit('update:path', $event)"
        @dragover="onDragOver"
        @drop="onDrop"
      />
      <div v-if="showToggles" class="entry-list__toggles">
        <button
          class="plain-button view-model-toggle"
          :title="
            validViewMode === 'list'
              ? $t('app.toggle_to_thumbnail')
              : $t('app.toggle_to_list')
          "
          @click="toggleViewMode"
        >
          <Icon
            :svg="validViewMode === 'list' ? '#icon-gallery' : '#icon-list'"
          />
        </button>
        <SimpleDropdown v-model="sortDropdownShowing">
          <span :title="$t('app.toggle_sort')">
            <Icon svg="#icon-sort" />
          </span>
          <template #dropdown>
            <ul class="sort-modes">
              <li
                v-for="s in sortModes"
                :key="s.key"
                class="sort-mode"
                :class="{ active: validSort === s.key }"
                @click="setSortBy(s.key)"
              >
                {{ $t(s.name) }}
              </li>
            </ul>
          </template>
        </SimpleDropdown>
      </div>
    </div>
    <ul class="entry-list__entries">
      <li v-if="!isRootPath" class="entry-list__item">
        <EntryLink
          ref="parentEntryRef"
          :entry="parentDirEntry"
          :get-link="getLink"
          :draggable="draggable"
          @click="entryClicked"
          @dragstart="onDragStart"
          @dragover="onDragOver"
          @drop="onDrop"
        >
          <EntryItem
            :view-mode="validViewMode"
            :entry="parentDirEntry"
            :icon="selected.length > 0 ? '#icon-duigou' : undefined"
            :show-thumbnail="false"
            @icon-click="parentIconClicked($event)"
          />
        </EntryLink>
      </li>
      <li
        v-for="entry in sortedEntries"
        :key="entry.path"
        class="entry-list__item"
        :class="{ selected: selectionMap[entry.path] }"
      >
        <EntryLink
          :ref="addEntryRef"
          :entry="entry"
          :get-link="getLink"
          :data-name="entry.name"
          :draggable="draggable"
          @click="entryClicked"
          @menu="entryContextMenu"
          @dragstart="onDragStart"
          @dragover="onDragOver"
          @drop="onDrop"
        >
          <EntryItem
            :view-mode="validViewMode"
            :entry="entry"
            show-thumbnail
            @icon-click="iconClicked(entry, $event)"
          />
        </EntryLink>
      </li>
    </ul>
    <div v-if="sortedEntries.length === 0" class="entry-list__empty">
      {{ $t('app.empty_list') }}
    </div>
  </div>
</template>
<script setup lang="ts">
import { Entry } from '@/types'
import { isRootPath as isRootPathFn, mapOf, pathClean, pathJoin } from '@/utils'
import { useHotKey } from '@/utils/hooks/hotkey'
import {
  ComponentPublicInstance,
  computed,
  nextTick,
  onBeforeUpdate,
  ref,
  watch,
} from 'vue'
import type { EntryEventData, GetLinkFn, ListViewMode } from '.'
import EntryLink from './EntryLink.vue'
import { useEntryDarg, EntryDragData } from './useDrag'

const SORTS_METHOD: O<(a: Entry, b: Entry) => number> = {
  name_asc: (a, b) =>
    a.type.localeCompare(b.type) || a.name.localeCompare(b.name),
  name_desc: (a, b) =>
    -a.type.localeCompare(b.type) || -a.name.localeCompare(b.name),
  mod_time_asc: (a, b) =>
    a.type.localeCompare(b.type) ||
    a.modTime - b.modTime ||
    a.name.localeCompare(b.name),
  mod_time_desc: (a, b) =>
    -a.type.localeCompare(b.type) ||
    b.modTime - a.modTime ||
    a.name.localeCompare(b.name),
  size_asc: (a, b) =>
    a.type.localeCompare(b.type) ||
    a.size - b.size ||
    a.name.localeCompare(b.name),
  size_desc: (a, b) =>
    -a.type.localeCompare(b.type) ||
    b.size - a.size ||
    a.name.localeCompare(b.name),
}

const props = defineProps({
  path: {
    type: String,
    required: true,
  },
  entries: {
    type: Array as PropType<Entry[]>,
    required: true,
  },
  sort: {
    type: String,
    default: 'name_asc',
  },
  selectable: {
    type: [Boolean, Function] as PropType<boolean | ((e: Entry) => boolean)>,
  },
  selection: {
    type: Array as PropType<Entry[]>,
  },
  viewMode: {
    type: String as PropType<ListViewMode>,
    default: 'list',
  },
  showToggles: {
    type: Boolean,
  },
  getLink: {
    type: Function as PropType<GetLinkFn>,
  },
  draggable: {
    type: Boolean,
  },
})

const emit = defineEmits<{
  (e: 'entries-change', data: Entry[]): void
  (e: 'update:path', data: EntryEventData): void
  (e: 'update:viewMode', data: ListViewMode): void
  (e: 'entry-click', data: EntryEventData): void
  (e: 'entry-menu', data: EntryEventData): void
  (e: 'update:selection', data: Entry[]): void
  (e: 'update:sort', data: string): void
  (e: 'drag-action', data: EntryDragData): void
}>()

const selected = ref<Entry[]>([])
const sortDropdownShowing = ref(false)
const sortModes = Object.keys(SORTS_METHOD).map((key) => ({
  key,
  name: `app.sort.${key}`,
}))

const validViewMode = computed(
  () =>
    (['list', 'thumbnail'].find((e) => e === props.viewMode) ??
      'list') as ListViewMode
)

const validSort = computed(() => {
  const sort = props.sort
  return SORTS_METHOD[sort] ? sort : 'name_asc'
})

const parentEntryRef = ref(null)
let entriesRef: InstanceType<typeof EntryLink>[] = []

const addEntryRef = (el: Element | ComponentPublicInstance | null) => {
  if (el) entriesRef.push(el as InstanceType<typeof EntryLink>)
}
onBeforeUpdate(() => {
  entriesRef = []
})

const parentDirEntry = computed<Entry>(() => ({
  path: pathClean(pathJoin(props.path, '..')),
  name: '..',
  meta: { writable: true },
  size: -1,
  type: 'dir',
  modTime: -1,
}))

const sortedEntries = computed(() => {
  const sortMethod = SORTS_METHOD[validSort.value] || SORTS_METHOD.name_asc
  return [...props.entries].sort(sortMethod)
})

const isRootPath = computed(() => isRootPathFn(props.path))
const selectionMap = computed(() =>
  mapOf(selected.value, (entry) => entry.path)
)

watch(
  () => props.selection,
  () => {
    if (props.selection === selected.value) return
    selected.value = [...(props.selection || [])]
  },
  { immediate: true }
)

watch(sortedEntries, (entries) => emit('entries-change', entries))

const entryClicked = (e: EntryEventData) => {
  const event = e.event as MouseEvent | undefined
  if (event?.ctrlKey) {
    // toggle selection if ctrl key is pressed
    event.preventDefault()
    if (e.entry!.name === '..') return
    toggleSelect(e.entry!)
    return
  }
  if (event?.shiftKey && selected.value.length > 0) {
    // if shift key is pressed, select range
    event.preventDefault()
    if (e.entry!.name === '..') return
    toggleSelectRange(e.entry!)
    return
  }
  if (selected.value.length > 0) {
    event?.preventDefault()
    // if there are selections, clear it
    selected.value = []
    emit('update:selection', selected.value)
    return
  }
  emit('entry-click', e)
}

const entryContextMenu = (e: EntryEventData) => emit('entry-menu', e)

const toggleSelect = (entry: Entry) => {
  if (selectionMap.value[entry.path]) {
    selected.value.splice(
      selected.value.findIndex((e) => e.path === entry.path),
      1
    )
  } else {
    if (typeof props.selectable === 'function') {
      if (!props.selectable(entry)) return
    }
    selected.value.push(entry)
  }
  emit('update:selection', selected.value)
}

const toggleSelectRange = (entry: Entry) => {
  if (selected.value.length === 0) return
  const index = sortedEntries.value.findIndex((e) => e.path === entry.path)
  const lastIndex = sortedEntries.value.findIndex(
    (e) => e.path === selected.value[selected.value.length - 1].path
  )
  selected.value = sortedEntries.value.slice(
    Math.min(index, lastIndex),
    Math.max(index, lastIndex) + 1
  )
  emit('update:selection', selected.value)
}

const toggleSelectAll = () => {
  if (selected.value.length === props.entries.length) {
    selected.value.splice(0)
  } else {
    let entries = props.entries
    if (typeof props.selectable === 'function') {
      entries = entries.filter(props.selectable)
    }
    selected.value = [...entries]
  }
  emit('update:selection', selected.value)
}

const setViewMode = (mode: ListViewMode) => {
  emit('update:viewMode', mode)
  return mode
}

const toggleViewMode = () => {
  setViewMode(validViewMode.value === 'list' ? 'thumbnail' : 'list')
}

const iconClicked = (entry: Entry, e: MouseEvent) => {
  if (validViewMode.value !== 'list') return
  if (!props.selectable) return
  e.stopPropagation()
  e.preventDefault()
  toggleSelect(entry)
}

const parentIconClicked = (e: MouseEvent) => {
  if (validViewMode.value !== 'list') return
  if (!props.selectable) return
  e.stopPropagation()
  e.preventDefault()
  toggleSelectAll()
}

const setSortBy = (sort: string) => {
  emit('update:sort', sort)
  sortDropdownShowing.value = false
}

const focusOnEntry = async (name: string) => {
  await nextTick()
  let dom
  if (name === '..') dom = parentEntryRef.value
  else {
    dom = entriesRef.find((el) => el.$el?.dataset.name === name)?.$el
  }
  dom?.focus()
}

const { onDragStart, onDragOver, onDrop } = useEntryDarg(
  computed(() => props.draggable),
  selected,
  (d) => emit('drag-action', d)
)

useHotKey(toggleViewMode, 't')
useHotKey(
  (e) => {
    toggleSelectAll()
    e.preventDefault()
  },
  'a',
  { ctrl: true }
)

defineExpose({
  focusOnEntry,
  setSortBy,
  setViewMode,
  toggleViewMode,
})
</script>
<style lang="scss">
.entry-list {
  .entry-link {
    color: var(--primary-text-color);
  }
}

.entry-list__head {
  display: flex;
  margin-bottom: 16px;
  padding: 0 16px;

  .path-bar {
    flex: 1;
  }
}

.entry-list__toggles {
  margin-left: auto;

  .icon {
    color: var(--secondary-text-color);
  }

  .view-model-toggle {
    cursor: pointer;
    font-size: 16px;
  }

  .sort-modes {
    margin: 0;
    padding: 0;
  }

  .sort-mode {
    margin: 0;
    list-style-type: none;
    white-space: nowrap;
    padding: 6px 12px;
    cursor: pointer;
    font-size: 14px;

    &:hover {
      background-color: var(--hover-bg-color);
    }

    &.active {
      background-color: var(--select-bg-color);
    }
  }
}

.entry-list--view-thumbnail {
  .entry-list__entries {
    display: flex;
    flex-wrap: wrap;

    & > li {
      width: 16.666%;
      margin-bottom: 10px;
    }

    .entry-link {
      display: block;
      height: 100%;
    }
  }

  @media screen and (max-width: 800px) {
    .entry-list__entries > li {
      width: 25%;
    }
  }

  @media screen and (max-width: 500px) {
    .entry-list__entries > li {
      width: 33.333%;
    }
  }

  @media screen and (max-width: 320px) {
    .entry-list__entries > li {
      width: 50%;
    }
  }
}

.entry-list__entries {
  margin: 0;
  padding: 0;

  & > li {
    margin: 0;
    padding: 0;
    list-style-type: none;
  }
}

.entry-list__item {
  animation: fade-in 0.3s;

  & > .entry-link {
    display: block;
    text-decoration: none;

    &:focus {
      background-color: var(--focus-bg-color);
    }

    &:hover {
      background-color: var(--hover-bg-color);
    }
  }

  &.selected > .entry-link {
    background-color: var(--select-bg-color);
  }
}

.entry-list__empty {
  user-select: none;
  -webkit-user-select: none;
  text-align: center;
  padding: 32px 0;
  color: var(--secondary-text-color);
}
</style>

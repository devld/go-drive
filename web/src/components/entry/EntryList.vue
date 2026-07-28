<template>
  <div
    class="entry-list"
    :class="[validViewMode ? `entry-list--view-${validViewMode}` : '']"
  >
    <div class="entry-list__head">
      <PathBar
        :path="path"
        :get-link="getLink"
        :drop-state="dragState"
        @path-change="emit('update:path', $event)"
        @dragover="onDragOver"
        @dragleave="onDragLeave"
        @drop="onDrop"
      />
      <div v-if="showToggles" class="entry-list__toggles">
        <button
          class="plain-button view-model-toggle"
          data-ui="button"
          data-variant="plain"
          :title="
            validViewMode === 'list'
              ? $t('app.toggle_to_thumbnail')
              : $t('app.toggle_to_list')
          "
          @click="toggleViewMode"
        >
          <Icon :name="validViewMode === 'list' ? 'grid' : 'list'" />
        </button>
        <SimpleDropdown v-model="sortDropdownShowing">
          <span :title="$t('app.toggle_sort')">
            <Icon name="sort" />
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
          :class="getEntryDropTargetClass(dragState, parentDirEntry.path)"
          @click="entryClicked"
          @dragstart="onDragStart"
          @dragend="onDragEnd"
          @dragover="onDragOver"
          @dragleave="onDragLeave"
          @drop="onDrop"
        >
          <EntryItem
            :view-mode="validViewMode"
            :entry="parentDirEntry"
            :icon="selected.length > 0 ? 'check' : undefined"
            :show-thumbnail="false"
            @icon-click="parentIconClicked($event)"
          />
        </EntryLink>
      </li>
      <li
        v-for="entry in sortedEntries"
        :key="entry.path"
        class="entry-list__item"
        :class="{
          selected: selectionMap[entry.path],
          dragging: dragSourceMap[entry.path],
        }"
      >
        <EntryLink
          :ref="addEntryRef"
          :entry="entry"
          :get-link="getLink"
          :data-name="entry.name"
          :draggable="draggable"
          :class="getEntryDropTargetClass(dragState, entry.path)"
          @click="entryClicked"
          @dragstart="onDragStart"
          @dragend="onDragEnd"
          @dragover="onDragOver"
          @dragleave="onDragLeave"
          @drop="onDrop"
        >
          <EntryItem
            :view-mode="validViewMode"
            :entry="entry"
            show-thumbnail
            @icon-click="iconClicked(entry, $event)"
          />
        </EntryLink>
        <button
          v-if="showMenuButton"
          class="entry-list__menu-button plain-button"
          data-ui="button"
          data-variant="plain"
          type="button"
          :title="$t('app.entry_actions')"
          :aria-label="$t('app.entry_actions')"
          @click="entryMenuClicked(entry, $event)"
        >
          <Icon name="menu-dots" />
        </button>
      </li>
    </ul>
    <div v-if="sortedEntries.length === 0" class="entry-list__empty">
      {{ $t('app.empty_list') }}
    </div>
    <EntryDragStatus :state="dragState" />
  </div>
</template>
<script setup lang="ts">
import { Entry } from '@/types'
import { isRootPath as isRootPathFn, mapOf, pathClean, pathJoin } from '@/utils'
import { useHotKey } from '@/utils/hooks/hotkey'
import { isPrimaryModifierPressed } from '@/utils/platform'
import {
  ComponentPublicInstance,
  computed,
  nextTick,
  onBeforeUpdate,
  ref,
  watch,
} from 'vue'
import type { EntryEventData, GetLinkFn, ListViewMode } from '.'
import EntryDragStatus from './EntryDragStatus.vue'
import EntryLink from './EntryLink.vue'
import { SORTS_METHOD, sortModes } from './sort'
import {
  EntryDragData,
  getEntryDropTargetClass,
  useEntryDrag,
} from './useDrag'

const props = defineProps({
  path: {
    type: String,
    required: true,
  },
  currentEntry: {
    type: Object as PropType<Entry>,
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
  showMenuButton: {
    type: Boolean,
    default: true,
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
const dragSourceMap = computed(() =>
  mapOf(dragState.value.entries, (entry) => entry.path)
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
  if (event && isPrimaryModifierPressed(event)) {
    // Toggle selection with Command on macOS and Ctrl elsewhere.
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

const entryMenuClicked = (entry: Entry, event: MouseEvent) => {
  emit('entry-menu', { entry, event })
}

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
  if (e.shiftKey && selected.value.length > 0) {
    toggleSelectRange(entry)
  } else {
    toggleSelect(entry)
  }
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

const {
  dragState,
  onDragStart,
  onDragOver,
  onDragLeave,
  onDrop,
  onDragEnd,
} = useEntryDrag(
  computed(() => props.draggable),
  selected,
  computed(() => props.currentEntry),
  (d) => emit('drag-action', d),
  (entries) => emit('update:selection', entries),
  (entries) =>
    entries.flatMap((entry) => {
      const link = entriesRef.find(
        (element) => element.$el?.dataset.name === entry.name
      )
      const item = link?.$el?.closest('.entry-list__item')
      return item instanceof HTMLElement ? [item] : []
    })
)

useHotKey(toggleViewMode, 't')
useHotKey(
  (e) => {
    toggleSelectAll()
    e.preventDefault()
  },
  'a',
  { primary: true }
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
    color: var(--color-text);
    transition:
      background-color var(--motion-duration-fast)
        var(--motion-easing-standard),
      box-shadow var(--motion-duration-fast) var(--motion-easing-standard),
      opacity var(--motion-duration-fast) var(--motion-easing-standard);
  }

  .entry-link--drop-valid {
    background-color: var(--color-bg-selected) !important;
    box-shadow: inset 0 0 0 2px var(--color-accent);
  }

  .entry-link--drop-invalid {
    background-color: var(--color-bg-invalid) !important;
    box-shadow: inset 0 0 0 2px var(--color-danger);
    cursor: not-allowed;
  }

  .entry-link--drop-move {
    cursor: move;
  }

  .entry-link--drop-copy {
    cursor: copy;
  }

  .entry-link--drop-link {
    cursor: alias;
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
  display: flex;
  gap: 8px;
  margin-left: auto;
  align-items: center;

  .icon {
    color: var(--color-text-muted);
  }

  .view-model-toggle {
    cursor: pointer;
    font-size: 16px;
  }

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
    background-color: var(--color-bg-hover);
  }

  &.active {
    background-color: var(--color-bg-selected);
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
  position: relative;
  animation: fade-in 0.3s;

  & > .entry-link {
    display: block;
    text-decoration: none;

    &:focus {
      background-color: var(--color-bg-focus);
    }

    &:hover {
      background-color: var(--color-bg-hover);
    }
  }

  &.selected > .entry-link {
    background-color: var(--color-bg-selected);
  }

  &.dragging > .entry-link {
    opacity: 0.55;
  }
}

.entry-drag-image {
  position: fixed;
  z-index: 10000;
  pointer-events: none;
}

.entry-drag-image__stack {
  position: absolute;
  filter: drop-shadow(0 1px 3px rgba(0, 0, 0, 0.06));
}

.entry-drag-image__layer {
  position: absolute;
  box-sizing: border-box;
  border-radius: 8px;
  opacity: 0.97;
}

.entry-drag-image__item {
  position: relative;
  margin: 0;
  box-sizing: border-box;
  overflow: hidden;
  animation: none;
  border-radius: 8px;
  background-color: var(--color-bg-surface);
}

.entry-list__menu-button {
  position: absolute;
  right: 12px;
  top: 50%;
  width: 32px;
  height: 32px;
  padding: 0;
  transform: translateY(-50%);
  border-radius: 50%;
  color: var(--color-text-muted);
  cursor: pointer;
  font-size: 18px;
  line-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;

  &:hover,
  &:focus-visible {
    background-color: var(--color-bg-hover);
  }

  .icon {
    vertical-align: middle;
    color: var(--color-text-muted);
  }

  &:hover .icon,
  &:focus-visible .icon {
    color: var(--color-text);
  }
}

.entry-list--view-list .entry-item {
  padding-right: 56px;
}

.entry-list--view-thumbnail {
  .entry-list__menu-button {
    top: auto;
    right: 12px;
    bottom: 8px;
    transform: none;
    font-size: 16px;
  }

  .entry-item__info {
    display: block;
    padding-right: 28px;
  }
}

.entry-list__empty {
  -webkit-user-select: none;
  user-select: none;
  text-align: center;
  padding: 32px 0;
  color: var(--color-text-muted);
}
</style>

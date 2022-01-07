<template>
  <div
    class="entry-list"
    :class="[validViewMode ? `entry-list--view-${validViewMode}` : '']"
  >
    <div class="entry-list__head">
      <path-bar
        :path="path"
        :get-link="getLink"
        @path-change="emit('update:path', $event)"
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
          <i-icon
            :svg="validViewMode === 'list' ? '#icon-gallery' : '#icon-list'"
          />
        </button>
        <simple-dropdown v-model="sortDropdownShowing">
          <span :title="$t('app.toggle_sort')">
            <i-icon svg="#icon-sort" />
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
        </simple-dropdown>
      </div>
    </div>
    <ul class="entry-list__entries">
      <li v-if="!isRootPath" class="entry-list__item">
        <entry-link
          ref="parentEntryRef"
          :entry="parentDirEntry"
          :get-link="getLink"
          @click="entryClicked"
        >
          <entry-item
            :view-mode="validViewMode"
            :entry="parentDirEntry"
            :icon="selected.length > 0 ? '#icon-duigou' : undefined"
            :show-thumbnail="false"
            @icon-click="parentIconClicked($event)"
          />
        </entry-link>
      </li>
      <li
        v-for="entry in sortedEntries"
        :key="entry.path"
        class="entry-list__item"
        :class="{ selected: selectionMap[entry.path] }"
      >
        <entry-link
          :ref="addEntryRef"
          :entry="entry"
          :get-link="getLink"
          :data-name="entry.name"
          @click="entryClicked"
          @menu="entryContextMenu"
        >
          <entry-item
            :view-mode="validViewMode"
            :entry="entry"
            show-thumbnail
            @icon-click="iconClicked(entry, $event)"
          />
        </entry-link>
      </li>
    </ul>
    <div v-if="sortedEntries.length === 0" class="entry-list__empty">
      {{ $t('app.empty_list') }}
    </div>
  </div>
</template>
<script setup>
import { isRootPath as isRootPathFn, mapOf, pathClean, pathJoin } from '@/utils'
import { useHotKey } from '@/utils/hooks/hotkey'
import { computed, nextTick, onBeforeUpdate, ref, watch } from 'vue'
import IIcon from './IIcon.vue'

const SORTS_METHOD = {
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
    type: Array,
    required: true,
  },
  sort: {
    type: String,
    default: 'name_asc',
  },
  selectable: {
    type: [Boolean, Function],
  },
  selection: {
    type: Array,
  },
  viewMode: {
    type: String,
    default: 'list',
  },
  showToggles: {
    type: Boolean,
  },
  getLink: {
    type: Function,
  },
})

const emit = defineEmits([
  'update:path',
  'update:viewMode',
  'entry-click',
  'entry-menu',
  'update:selection',
  'update:sort',
])

const selected = ref([])
const sortDropdownShowing = ref(false)
const sortModes = Object.keys(SORTS_METHOD).map((key) => ({
  key,
  name: `app.sort.${key}`,
}))

const validViewMode = computed(
  () => ['list', 'thumbnail'].find((e) => e === props.viewMode) || 'list'
)

const validSort = computed(() => {
  const sort = props.sort
  return SORTS_METHOD[sort] ? sort : 'name_asc'
})

const parentEntryRef = ref(null)
let entriesRef = []

const addEntryRef = (el) => {
  if (el) entriesRef.push(el)
}
onBeforeUpdate(() => {
  entriesRef = []
})

const parentDirEntry = computed(() => ({
  path: pathClean(pathJoin(props.path, '..')),
  name: '..',
  meta: {},
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

const entryClicked = (e) => emit('entry-click', e)

const entryContextMenu = (e) => emit('entry-menu', e)

const toggleSelect = (entry) => {
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

const setViewMode = (mode) => {
  emit('update:viewMode', mode)
  return mode
}

const toggleViewMode = () => {
  setViewMode(validViewMode.value === 'list' ? 'thumbnail' : 'list')
}

const iconClicked = (entry, e) => {
  if (validViewMode.value !== 'list') return
  if (!props.selectable) return
  e.stopPropagation()
  e.preventDefault()
  toggleSelect(entry)
}

const parentIconClicked = (e) => {
  if (validViewMode.value !== 'list') return
  if (!props.selectable) return
  e.stopPropagation()
  e.preventDefault()
  toggleSelectAll()
}

const setSortBy = (sort) => {
  emit('update:sort', sort)
  sortDropdownShowing.value = false
}

const focusOnEntry = async (name) => {
  await nextTick()
  let dom
  if (name === '..') dom = parentEntryRef.value
  else {
    dom = entriesRef.find((el) => el.$el?.dataset.name === name)?.$el
  }
  dom?.focus()
}

useHotKey(() => {
  toggleViewMode()
}, 't')

defineExpose({ focusOnEntry, setSortBy, setViewMode, toggleViewMode })
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
  text-align: center;
  padding: 32px 0;
  color: var(--secondary-text-color);
}
</style>

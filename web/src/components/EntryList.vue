<template>
  <div
    class="entry-list"
    :class="[viewMode ? `entry-list--view-${viewMode}` : '']"
  >
    <div class="entry-list__head">
      <path-bar :path="path" @path-change="$emit('path-change', $event)" />
      <div class="entry-list__toggles" v-if="showToggles">
        <button
          class="plain-button view-model-toggle"
          :title="
            viewMode === 'list'
              ? $t('app.toggle_to_thumbnail')
              : $t('app.toggle_to_list')
          "
          @click="
            $emit('update:viewMode', viewMode === 'list' ? 'thumbnail' : 'list')
          "
        >
          <i-icon :svg="viewMode === 'list' ? '#icon-gallery' : '#icon-list'" />
        </button>
        <simple-dropdown v-model="sortDropdownShowing">
          <span :title="$t('app.toggle_sort')">
            <i-icon svg="#icon-sort" />
          </span>
          <ul slot="dropdown" class="sort-modes">
            <li
              class="sort-mode"
              :class="{ active: sort === s.key }"
              v-for="s in sortModes"
              :key="s.key"
              @click="setSortBy(s.key)"
            >
              {{ $t(s.name) }}
            </li>
          </ul>
        </simple-dropdown>
      </div>
    </div>
    <ul class="entry-list__entries">
      <li class="entry-list__item" v-if="!isRootPath">
        <entry-link
          ref="parentEntry"
          :entry="parentDirEntry"
          @click="entryClicked"
        >
          <entry-item
            :view-mode="viewMode"
            :entry="parentDirEntry"
            :icon="selected.length > 0 ? '#icon-duigou' : undefined"
            @icon-click="parentIconClicked($event)"
            :show-thumbnail="false"
          />
        </entry-link>
      </li>
      <li
        class="entry-list__item"
        v-for="entry in sortedEntries"
        :key="entry.path"
        :class="{ selected: selectionMap[entry.path] }"
      >
        <entry-link
          ref="entries"
          :entry="entry"
          @click="entryClicked"
          @menu="entryContextMenu"
        >
          <entry-item
            :view-mode="viewMode"
            :entry="entry"
            @icon-click="iconClicked(entry, $event)"
            show-thumbnail
          />
        </entry-link>
      </li>
    </ul>
    <div class="entry-list__empty" v-if="sortedEntries.length === 0">
      {{ $t("app.empty_list") }}
    </div>
  </div>
</template>
<script>
import { pathJoin, pathClean, isRootPath, mapOf } from '@/utils'
import IIcon from './IIcon.vue'

const SORTS_METHOD = {
  name_asc: (a, b) => a.type.localeCompare(b.type) || a.name.localeCompare(b.name),
  name_desc: (a, b) => -a.type.localeCompare(b.type) || -a.name.localeCompare(b.name),
  mod_time_asc: (a, b) => a.type.localeCompare(b.type) || a.mod_time - b.mod_time || a.name.localeCompare(b.name),
  mod_time_desc: (a, b) => -a.type.localeCompare(b.type) || b.mod_time - a.mod_time || a.name.localeCompare(b.name),
  size_asc: (a, b) => a.type.localeCompare(b.type) || a.size - b.size || a.name.localeCompare(b.name),
  size_desc: (a, b) => -a.type.localeCompare(b.type) || b.size - a.size || a.name.localeCompare(b.name)
}

export default {
  components: { IIcon },
  name: 'EntryList',
  props: {
    path: {
      type: String,
      required: true
    },
    entries: {
      type: Array,
      required: true
    },
    sort: {
      type: String,
      default: 'name_asc'
    },
    selectable: {
      type: [Boolean, Function]
    },
    selection: {
      type: Array
    },
    viewMode: {
      type: String,
      default: 'list'
    },
    showToggles: {
      type: Boolean
    }
  },
  watch: {
    selection: {
      immediate: true,
      handler (val) {
        if (val === this.selection) return
        this.selection = [...(val || [])]
      }
    }
  },
  data () {
    return {
      selected: [],

      sortDropdownShowing: false,
      sortModes: [
        { key: 'name_asc', name: 'app.sort.name_asc' },
        { key: 'name_desc', name: 'app.sort.name_desc' },
        { key: 'mod_time_asc', name: 'app.sort.mod_time_asc' },
        { key: 'mod_time_desc', name: 'app.sort.mod_time_desc' },
        { key: 'size_asc', name: 'app.sort.size_asc' },
        { key: 'size_desc', name: 'app.sort.size_desc' }
      ]
    }
  },
  computed: {
    parentDirEntry () {
      return {
        path: pathClean(pathJoin(this.path, '..')),
        name: '..',
        meta: {},
        size: -1,
        type: 'dir',
        mod_time: -1
      }
    },
    sortedEntries () {
      const sortMethod = SORTS_METHOD[this.sort] || SORTS_METHOD.name_asc
      return [...this.entries].sort(sortMethod)
    },
    isRootPath () {
      return isRootPath(this.path)
    },
    selectionMap () {
      return mapOf(this.selected, e => e.path)
    }
  },
  methods: {
    entryClicked (e) {
      this.$emit('entry-click', e)
    },
    entryContextMenu (e) {
      this.$emit('entry-menu', e)
    },
    iconClicked (entry, e) {
      if (this.viewMode !== 'list') return
      if (!this.selectable) return
      e.stopPropagation()
      e.preventDefault()
      this.toggleSelect(entry)
    },
    parentIconClicked (e) {
      if (this.viewMode !== 'list') return
      if (!this.selectable) return
      e.stopPropagation()
      e.preventDefault()
      this.toggleSelectAll()
    },
    toggleSelect (entry) {
      if (this.selectionMap[entry.path]) {
        this.selected.splice(this.selected.findIndex(e => e.path === entry.path), 1)
      } else {
        if (typeof (this.selectable) === 'function') {
          if (!this.selectable(entry)) return
        }
        this.selected.push(entry)
      }
      this.$emit('update:selection', this.selected)
    },
    toggleSelectAll () {
      if (this.selected.length === this.entries.length) {
        this.selected.splice(0)
      } else {
        let entries = this.entries
        if (typeof (this.selectable) === 'function') {
          entries = entries.filter(this.selectable)
        }
        this.selected = [...entries]
      }
      this.$emit('update:selection', this.selected)
    },
    setSortBy (sort) {
      this.$emit('update:sort', sort)
      this.sortDropdownShowing = false
    },
    focusOnEntry (name) {
      let dom
      if (name === '..') dom = this.$refs.parentEntry
      else {
        const index = this.sortedEntries.findIndex(e => e.name === name)
        if (index >= 0) dom = this.$refs.entries[index]
      }
      dom = (dom && dom.$el) || dom
      this.$nextTick(() => {
        dom && dom.focus()
      })
    }
  }
}
</script>
<style lang="scss">
.entry-list {
  .entry-link {
    @include var(color, primary-text-color);
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
    @include var(color, secondary-text-color);
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
    padding: 0;
    list-style-type: none;
    white-space: nowrap;
    padding: 6px 12px;
    cursor: pointer;
    font-size: 14px;

    &:hover {
      @include var(background-color, hover-bg-color);
    }

    &.active {
      @include var(background-color, select-bg-color);
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
      @include var(background-color, focus-bg-color);
    }

    &:hover {
      @include var(background-color, hover-bg-color);
    }
  }

  &.selected > .entry-link {
    @include var(background-color, select-bg-color);
  }
}

.entry-list__empty {
  user-select: none;
  text-align: center;
  padding: 32px 0;
  @include var(color, secondary-text-color);
}
</style>

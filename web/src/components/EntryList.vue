<template>
  <div class="entry-list">
    <path-bar
      v-if="!isRootPath"
      :path="path"
      @path-change="$emit('path-change', $event)"
    />
    <ul class="entry-list__entries">
      <li class="entry-list__item" v-if="!isRootPath">
        <entry-link
          ref="parentEntry"
          :entry="parentDirEntry"
          @click="entryClicked"
        >
          <entry-item
            :entry="parentDirEntry"
            :icon="selected.length > 0 ? '#icon-duigou' : undefined"
            @icon-click="toggleSelectAll($event)"
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
            :entry="entry"
            @icon-click="toggleSelect(entry, $event)"
          />
        </entry-link>
      </li>
    </ul>
    <div class="entry-list__empty" v-if="sortedEntries.length === 0">
      Nothing here
    </div>
  </div>
</template>
<script>
import { pathJoin, pathClean, isRootPath, mapOf } from '@/utils'

const SORTS_METHOD = {
  default: (a, b) => {
    if (a.type === 'dir' && b.type !== 'dir') return -1
    if (a.type !== 'dir' && b.type === 'dir') return 1
    if (a.name > b.name) return 1
    else if (a.name < b.name) return -1
    return 0
  }
}

export default {
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
      default: 'default'
    },
    selectable: {
      type: [Boolean, Function]
    },
    selection: {
      type: Array
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
      selected: []
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
      const sortMethod = SORTS_METHOD[this.sort]
      return sortMethod ? [...this.entries].sort(sortMethod) : this.entries
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
    toggleSelect (entry, e) {
      if (!this.selectable) return
      e.stopPropagation()
      e.preventDefault()
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
    toggleSelectAll (e) {
      if (!this.selectable) return
      e.stopPropagation()
      e.preventDefault()
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
    focusOnEntry (name) {
      let dom
      if (name === '..') dom = this.$refs.parentEntry
      else {
        const index = this.sortedEntries.findIndex(e => e.name === name)
        if (index >= 0) dom = this.$refs.entries[index]
      }
      dom = (dom && dom.$el) || dom
      dom && dom.scrollIntoView()
      dom && dom.focus()
    }
  }
}
</script>
<style lang="scss">
.entry-list {
  .path-bar {
    margin-bottom: 16px;

    .entry-link {
      color: #000;
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

  & > a {
    display: block;
    text-decoration: none;
    color: #000;

    &:focus {
      background-color: rgba(0, 0, 0, 0.08);
    }

    &:hover {
      background-color: #f6f6f6;
    }
  }

  &.selected > a {
    background-color: #c1ecff;
  }
}

.entry-list__empty {
  user-select: none;
  text-align: center;
  padding: 32px 0;
  color: gray;
}
</style>

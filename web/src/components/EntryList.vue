<template>
  <div class="entry-list">
    <path-bar
      v-if="!isRootPath"
      :path="path"
      :entry-link="getEntryLink"
      @path-change="$emit('path-change', $event)"
    />
    <ul class="entry-list__entries">
      <li class="entry-list__item" v-if="!isRootPath">
        <a
          :href="getEntryLink(parentDirEntry)"
          @click="entryClicked(parentDirEntry, $event)"
          ref="parentEntry"
        >
          <entry-item :entry="parentDirEntry" />
        </a>
      </li>
      <li class="entry-list__item" v-for="entry in sortedEntries" :key="path + entry.name">
        <a :href="getEntryLink(entry)" @click="entryClicked(entry, $event)" ref="entries">
          <entry-item :entry="entry" />
        </a>
      </li>
    </ul>
  </div>
</template>
<script>
import { pathJoin, pathClean } from '@/utils'

const SORTS_METHOD = {
  default: (a, b) => {
    const aType = a.type === 'drive' ? 'dir' : a.type
    const bType = b.type === 'drive' ? 'dir' : b.type
    if (aType === 'dir' && bType !== 'dir') return -1
    if (aType !== 'dir' && bType === 'dir') return 1
    if (a.name > b.name) return 1
    else if (a.name < b.name) return -1
    return 0
  }
}

const parentDirEntry = {
  name: '..',
  meta: {},
  size: -1,
  type: 'dir',
  created_at: -1,
  updated_at: -1
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
    entryLink: {
      type: Function
    }
  },
  data () {
    return {
      parentDirEntry
    }
  },
  computed: {
    sortedEntries () {
      const sortMethod = SORTS_METHOD[this.sort]
      return sortMethod ? [...this.entries].sort(sortMethod) : this.entries
    },
    isRootPath () {
      return this.path === '/'
    }
  },
  methods: {
    entryClicked (entry, e) {
      e.preventDefault()
      const path = pathClean(pathJoin(this.path, entry.name))
      if (entry.type === 'drive' || entry.type === 'dir') {
        this.$emit('path-change', path)
      } else if (entry.type === 'file') {
        this.$emit('open-file', { path, entry })
      }
    },
    getEntryLink (entry) {
      let link
      if (this.entryLink) {
        if (typeof (entry) === 'string') {
          link = this.entryLink(entry)
        } else {
          link = this.entryLink(entry, this.path)
        }
      }
      if (!link) link = 'javascript:;'
      return link
    },
    focusOnEntry (name) {
      let dom
      if (name === '..') dom = this.$refs.parentEntry
      else {
        const index = this.sortedEntries.findIndex(e => e.name === name)
        if (index >= 0) dom = this.$refs.entries[index]
      }
      dom && dom.focus()
    }
  }
}
</script>
<style lang="scss">
.entry-list {
  .path-bar {
    margin-bottom: 16px;
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
  animation: fadein 0.3s;

  & > a {
    display: block;
    text-decoration: none;
    color: unset;

    &:focus {
      background-color: rgba(0, 0, 0, 0.08);
    }

    &:hover {
      background-color: rgba(0, 0, 0, 0.08);
    }
  }
}
</style>

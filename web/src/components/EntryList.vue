<template>
  <div class="entry-list">
    <path-bar :path="path" @path-change="$emit('path-change', $event)" />
    <div class="entry-list__entries">
      <entry-item v-if="!isRootPath" :entry="parentDirEntry" @click="entryClicked" />
      <entry-item
        v-for="(entry, i) in sortedEntries"
        :key="i"
        :entry="entry"
        @click="entryClicked"
      />
    </div>
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
    entryClicked (entry) {
      const path = pathClean(pathJoin(this.path, entry.name))
      if (entry.type === 'drive' || entry.type === 'dir') {
        this.$emit('path-change', path)
      } else if (entry.type === 'file') {
        this.$emit('open-file', path)
      }
    }
  }
}
</script>
<style lang="scss">
.entry-list .path-bar {
  margin-bottom: 16px;
}
</style>

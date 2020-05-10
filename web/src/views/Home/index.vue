<template>
  <div class="home">
    <main class="files-list">
      <entry-list-view :path="path" @entries-load="entriesLoaded" :entry-link="makeEntryRouteLink" />
    </main>
    <footer>
      <page-footer v-if="readmeContent" :readme="readmeContent" />
    </footer>
  </div>
</template>
<script>
import { getContent } from '@/api'
import PageFooter from './PageFooter.vue'
import { pathJoin, pathClean } from '@/utils'
import EntryListView from '@/views/EntryListView'

const README_FILENAME = 'readme.md'
const README_FAILED_CONTENT = '<p style="text-align: center;">Failed to load README.md</p>'

export default {
  name: 'Home',
  components: { EntryListView, PageFooter },
  data () {
    return {
      path: '/',
      readmeContent: ''
    }
  },
  beforeRouteUpdate (to, from, next) {
    this.path = '/' + to.params.path
    next()
  },
  created () {
    const path = this.$route.params.path
    this.path = '/' + (path || '')
  },
  methods: {
    entriesLoaded ({ entries, path }) {
      if (path !== this.path) {
        this.$router.push(`/files${path}`)
      }
      this.tryLoadReadme(entries)
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
        await this.loadReadme(readmeFound.name)
      } else {
        this.readmeContent = ''
      }
    },
    async loadReadme (name) {
      try {
        this.readmeContent = await getContent(pathJoin(this.path, name))
      } catch (e) {
        this.readmeContent = README_FAILED_CONTENT
      }
    },
    makeEntryRouteLink (entryOrPath, path) {
      if (typeof (entryOrPath) === 'string') {
        return `#/files${entryOrPath}`
      }
      if (entryOrPath.type === 'file') return
      return '#/files' + pathClean(pathJoin(path, entryOrPath.name))
    }
  }
}
</script>
<style lang="scss">
.home {
  max-width: 880px;
  margin: 42px auto;
  background-color: #fff;
  padding: 16px 0;
  border-radius: 16px;
}

@media screen and (max-width: 900px) {
  .home {
    margin: 16px;
  }
}
</style>

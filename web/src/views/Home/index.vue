<template>
  <div class="home">
    <div class="files-list">
      <entry-list :path="path" :entries="entries" @path-change="pathChange" />
    </div>
  </div>
</template>
<script>
import { listEntries } from '@/api'

export default {
  name: 'Home',
  data () {
    return {
      path: '/',
      entries: []
    }
  },
  beforeRouteUpdate (to, from, next) {
    this.path = '/' + to.params.path
    this.loadEntries()
    next()
  },
  created () {
    const path = this.$route.params.path
    this.path = '/' + (path || '')
    this.loadEntries()
  },
  methods: {
    pathChange (path) {
      if (path !== this.path) {
        this.$router.replace(`/files${path}`)
      }
    },
    async loadEntries () {
      this.entries = await listEntries(this.path)
    }
  }
}
</script>
<style lang="scss">
.home {
  max-width: 980px;
  margin: 42px auto;
  background-color: #fff;
  padding: 16px 0;
  border-radius: 16px;
}
</style>

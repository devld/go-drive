<template>
  <ul class="path-bar">
    <li class="path-bar__segment" v-for="s in segments" :key="s.path">
      <a
        class="path-bar__path"
        :href="entryLink ? entryLink(s.path) : 'javascript:;'"
        @click="pathChange(s)"
      >{{ s.name }}</a>
    </li>
  </ul>
</template>
<script>
export default {
  name: 'PathBar',
  model: {
    prop: 'path',
    event: 'path-change'
  },
  props: {
    path: {
      type: String,
      required: true
    },
    entryLink: {
      type: Function
    }
  },
  computed: {
    segments () {
      if (!this.path) return []
      const ss = this.path.replace(/\/+/g, '/').split('/').filter(s => !!s)
      const pathSegments = [{ name: '/', path: '/' }]
      ss.forEach((s, i) => {
        pathSegments.push({ name: s, path: '/' + ss.slice(0, i + 1).join('/') })
      })
      return pathSegments
    }
  },
  methods: {
    pathChange (s) {
      if (this.entryLink) return
      this.$emit('path-change', s.path)
    }
  }
}
</script>
<style lang="scss">
.path-bar {
  margin: 0;
  padding: 0 16px;
  list-style-type: none;
}

.path-bar__segment {
  margin: 0;
  padding: 0;
  display: inline-block;

  &:not(:last-child) {
    &::after {
      content: ">";
      margin: 0 0.5em;
      color: #888;
    }
  }
}

.path-bar__path {
  cursor: pointer;
  text-decoration: none;
  color: unset;
}
</style>

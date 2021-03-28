<template>
  <ul class="path-bar">
    <li class="path-bar__segment" v-for="s in segments" :key="s.path">
      <entry-link class="path-bar__path" :path="s.path" @click="pathChange">{{
        s.name
      }}</entry-link>
    </li>
  </ul>
</template>
<script>
export default {
  name: 'PathBar',
  model: {
    prop: 'path',
    event: 'path-change',
  },
  props: {
    path: {
      type: String,
      required: true,
    },
  },
  computed: {
    segments() {
      const ss = this.path.replace(/\/+/g, '/').split('/').filter(Boolean)
      const pathSegments = [{ name: this.$t('app.root_path'), path: '' }]
      ss.forEach((s, i) => {
        pathSegments.push({ name: s, path: ss.slice(0, i + 1).join('/') })
      })
      return pathSegments
    },
  },
  methods: {
    pathChange(e) {
      this.$emit('path-change', e)
    },
  },
}
</script>
<style lang="scss">
.path-bar {
  margin: 0;
  padding: 0;
  list-style-type: none;
}

.path-bar__segment {
  margin: 0;
  padding: 0;
  display: inline-block;

  &:not(:last-child) {
    &::after {
      content: '>';
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

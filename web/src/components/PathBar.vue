<template>
  <ul class="path-bar">
    <li class="path-bar__segment" v-for="(s, i) in segments" :key="i">
      <span class="path-bar__path" @click="pathChange(i)">{{ s }}</span>
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
    }
  },
  computed: {
    segments () {
      const segments = this.path.replace(/\/+/g, '/').split('/').filter(s => !!s)
      segments.splice(0, 0, '/')
      return segments
    }
  },
  methods: {
    pathChange (i) {
      const path = this.segments.slice(0, i + 1).join('/').replace(/\/+/g, '/')
      this.$emit('path-change', path)
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
}
</style>

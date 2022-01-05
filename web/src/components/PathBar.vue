<template>
  <ul class="path-bar">
    <li v-for="s in segments" :key="s.path" class="path-bar__segment">
      <entry-link
        class="path-bar__path"
        :path="s.path"
        :get-link="getLink"
        @click="pathChange"
        >{{ s.name }}</entry-link
      >
    </li>
  </ul>
</template>
<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  path: {
    type: String,
    required: true,
  },
  getLink: {
    type: Function,
  },
})

const { t } = useI18n()

const emit = defineEmits(['update:path'])

const segments = computed(() => {
  const ss = props.path.replace(/\/+/g, '/').split('/').filter(Boolean)
  const pathSegments = [{ name: t('app.root_path'), path: '' }]
  ss.forEach((s, i) => {
    pathSegments.push({ name: s, path: ss.slice(0, i + 1).join('/') })
  })
  return pathSegments
})

const pathChange = (e) => emit('update:path', e)
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

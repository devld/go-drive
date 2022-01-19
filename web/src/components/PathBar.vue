<template>
  <ul class="path-bar">
    <li v-for="s in segments" :key="s.path" class="path-bar__segment">
      <entry-link
        class="path-bar__path"
        :path="s.path"
        :get-link="getLink"
        @click="pathChange"
      >
        <slot v-if="isRootPath(s.path)" name="root" :item="s">{{
          $t('app.root_path')
        }}</slot>
        <slot v-else name="item" :item="s">
          {{ s.name }}
        </slot>
      </entry-link>
    </li>
  </ul>
</template>
<script setup>
import { isRootPath } from '@/utils'
import { computed } from 'vue'

const props = defineProps({
  path: {
    type: String,
    required: true,
  },
  getLink: {
    type: Function,
  },
})

const emit = defineEmits(['update:path'])

const segments = computed(() => {
  const ss = props.path.replace(/\/+/g, '/').split('/').filter(Boolean)
  const pathSegments = [{ name: '', path: '' }]
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

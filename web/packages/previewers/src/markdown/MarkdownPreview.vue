<template>
  <article class="preview-markdown markdown-body" v-html="html" />
</template>
<script setup lang="ts">
import { computed } from 'vue'
import renderMarkdown from './marked'

const props = withDefaults(
  defineProps<{
    content?: string
  }>(),
  { content: '' }
)

const html = computed(() => renderMarkdown(props.content))
</script>
<style lang="scss">
.preview-markdown {
  box-sizing: border-box;
  min-height: 100%;
  padding: 24px;
  overflow: auto;
}
</style>

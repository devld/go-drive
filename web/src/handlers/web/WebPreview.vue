<template>
  <div class="web-preview">
    <iframe
      class="web-preview__frame"
      :srcdoc="documentContent"
      sandbox=""
      referrerpolicy="no-referrer"
      title="File preview"
    />
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    content?: string
    type?: 'html' | 'svg'
    baseUrl?: string
  }>(),
  {
    content: '',
    type: 'html',
  }
)

const escapeAttribute = (value: string) =>
  value.replace(/&/g, '&amp;').replace(/"/g, '&quot;')

const documentContent = computed(() => {
  const base = props.baseUrl
    ? `<base href="${escapeAttribute(props.baseUrl)}">`
    : ''
  if (props.type === 'svg') {
    return `<!doctype html><html><head><meta charset="utf-8">${base}<style>html,body{margin:0;min-height:100%;display:flex;align-items:center;justify-content:center;background:transparent}svg{max-width:100%;max-height:100vh}</style></head><body>${props.content}</body></html>`
  }
  return `<!doctype html><html><head><meta charset="utf-8">${base}</head><body>${props.content}</body></html>`
})
</script>
<style lang="scss">
.web-preview {
  width: 100%;
  height: 100%;
  overflow: hidden;
  background: #fff;
}

.web-preview__frame {
  display: block;
  width: 100%;
  height: 100%;
  border: 0;
  background: #fff;
}
</style>

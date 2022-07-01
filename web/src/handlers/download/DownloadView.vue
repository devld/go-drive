<template>
  <div class="download-view-page">
    <h1 class="page-title">
      <span>{{ $t('hv.download.download') }}</span>
      <button class="plain-button close-button" @click="emit('close')">
        <Icon svg="#icon-close" />
      </button>
    </h1>
    <div class="page-content">
      <template v-if="singleEntry">
        <EntryIcon :entry="singleEntry" />
        <h2 class="filename">{{ singleEntry.name }}</h2>
        <a
          class="download-button"
          target="_blank"
          rel="noreferrer noopener nofollow"
          :download="filename(singleEntry.path)"
          :href="fileUrl(singleEntry.path, singleEntry.meta)"
        >
          {{ $t('hv.download.download') }}
          <span v-if="singleEntry.size >= 0" class="file-size">{{
            formatBytes(singleEntry.size)
          }}</span>
        </a>
      </template>
      <template v-else>
        <textarea
          ref="linksEl"
          v-focus
          class="download-links"
          readonly
          :value="downloadLinks"
          @focus="downloadLinksFocus"
        ></textarea>
        <a class="download-button" href="javascript:;" @click="downloadFiles">
          {{ $t('hv.download.downloads', { n: entry.length }) }}
        </a>
      </template>
    </div>
  </div>
</template>
<script setup lang="ts">
import { filename, formatBytes } from '@/utils'
import { fileUrl } from '@/api'
import { computed, ref } from 'vue'
import { Entry } from '@/types'

const props = defineProps({
  entry: {
    type: Array as PropType<Entry[]>,
    required: true,
  },
  entries: { type: Array },
})
const emit = defineEmits<{ (e: 'close'): void }>()

const linksEl = ref<HTMLTextAreaElement | null>(null)

const singleEntry = computed(() => props.entry[0])

const downloadLinks = computed(() => {
  if (singleEntry.value) return ''
  return props.entry.map((e) => fileUrl(e.path, e.meta)).join('\n')
})

const downloadLinksFocus = () => {
  linksEl.value!.select()
  linksEl.value!.scrollTop = 0
  linksEl.value!.scrollLeft = 0
}

const downloadFiles = () => {
  props.entry.forEach((f) => {
    const a = document.createElement('a')
    a.rel = 'noreferrer noopener nofollow'
    a.href = fileUrl(f.path, f.meta)
    a.download = filename(f.path)
    a.click()
  })
}
</script>
<style lang="scss">
.download-view-page {
  position: relative;
  width: 300px;
  background-color: var(--secondary-bg-color);
  box-shadow: 0 0 6px rgba(0, 0, 0, 0.1);
  padding: 16px 16px 20px;

  .page-title {
    font-size: 28px;
    margin: 0 0 20px;
    font-weight: normal;
    user-select: none;

    .close-button {
      float: right;
    }
  }

  .entry-icon {
    width: 150px;
    height: 150px;
    font-size: 150px;
  }

  .page-content {
    text-align: center;
  }

  .filename {
    font-weight: normal;
    font-size: 18px;
    margin: 10px 0;
    word-break: break-all;
  }

  .download-button {
    display: inline-block;
    color: #fff;
    background-color: #00bfa5;
    text-decoration: none;
    padding: 10px 16px;
    margin-top: 16px;
    transition: 0.3s;
    user-select: none;

    &:hover {
      box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
    }
  }

  .download-links {
    width: 100%;
    min-height: 200px;
    max-height: 40vh;
    outline: none;
    border: none;
    resize: none;
    white-space: pre;
    overflow: auto;
  }
}
</style>

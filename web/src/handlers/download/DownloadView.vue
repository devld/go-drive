<template>
  <div
    class="download-view-page"
    data-ui="preview"
    data-handler="download"
  >
    <h1 class="page-title">
      <span>{{ $t('hv.download.download') }}</span>
      <SimpleButton
        class="close-button"
        variant="plain"
        icon="close"
        :aria-label="$t('dialog.base.close')"
        @click="emit('close')"
      />
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
          <span v-if="singleEntry.size >= 0">{{
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
        <a class="download-button" href="javascript:;" @click="downloadSelected">
          {{ $t('hv.download.downloads', { n: entry.length }) }}
        </a>
      </template>
    </div>
  </div>
</template>
<script lang="ts">
import { fileUrl } from '@/api'
import { Entry } from '@/types'
import { filename } from '@/utils'
import { triggerDownloadFile } from '@go-drive/utils'

export function downloadFiles(entries: Entry[]) {
  entries.forEach((file) => {
    triggerDownloadFile(fileUrl(file.path, file.meta), filename(file.path))
  })
}
</script>
<script setup lang="ts">
import { formatBytes } from '@/utils'
import { computed, ref } from 'vue'

const props = defineProps({
  entry: {
    type: Array as PropType<Entry[]>,
    required: true,
  },
  entries: { type: Array },
})
const emit = defineEmits<{ (e: 'close'): void }>()

const linksEl = ref<HTMLTextAreaElement | null>(null)

const singleEntry = computed(() =>
  props.entry.length > 1 ? undefined : props.entry[0]
)

const downloadLinks = computed(() => {
  if (singleEntry.value) return ''
  return props.entry.map((e) => fileUrl(e.path, e.meta)).join('\n')
})

const downloadLinksFocus = () => {
  linksEl.value!.select()
  linksEl.value!.scrollTop = 0
  linksEl.value!.scrollLeft = 0
}

const downloadSelected = () => {
  downloadFiles(props.entry)
}
</script>
<style lang="scss">
.download-view-page {
  position: relative;
  width: 300px;
  padding: 16px 16px 20px;

  .page-title {
    font-size: 28px;
    margin: 0 0 20px;
    font-weight: normal;
    -webkit-user-select: none;
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
    color: var(--color-on-accent);
    background-color: var(--color-accent-strong);
    text-decoration: none;
    padding: 10px 16px;
    margin-top: 16px;
    border-radius: var(--radius-control);
    transition: box-shadow var(--motion-duration-fast)
      var(--motion-easing-standard);
    -webkit-user-select: none;
    user-select: none;

    &:hover {
      box-shadow: var(--shadow-control);
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

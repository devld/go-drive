<template>
  <div class="cbz-preview">
    <div v-if="error" class="cbz-preview__error">{{ error }}</div>
    <template v-else-if="pages.length">
      <div class="cbz-preview__toolbar">
        <button
          v-for="(page, index) in pages"
          :key="page.name"
          type="button"
          :class="{ active: index === selected }"
          @click="selected = index"
        >
          {{ index + 1 }}
        </button>
      </div>
      <div class="cbz-preview__page">
        <img :src="pages[selected].url" :alt="pages[selected].name" />
      </div>
    </template>
    <div v-else class="cbz-preview__empty">
      {{ t('preview.cbz.empty') }}
    </div>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from '@go-drive/i18n'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import JSZip from 'jszip'
import { loadArrayBuffer } from '../utils/load'

const { t } = useI18n({ useScope: 'global' })

interface CbzPage {
  name: string
  url: string
}

const props = defineProps<{
  src?: string
  data?: ArrayBuffer
}>()

const pages = ref<CbzPage[]>([])
const selected = ref(0)
const error = ref('')
let loadToken = 0
let requestController: AbortController | undefined

const clearPages = () => {
  pages.value.forEach((page) => URL.revokeObjectURL(page.url))
  pages.value = []
  selected.value = 0
}

const load = async () => {
  const token = ++loadToken
  requestController?.abort()
  const controller = new AbortController()
  requestController = controller
  clearPages()
  error.value = ''
  if (!props.data && !props.src) return

  const loaded: CbzPage[] = []
  const discardLoaded = () => {
    loaded.forEach((page) => URL.revokeObjectURL(page.url))
  }

  try {
    const data = props.data || (await loadArrayBuffer(props.src!, controller.signal))
    if (token !== loadToken) return
    const archive = await JSZip.loadAsync(data)
    const files = Object.values(archive.files)
      .filter((file) => !file.dir && /\.(avif|gif|jpe?g|png|webp)$/i.test(file.name))
      .sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true }))
    for (const file of files) {
      const blob = await file.async('blob')
      if (token !== loadToken) {
        discardLoaded()
        return
      }
      loaded.push({ name: file.name, url: URL.createObjectURL(blob) })
    }
    pages.value = loaded
  } catch (e) {
    discardLoaded()
    if (token !== loadToken) return
    error.value = e instanceof Error ? e.message : t('preview.cbz.error')
  }
}

onMounted(load)
watch(() => [props.src, props.data], load)
onBeforeUnmount(() => {
  loadToken++
  requestController?.abort()
  requestController = undefined
  clearPages()
})
</script>
<style lang="scss">
.cbz-preview {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  min-height: 0;
  background: #111;
}

.cbz-preview__toolbar {
  display: flex;
  flex: none;
  gap: 4px;
  max-width: 100%;
  overflow: auto;
  padding: 8px;
  background: rgba(0, 0, 0, 0.85);

  button {
    flex: none;
    min-width: 32px;
    padding: 5px 8px;
    border: 1px solid #666;
    border-radius: 5px;
    color: #ddd;
    background: transparent;
    cursor: pointer;

    &.active {
      color: #111;
      background: #fff;
    }
  }
}

.cbz-preview__page {
  display: flex;
  flex: 1;
  min-height: 0;
  align-items: center;
  justify-content: center;
  overflow: auto;
  padding: 16px;

  img {
    display: block;
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
  }
}

.cbz-preview__empty,
.cbz-preview__error {
  display: grid;
  flex: 1;
  place-items: center;
  padding: 32px;
  color: #ddd;
}

.cbz-preview__error {
  color: #ff9d9d;
}
</style>

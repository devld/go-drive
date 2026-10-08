<template>
  <div class="epub-preview">
    <div class="epub-preview__toolbar">
      <button type="button" :disabled="!ready" @click="previous">
        {{ t('preview.ebook.previous') }}
      </button>
      <button type="button" :disabled="!ready" @click="next">
        {{ t('preview.ebook.next') }}
      </button>
    </div>
    <div ref="container" class="epub-preview__content" />
    <div v-if="error" class="epub-preview__error">{{ error }}</div>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from '@go-drive/i18n'
import ePub from 'epubjs'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { loadArrayBuffer } from '../utils/load'

const { t } = useI18n({ useScope: 'global' })

const props = defineProps<{
  src?: string
  data?: ArrayBuffer
}>()

const container = ref<HTMLDivElement | null>(null)
const error = ref('')
const ready = ref(false)
let book: any
let rendition: any
let renderToken = 0
let requestController: AbortController | undefined

const destroyBook = () => {
  requestController?.abort()
  requestController = undefined
  ready.value = false
  rendition?.destroy?.()
  book?.destroy?.()
  rendition = undefined
  book = undefined
  if (container.value) container.value.replaceChildren()
}

const renderBook = async () => {
  const token = ++renderToken
  destroyBook()
  error.value = ''
  if ((!props.data && !props.src) || !container.value) return

  try {
    const controller = new AbortController()
    requestController = controller
    if (token !== renderToken || !container.value) return
    const data = props.data || (await loadArrayBuffer(props.src!, controller.signal))
    if (token !== renderToken || !container.value) return
    book = ePub(data)
    rendition = book.renderTo(container.value, {
      width: '100%',
      height: '100%',
      flow: 'scrolled-doc',
      allowScriptedContent: false,
    })
    await rendition.display()
    if (token === renderToken) ready.value = true
  } catch (e) {
    if (token === renderToken) {
      error.value = e instanceof Error ? e.message : t('preview.ebook.error')
    }
  }
}

const previous = () => void rendition?.prev()
const next = () => void rendition?.next()

onMounted(renderBook)
watch(() => [props.src, props.data], renderBook)
onBeforeUnmount(() => {
  renderToken++
  destroyBook()
})
</script>
<style lang="scss">
.epub-preview {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  min-height: 0;
  background: var(--color-bg-elevated, #fff);
}

.epub-preview__toolbar {
  display: flex;
  flex: none;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--color-border, #ddd);

  button {
    padding: 5px 10px;
    border: 1px solid var(--color-border, #bbb);
    border-radius: 6px;
    background: var(--color-field-bg, #fff);
    color: inherit;
    cursor: pointer;
  }
}

.epub-preview__content {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

.epub-preview__error {
  position: absolute;
  inset: 48px 0 0;
  display: grid;
  place-items: center;
  padding: 24px;
  color: var(--color-danger, #c00);
  background: var(--color-bg-elevated, #fff);
}
</style>

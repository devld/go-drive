<template>
  <div class="markdown-split-view">
    <div class="markdown-split-view__toolbar" role="tablist">
      <button
        v-for="item in visibleModes"
        :key="item.value"
        type="button"
        :class="{ active: currentMode === item.value }"
        role="tab"
        :aria-selected="currentMode === item.value"
        @click="currentMode = item.value"
      >
        {{ item.label }}
      </button>
    </div>
    <div class="markdown-split-view__panes">
      <section
        v-show="currentMode === 'split' || currentMode === 'edit'"
        class="markdown-split-view__pane markdown-split-view__pane--editor"
      >
        <slot name="editor" />
      </section>
      <section
        v-show="currentMode === 'split' || currentMode === 'preview'"
        class="markdown-split-view__pane markdown-split-view__pane--preview"
      >
        <MarkdownPreview :content="content" />
      </section>
    </div>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from '@go-drive/i18n'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import MarkdownPreview from './MarkdownPreview.vue'

const props = withDefaults(
  defineProps<{
    content?: string
  }>(),
  { content: '' }
)

type Mode = 'split' | 'edit' | 'preview'
const currentMode = ref<Mode>('split')
const compact = ref(false)
const { t } = useI18n({ useScope: 'global' })

const modes = computed<{ value: Mode; label: string }[]>(() => [
  { value: 'split', label: t('preview.markdown.split') },
  { value: 'edit', label: t('preview.markdown.code') },
  { value: 'preview', label: t('preview.markdown.preview') },
])

const visibleModes = computed(() =>
  compact.value ? modes.value.slice(1) : modes.value
)
const media =
  typeof window !== 'undefined' && typeof window.matchMedia === 'function'
    ? window.matchMedia('(max-width: 720px)')
    : undefined

const updateCompact = () => {
  compact.value = !!media?.matches
  if (compact.value && currentMode.value === 'split') {
    currentMode.value = 'preview'
  }
}

onMounted(() => {
  updateCompact()
  media?.addEventListener('change', updateCompact)
})
onBeforeUnmount(() => media?.removeEventListener('change', updateCompact))
</script>
<style lang="scss">
.markdown-split-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  background: var(--color-bg-elevated, #fff);
}

.markdown-split-view__toolbar {
  display: flex;
  flex: none;
  gap: 4px;
  padding: 6px 10px;
  border-bottom: 1px solid var(--color-border, #ddd);
  background: var(--color-bg, #f7f7f7);

  button {
    border: 0;
    border-radius: 6px;
    padding: 5px 10px;
    color: inherit;
    background: transparent;
    cursor: pointer;

    &.active {
      background: var(--color-bg-hover, #e8e8e8);
      font-weight: 600;
    }
  }
}

.markdown-split-view__panes {
  display: flex;
  flex: 1;
  min-height: 0;
}

.markdown-split-view__pane {
  min-width: 0;
  min-height: 0;
  flex: 1;
}

.markdown-split-view__pane--editor {
  border-right: 1px solid var(--color-border, #ddd);
}

@media (max-width: 720px) {
  .markdown-split-view__panes {
    display: block;
  }

  .markdown-split-view__pane--editor,
  .markdown-split-view__pane--preview {
    height: 100%;
  }

  .markdown-split-view__pane--editor {
    border-right: 0;
  }
}
</style>

<template>
  <ul class="path-bar" data-ui="path-bar">
    <li
      v-for="s in segments"
      :key="s.path"
      class="path-bar__segment"
      data-ui="path-segment"
      :data-root="!s.path ? true : undefined"
    >
      <EntryLink
        class="path-bar__path"
        data-ui="path-link"
        :class="getEntryDropTargetClass(dropState, s.path)"
        :path="s.path"
        :get-link="getLink"
        :draggable="draggable"
        @click="pathChange"
        @dragstart="onDragStart"
        @dragover="onDragOver"
        @dragleave="onDragLeave"
        @drop="onDrop"
        >{{ s.name }}</EntryLink
      >
    </li>
  </ul>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { EntryEventData, GetLinkFn } from '.'
import {
  EntryDragState,
  getEntryDropTargetClass,
} from './useDrag'

const props = defineProps({
  path: {
    type: String,
    required: true,
  },
  getLink: {
    type: Function as PropType<GetLinkFn>,
  },
  draggable: {
    type: Boolean,
  },
  dropState: {
    type: Object as PropType<EntryDragState>,
  },
})

const { t } = useI18n()

const emit = defineEmits<{
  (e: 'update:path', data: EntryEventData): void
  (e: 'dragstart', data: EntryEventData): void
  (e: 'dragover', data: EntryEventData): void
  (e: 'dragleave', data: EntryEventData): void
  (e: 'drop', data: EntryEventData): void
}>()

const segments = computed(() => {
  const ss = props.path.replace(/\/+/g, '/').split('/').filter(Boolean)
  const pathSegments = [{ name: t('app.root_path'), path: '' }]
  ss.forEach((s, i) => {
    pathSegments.push({ name: s, path: ss.slice(0, i + 1).join('/') })
  })
  return pathSegments
})

const pathChange = (e: EntryEventData) => emit('update:path', e)
const onDragStart = (e: EntryEventData) => emit('dragstart', e)
const onDragOver = (e: EntryEventData) => emit('dragover', e)
const onDragLeave = (e: EntryEventData) => emit('dragleave', e)
const onDrop = (e: EntryEventData) => emit('drop', e)
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
      color: var(--color-text-muted);
    }
  }
}

.path-bar__path {
  cursor: pointer;
  text-decoration: none;
  color: unset;
  border-radius: 4px;
}
</style>

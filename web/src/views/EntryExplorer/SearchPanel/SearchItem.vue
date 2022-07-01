<template>
  <li class="search-panel__item" tabindex="-1">
    <EntryLink :entry="entry" @click="itemClicked">
      <EntryIcon :entry="entry" :show-thumbnail="false" />
      <div class="search-panel__item-info">
        <div class="search-panel__item-info-primary">
          <span class="search-panel__item-name">{{ entry.name }}</span>
          <span class="search-panel__item-size">{{
            formatBytes(entry.size)
          }}</span>
        </div>
        <div class="search-panel__item-info-secondary">
          <span class="search-panel__item-path">{{ dirFn(entry.path) }}</span>
          <span class="search-panel__item-mod-time">{{
            formatTime(entry.modTime)
          }}</span>
        </div>
      </div>
    </EntryLink>
  </li>
</template>
<script setup lang="ts">
import { EntryEventData } from '@/components/entry'
import { Entry, SearchHitItem } from '@/types'
import { dir as dirFn, formatBytes, formatTime } from '@/utils'
import { computed } from 'vue'

const props = defineProps({
  item: {
    type: Object as PropType<SearchHitItem>,
    required: true,
  },
})

const emit = defineEmits<{ (e: 'click', v: EntryEventData): void }>()

const entry = computed<Entry>(() => ({
  ...props.item.entry,
  modTime: new Date(props.item.entry.modTime).getTime(),
  meta: {},
}))

const itemClicked = (e: EntryEventData) => emit('click', e)
</script>
<style lang="scss">
.search-panel__item {
  margin: 0;
  padding: 8px 0;
  list-style-type: none;
  overflow: hidden;

  .entry-link {
    display: flex;
    color: var(--primary-text-color);
    text-decoration: none;
    padding: 0 16px;
  }

  .entry-icon {
    margin-right: 0.5em;
    flex-shrink: 0;
  }

  &:hover,
  &:focus {
    background-color: var(--hover-bg-color);
  }
}

.search-panel__item-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  overflow: hidden;

  & > div {
    display: flex;
    justify-content: space-between;
  }
}

.search-panel__item-name,
.search-panel__item-path {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.search-panel__item-size,
.search-panel__item-mod-time {
  overflow: hidden;
  white-space: nowrap;
}

.search-panel__item-info-secondary {
  font-size: 12px;
  color: var(--secondary-text-color);
}
</style>

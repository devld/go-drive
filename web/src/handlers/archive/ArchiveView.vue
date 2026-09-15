<template>
  <div class="archive-view" data-ui="preview" data-handler="archive">
    <HandlerTitleBar :title="entry.name" @close="emit('close')">
      <template #actions>
        <SimpleButton
          v-if="currentDir"
          variant="plain"
          small
          @click="goParent"
        >
          {{ $t('handler.archive.parent') }}
        </SimpleButton>
      </template>
    </HandlerTitleBar>

    <div class="archive-view__body">
      <div class="archive-view__location" :title="currentDir || '/'">
        {{ currentDir || '/' }}
      </div>

      <LoadingState v-if="loading" variant="panel" />
      <ErrorView
        v-else-if="error"
        :status="error.status"
        :message="error.message"
      />
      <div v-else class="archive-view__content">
        <div class="archive-view__entries">
          <button
            v-for="item in items"
            :key="item.path"
            class="archive-view__entry"
            :class="{ 'is-selected': selected?.path === item.path }"
            type="button"
            @click="open(item)"
          >
            <span class="archive-view__entry-icon" aria-hidden="true">
              {{ item.type === 'dir' ? '📁' : '📄' }}
            </span>
            <span class="archive-view__entry-name">{{ item.name }}</span>
            <span v-if="item.type === 'file'" class="archive-view__entry-size">
              {{ formatBytes(item.size) }}
            </span>
          </button>
          <span v-if="items.length === 0" class="archive-view__empty">
            {{ $t('handler.archive.empty') }}
          </span>
        </div>

        <div v-if="selected" class="archive-view__preview">
          <div class="archive-view__preview-title">
            <span>{{ selected.name }}</span>
            <a
              class="archive-view__download"
              :href="selectedURL"
              target="_blank"
              rel="noreferrer noopener nofollow"
              :download="selected.name"
            >
              {{ $t('handler.archive.open') }}
            </a>
          </div>
          <iframe
            class="archive-view__frame"
            :src="selectedURL"
            sandbox="allow-downloads"
            :title="selected.name"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { archiveContentUrl, ArchiveEntry, listArchiveEntries } from '@/api'
import ErrorView from '@/components/ErrorView.vue'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import { Entry } from '@/types'
import { formatBytes } from '@/utils'
import { LoadingState } from '@go-drive/utils'
import { computed, ref, watch } from 'vue'
import { EntryHandlerContext } from '../types'

const props = defineProps({
  entry: {
    type: Object as PropType<Entry>,
    required: true,
  },
  entries: { type: Array as PropType<Entry[]> },
  ctx: {
    type: Object as PropType<EntryHandlerContext>,
    required: true,
  },
})

const emit = defineEmits<{ (e: 'close'): void }>()

const currentDir = ref('')
const items = ref<ArchiveEntry[]>([])
const selected = ref<ArchiveEntry>()
const loading = ref(false)
const error = ref<any>()
let loadRequest = 0

const selectedURL = computed(() =>
  selected.value
    ? archiveContentUrl(
        props.entry.path,
        props.entry.meta,
        selected.value.path
      )
    : ''
)

const load = async () => {
  const request = ++loadRequest
  loading.value = true
  error.value = undefined
  selected.value = undefined
  try {
    const result = await listArchiveEntries(
      props.entry.path,
      props.entry.meta,
      currentDir.value
    )
    if (request === loadRequest) items.value = result
  } catch (e: any) {
    if (request === loadRequest) error.value = e
  } finally {
    if (request === loadRequest) loading.value = false
  }
}

const open = (item: ArchiveEntry) => {
  if (item.type === 'dir') {
    currentDir.value = item.path
    return
  }
  selected.value = item
}

const goParent = () => {
  const index = currentDir.value.lastIndexOf('/')
  currentDir.value = index < 0 ? '' : currentDir.value.substring(0, index)
}

watch(
  () => [props.entry.path, currentDir.value],
  () => {
    if (!props.entry.path) return
    load()
  },
  { immediate: true }
)

watch(
  () => props.entry.path,
  () => {
    currentDir.value = ''
  }
)
</script>

<style lang="scss">
.archive-view {
  position: relative;
  display: flex;
  flex-direction: column;
  width: min(900px, 90vw);
  height: min(640px, 80vh);
  overflow: hidden;
  background-color: var(--color-bg-elevated);
}

.archive-view__body {
  display: flex;
  flex: 1;
  min-height: 0;
  flex-direction: column;
}

.archive-view__location {
  flex: none;
  padding: 8px 16px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-text-muted);
  border-bottom: 1px solid var(--color-border);
}

.archive-view__content {
  display: flex;
  flex: 1;
  min-height: 0;
}

.archive-view__entries {
  flex: 0 0 min(360px, 38vw);
  min-width: 220px;
  overflow: auto;
  padding: 8px;
  border-right: 1px solid var(--color-border);
}

.archive-view__entry {
  display: flex;
  align-items: center;
  width: 100%;
  gap: 8px;
  padding: 8px 10px;
  color: var(--color-text);
  text-align: left;
  border: 0;
  border-radius: var(--radius-control);
  background: transparent;
  cursor: pointer;

  &:hover,
  &.is-selected {
    background: var(--color-bg-hover);
  }
}

.archive-view__entry-icon {
  flex: none;
}

.archive-view__entry-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.archive-view__entry-size {
  margin-left: auto;
  color: var(--color-text-muted);
  font-size: 12px;
}

.archive-view__empty {
  display: block;
  padding: 24px 10px;
  color: var(--color-text-muted);
  text-align: center;
}

.archive-view__preview {
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
}

.archive-view__preview-title {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: none;
  padding: 8px 12px;
  overflow: hidden;
  white-space: nowrap;
  border-bottom: 1px solid var(--color-border);

  > span {
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

.archive-view__download {
  margin-left: auto;
  color: var(--color-accent);
}

.archive-view__frame {
  flex: 1;
  width: 100%;
  min-height: 0;
  border: 0;
  background: #fff;
}

@media (max-width: 700px) {
  .archive-view__content {
    flex-direction: column;
  }

  .archive-view__entries {
    flex: 0 0 35%;
    width: auto;
    min-width: 0;
    border-right: 0;
    border-bottom: 1px solid var(--color-border);
  }
}
</style>

<template>
  <div class="upload-task-item" :class="`task-status-${task.status}`">
    <div
      v-if="progress && progress !== 1"
      class="upload-task-item__progress-track"
    >
      <div
        class="upload-task-item__progress-bar"
        :style="{ width: `${progress * 100}%` }"
      ></div>
    </div>
    <span class="upload-task-item__filename" :title="filename">
      <EntryIcon
        class="upload-task-item__icon"
        :entry="entry"
        :show-thumbnail="false"
      />
      <span class="upload-task__name">{{ filename }}</span>
    </span>
    <span
      class="upload-task-item__size"
      :title="formatBytes(task.task.size!, 1)"
      >{{ formatBytes(task.task.size!, 1) }}</span
    >
    <span class="upload-task-item__location">
      <EntryLink
        :path="dir"
        :get-link="getLink"
        :title="dir"
        @click="emit('navigate', $event)"
      >
        {{ filenameFn(dir) }}
      </EntryLink>
    </span>
    <span
      class="upload-task-item__status"
      :title="task.error?.message || task.error"
      >{{ statusText }}</span
    >
    <span class="upload-task-item__ops">
      <SimpleButton
        v-if="showStart"
        variant="plain"
        icon="play"
        :title="t('p.task.start')"
        :aria-label="t('p.task.start')"
        @click="emit('start')"
      />
      <SimpleButton
        v-if="showPause"
        variant="plain"
        icon="pause"
        :title="t('p.task.pause')"
        :aria-label="t('p.task.pause')"
        @click="emit('pause')"
      />
      <SimpleButton
        v-if="showStop"
        variant="plain"
        icon="stop"
        :title="t('p.task.stop')"
        :aria-label="t('p.task.stop')"
        @click="emit('stop')"
      />
      <SimpleButton
        v-if="showRemove"
        variant="plain"
        icon="close"
        :title="t('p.task.remove')"
        :aria-label="t('p.task.remove')"
        @click="emit('remove')"
      />
    </span>
  </div>
</template>
<script setup lang="ts">
import {
  filename as filenameFn,
  dir as dirFn,
  formatPercent,
  formatBytes,
} from '@/utils'
import {
  STATUS_CREATED,
  STATUS_PAUSED,
  STATUS_UPLOADING,
  STATUS_STOPPED,
  STATUS_ERROR,
  STATUS_COMPLETED,
  STATUS_MASK_CAN_START,
  STATUS_MASK_CAN_PAUSE,
  STATUS_MASK_CAN_STOP,
  STATUS_STARTING,
  UploadTaskItem,
} from '@/api/upload-manager/task'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { EntryEventData, GetLinkFn } from '@/components/entry'
import { Entry } from '@/types'

const { t } = useI18n()

const props = defineProps({
  task: {
    type: Object as PropType<UploadTaskItem>,
    required: true,
  },
  getLink: {
    type: Function as PropType<GetLinkFn>,
  },
})

const emit = defineEmits<{
  (e: 'navigate', v: EntryEventData): void
  (e: 'start'): void
  (e: 'remove'): void
  (e: 'stop'): void
  (e: 'pause'): void
}>()

const entry = computed<Entry>(() => ({
  type: 'file',
  name: filenameFn(props.task.task.path),
  path: props.task.task.path,
  size: -1,
  modTime: -1,
  meta: {},
}))

const dir = computed(() => dirFn(entry.value.path))
const filename = computed(() => entry.value.name)

const statusText = computed(() => {
  switch (props.task.status) {
    case STATUS_CREATED:
      return t('p.task.s_created')
    case STATUS_STARTING:
      return t('p.task.s_starting')
    case STATUS_PAUSED:
      return t('p.task.s_paused')
    case STATUS_UPLOADING:
      return formatPercent(progress.value!)
    case STATUS_STOPPED:
      return t('p.task.s_stopped')
    case STATUS_ERROR:
      return t('p.task.s_error')
    case STATUS_COMPLETED:
      return t('p.task.s_completed')
  }
  return ''
})

const progress = computed(() => {
  const p = props.task.progress
  if (!p) return null
  return p.loaded / p.total
})

const showStart = computed(() => props.task.isStatus(STATUS_MASK_CAN_START))
const showPause = computed(() => props.task.isStatus(STATUS_MASK_CAN_PAUSE))
const showStop = computed(() => props.task.isStatus(STATUS_MASK_CAN_STOP))
const showRemove = computed(() => !showStop.value)
</script>
<style lang="scss">
.upload-task-item {
  position: relative;
  box-sizing: border-box;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 72px 88px 72px max-content;
  align-items: center;
  column-gap: 8px;
  min-width: 0;
  padding: 8px 8px 11px;
  font-size: 14px;
  overflow: hidden;

  & > * {
    z-index: 1;
    min-width: 0;
  }

  &.task-status-64 {
    .upload-task-item__status {
      color: var(--color-danger);
    }
  }
}

.upload-task-item__progress-track {
  position: absolute;
  right: 8px;
  left: 8px;
  bottom: 4px;
  height: 3px;
  border-radius: 999px;
  z-index: 2;
  overflow: hidden;
  background-color: var(--color-progress-track-paused);
}

.upload-task-item__progress-bar {
  height: 100%;
  border-radius: inherit;
  transition:
    width 400ms linear,
    background-color var(--motion-duration-normal)
      var(--motion-easing-standard);
  background-color: var(--color-progress-value);
}

.task-status-1 {
  .upload-task-item__progress-bar {
    background-color: var(--color-text-muted);
  }
}

.entry-icon.upload-task-item__icon {
  flex: 0 0 auto;
  width: 26px;
  height: 26px;
  margin-right: 0.5em;
}

.upload-task-item__filename {
  display: flex;
  align-items: center;
  min-width: 0;
  overflow: hidden;
}

.upload-task__name {
  flex: 1 1 0;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.upload-task-item__size {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.upload-task-item__location {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;

  a {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    text-decoration: none;
    color: var(--color-accent);
  }
}

.upload-task-item__status {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.upload-task-item__ops {
  white-space: nowrap;
  text-align: right;
  display: inline-flex;
  justify-content: flex-end;
  align-items: center;
  gap: 4px;

  .simple-button.plain {
    font-size: 18px;
  }
}

@media screen and (max-width: 600px) {
  .upload-task-item {
    grid-template-columns: minmax(0, 1fr) max-content;
    grid-template-areas:
      'name ops'
      'size status';
    row-gap: 4px;
  }

  .upload-task-item__filename {
    grid-area: name;
  }

  .upload-task-item__ops {
    grid-area: ops;
  }

  .upload-task-item__size {
    grid-area: size;
  }

  .upload-task-item__status {
    grid-area: status;
  }

  .upload-task-item__location {
    display: none;
  }
}
</style>

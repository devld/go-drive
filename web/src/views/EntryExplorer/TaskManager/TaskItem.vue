<template>
  <div class="upload-task-item" :class="`task-status-${task.status}`">
    <div
      v-if="progress && progress !== 1"
      class="upload-task-item__progress-bar"
      :style="{ width: `${progress * 100}%` }"
    ></div>
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
        @click="emit('navigate', $event)"
      >
        {{ filenameFn(dir) }}
      </EntryLink>
    </span>
    <span class="upload-task-item__status">{{ statusText }}</span>
    <span class="upload-task-item__ops">
      <button
        v-if="showStart"
        class="upload-task-item__start plain-button"
        :title="t('p.task.start')"
        @click="emit('start')"
      >
        <Icon svg="#icon-play" />
      </button>
      <button
        v-if="showPause"
        class="upload-task-item__pause plain-button"
        :title="t('p.task.pause')"
        @click="emit('pause')"
      >
        <Icon svg="#icon-pause" />
      </button>
      <button
        v-if="showStop"
        class="upload-task-item__stop plain-button"
        :title="t('p.task.stop')"
        @click="emit('stop')"
      >
        <Icon svg="#icon-stop" />
      </button>
      <button
        v-if="showRemove"
        class="upload-task-item__remove plain-button"
        :title="t('p.task.remove')"
        @click="emit('remove')"
      >
        <Icon svg="#icon-close" />
      </button>
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

const filename = computed(() => filenameFn(entry.value.name))

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
  display: flex;
  align-items: center;
  padding: 8px;
  font-size: 14px;

  & > * {
    z-index: 1;
  }
}

.upload-task-item__progress-bar {
  position: absolute;
  top: 0;
  left: 0;
  bottom: 0;
  right: 0;
  transition: 0.4s;
  background-color: #bcdffb;
  background-color: var(--progress-bar-color);
}

.task-status-1 {
  .upload-task-item__progress-bar {
    background-color: #e2eeff;
  }
}

.entry-icon.upload-task-item__icon {
  width: 26px;
  height: 26px;
  margin-right: 0.5em;
  vertical-align: middle;
}

.upload-task-item__filename {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.upload-task__name {
  vertical-align: middle;
}

.upload-task-item__size {
  width: 60px;
  white-space: nowrap;
}

.upload-task-item__location {
  width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;

  a {
    text-decoration: none;
    color: var(--link-color);
  }
}

.upload-task-item__status {
  width: 60px;
}

.upload-task-item__ops {
  width: 60px;
  white-space: nowrap;
  text-align: right;
  .plain-button {
    font-size: 18px;
  }
}
</style>

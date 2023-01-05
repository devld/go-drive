<template>
  <div class="upload-task-manager">
    <UploadTaskItemComponent
      v-for="task of tasks"
      :key="task.id"
      :task="task"
      @navigate="emit('navigate', $event)"
      @start="emit('start', task)"
      @pause="emit('pause', task)"
      @stop="emit('stop', task)"
      @remove="emit('remove', task)"
    />
    <div v-if="tasks.length === 0" class="no-task">
      {{ $t('p.task.empty') }}
    </div>
  </div>
</template>
<script lang="ts">
export default { name: 'TaskManager' }
</script>
<script setup lang="ts">
import { UploadTaskItem } from '@/api/upload-manager/task'
import { EntryEventData } from '@/components/entry'
import UploadTaskItemComponent from './TaskItem.vue'

defineProps({
  tasks: {
    type: Array as PropType<UploadTaskItem[]>,
    required: true,
  },
})

const emit = defineEmits<{
  (e: 'navigate', v: EntryEventData): void
  (e: 'start', v: UploadTaskItem): void
  (e: 'remove', v: UploadTaskItem): void
  (e: 'stop', v: UploadTaskItem): void
  (e: 'pause', v: UploadTaskItem): void
}>()
</script>
<style lang="scss">
.upload-task-manager {
  padding: 16px;
  max-height: 60vh;
  overflow-x: hidden;
  overflow-y: auto;

  .no-task {
    width: 100%;
    height: 60px;
    line-height: 60px;
    text-align: center;
    color: var(--secondary-text-color);
    user-select: none;
    -webkit-user-select: none;
  }
}
</style>

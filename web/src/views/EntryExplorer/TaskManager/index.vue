<template>
  <div class="upload-task-manager">
    <upload-task-item
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
<script>
export default { name: 'TaskManager' }
</script>
<script setup>
import UploadTaskItem from './TaskItem.vue'

defineProps({
  tasks: {
    type: Array,
    required: true,
  },
})

const emit = defineEmits(['navigate', 'start', 'pause', 'stop', 'remove'])
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
  }
}
</style>

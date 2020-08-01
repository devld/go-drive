<template>
  <div class="upload-task-manager">
    <upload-task-item
      v-for="task of tasks"
      :key="task.id"
      :task="task"
      @navigate="$emit('navigate', $event)"
      @start="$emit('start', task)"
      @pause="$emit('pause', task)"
      @stop="$emit('stop', task)"
      @remove="$emit('remove', task)"
    />
    <div v-if="tasks.length === 0" class="no-task">Nothing here</div>
  </div>
</template>
<script>
import UploadTaskItem from './TaskItem'

export default {
  name: 'UploadTaskManager',
  components: { UploadTaskItem },
  props: {
    tasks: {
      type: Array,
      required: true
    }
  }
}
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
    color: #aaa;
    user-select: none;
  }
}
</style>

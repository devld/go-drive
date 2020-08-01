<template>
  <div class="new-entry-area">
    <float-button
      class="button-new-item"
      v-model="floatMenuShowing"
      title="New Item"
      :buttons="[
        { slot: 'file', title: 'Upload file' },
        { slot: 'folder', title: 'Create folder' }
      ]"
      @click="newButtonClicked"
    >
      <span class="icon-new-item" :class="{ 'active': floatMenuShowing }">
        <i-icon svg="#icon-add" />
      </span>
      <i-icon slot="file" svg="#icon-file" />
      <i-icon slot="folder" svg="#icon-folder" />
    </float-button>

    <dialog-view
      v-model="taskManagerShowing"
      title="Upload Tasks"
      esc-close
      overlay-close
      transition="tm-dialog"
      @closed="taskManagerClosed"
    >
      <task-manager
        class="task-manager"
        :tasks="tasks"
        @navigate="hideTaskManager"
        @start="startTask"
        @pause="pauseTask"
        @stop="stopTask"
        @remove="removeTask"
      />
    </dialog-view>

    <button
      class="button-task-manager"
      v-if="taskManagerButtonShowing"
      @click="showTaskManager"
    >Tasks{{ (uploadStatus && uploadStatus.total > 0) ? `: ${uploadStatus.completed}/${uploadStatus.total}` : '' }}</button>
    <input class="hidden-input-file" ref="file" type="file" @change="onFilesChosen" multiple />
  </div>
</template>
<script>
import TaskManager from './TaskManager'
import { makeDir } from '@/api'
import { pathClean, pathJoin } from '@/utils'
import uploadManager from '@/api/upload-manager'
// eslint-disable-next-line no-unused-vars
import { UploadTaskItem, STATUS_COMPLETED } from '@/api/upload-manager/task'

export default {
  name: 'NewEntryArea',
  components: { TaskManager },
  props: {
    path: {
      type: String,
      required: true
    }
  },
  data () {
    return {
      floatMenuShowing: false,

      taskManagerShowing: false,
      taskManagerButtonShowing: true,

      uploadStatus: { completed: 0, total: 0 },

      /**
       * @type {Array.<UploadTaskItem>}
       */
      tasks: []
    }
  },
  created () {
    this.tasks = uploadManager.getTasks()
    this.updateTasksSummary()
    uploadManager.on('taskChanged', this.onTasksChanged)

    window.addEventListener('beforeunload', this.onWindowUnload)
  },
  beforeDestroy () {
    uploadManager.off('taskChanged', this.onTasksChanged)
    window.removeEventListener('beforeunload', this.onWindowUnload)
  },
  methods: {
    onFilesChosen () {
      const files = [...this.$refs.file.files]
      this.$refs.file.value = null
      if (!files.length) return
      files.forEach(file => {
        uploadManager.submitTask({
          path: pathClean(pathJoin(this.path, file.name)),
          file
        })
      })
      this.showTaskManager()
    },
    uploadFile () {
      this.$refs.file.click()
    },
    createDir () {
      this.$input({
        title: 'Create Folder',
        validator: {
          pattern: /^[^/]+$/,
          message: 'Invalid folder name.'
        },
        onOk: text => {
          return makeDir(pathClean(pathJoin(this.path, text)))
            .then(() => {
              this.$emit('update')
            })
            .catch(e => {
              this.$alert(e.message).catch(() => { })
              return Promise.reject(e)
            })
        }
      })
    },
    onTasksChanged ({ tasks, task }) {
      this.tasks = tasks
      this.updateTasksSummary()
      if (task && task.status === STATUS_COMPLETED) {
        this.$emit('update')
      }
    },
    startTask (task) {
      uploadManager.startTask(task.id)
    },
    pauseTask (task) {
      uploadManager.pauseTask(task.id)
    },
    async stopTask (task) {
      try { await this.$confirm('Stop this task?') } catch { return }
      uploadManager.stopTask(task.id)
    },
    async removeTask (task) {
      try {
        await this.$confirm({
          message: 'Remove this task, cannot be undone?',
          confirmType: 'danger'
        })
      } catch { return }
      uploadManager.removeTask(task.id)
    },
    updateTasksSummary () {
      const completed = this.tasks.filter(t => t.status === STATUS_COMPLETED).length
      this.uploadStatus = { completed, total: this.tasks.length }
    },
    newButtonClicked ({ button, index }) {
      if (index === 0) this.uploadFile()
      if (index === 1) this.createDir()
    },
    showTaskManager () {
      this.taskManagerButtonShowing = false
      this.taskManagerShowing = true
    },
    taskManagerClosed () {
      this.taskManagerButtonShowing = true
    },
    hideTaskManager () {
      this.taskManagerShowing = false
    },
    onWindowUnload (e) {
      if (this.uploadStatus.completed < this.uploadStatus.total) {
        e.preventDefault()
        e.returnValue = ''
      }
    }
  }
}
</script>
<style lang="scss">
.new-entry-area {
  position: fixed;

  .float-button.button-new-item {
    position: fixed;
    bottom: 5vh;
    right: 5vw;
  }

  .icon-new-item {
    display: inline-block;
    box-sizing: border-box;
    background-color: #fff;
    border-radius: 50%;
    margin: 5px;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);

    .icon {
      display: block;
      transition: 0.3s;
      fill: #546e7a;
    }

    &.active {
      .icon {
        transform: rotate(135deg);
      }
    }
  }

  .button-task-manager {
    position: fixed;
    right: calc(5vw + 100px);
    bottom: 0;

    outline: none;
    padding: 10px 26px;
    background-color: #fff;
    border: none;
    box-shadow: 0 0 4px rgba(0, 0, 0, 0.1);
    cursor: pointer;
  }

  .task-manager {
    width: 50vw;
    max-width: 700px;
    min-width: 600px;
  }

  @media screen and (max-width: 600px) {
    .task-manager {
      width: calc(100vw - 64px);
      max-width: unset;
      min-width: unset;
    }

    .upload-task-item__location {
      display: none;
    }
  }

  .hidden-input-file {
    position: fixed;
    top: -200px;
    left: -200px;
  }
}

@keyframes tm-dialog {
  from {
    right: calc(5vw + 100px);
    transform: scale(0.25);
    bottom: 0;
    opacity: 0;
  }
  to {
    right: 50vw;
    bottom: 50vh;
    transform: translate(50%, 50%) scale(1);
    opacity: 1;
  }
}

.tm-dialog-enter-active {
  position: fixed;
  overflow: hidden;
  transform-origin: right bottom;
  animation: tm-dialog 0.3s;
}

.tm-dialog-leave-active {
  position: fixed;
  overflow: hidden;
  transform-origin: right bottom;
  animation: tm-dialog 0.3s reverse;
}
</style>

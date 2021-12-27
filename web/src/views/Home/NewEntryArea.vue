<template>
  <div class="new-entry-area">
    <float-button
      v-if="!readonly"
      class="button-new-item"
      v-model="floatMenuShowing"
      :title="$t('p.new_entry.new_item')"
      :buttons="[
        {
          slot: 'new-file',
          title: $t('p.new_entry.create_file'),
          fn: 'createEmptyFile',
        },
        {
          slot: 'upload-file',
          title: $t('p.new_entry.upload_file'),
          fn: 'uploadFile',
        },
        {
          slot: 'new-folder',
          title: $t('p.new_entry.create_folder'),
          fn: 'createDir',
        },
      ]"
      @click="newButtonClicked"
    >
      <span class="icon-new-item" :class="{ active: floatMenuShowing }">
        <i-icon svg="#icon-add1" />
      </span>
      <i-icon slot="new-file" svg="#icon-new-file" />
      <i-icon slot="upload-file" svg="#icon-upload-file" />
      <i-icon slot="new-folder" svg="#icon-new-folder" />
    </float-button>

    <dialog-view
      v-model="taskManagerShowing"
      :title="$t('p.new_entry.upload_tasks')"
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
    >
      {{
        $t('p.new_entry.tasks_status', {
          p:
            uploadStatus && uploadStatus.total > 0
              ? `: ${uploadStatus.completed}/${uploadStatus.total}`
              : '',
        })
      }}
    </button>
    <input
      class="hidden-input-file"
      ref="file"
      type="file"
      @change="onFilesChosen"
      multiple
    />

    <div v-if="dropZoneActive" class="drop-zone-indicator">
      {{ $t('p.new_entry.drop_tip') }}
    </div>
  </div>
</template>
<script>
import TaskManager from './TaskManager'
import { makeDir } from '@/api'
import { dir, pathClean, pathJoin } from '@/utils'
import { UploadManager } from '@/api/upload-manager'
// eslint-disable-next-line no-unused-vars
import { UploadTaskItem, STATUS_COMPLETED } from '@/api/upload-manager/task'
import { createDialog } from '@/utils/ui-utils/base-dialog'
import FileExistsDialogInner from './FileExistsConfirmDialog'

const FileExistsDialog = createDialog('FileExistsDialog', FileExistsDialogInner)

const uploadManager = new UploadManager({ concurrent: 3 })

function getFiles(dataTransfer) {
  if (!dataTransfer || !dataTransfer.items) return
  const files = []
  for (const f of dataTransfer.items) {
    if (typeof f.webkitGetAsEntry === 'function') {
      const entry = f.webkitGetAsEntry()
      if (!entry || !entry.isFile) continue
    }
    files.push(f.getAsFile())
  }
  return files.length === 0 ? undefined : files
}

export default {
  name: 'NewEntryArea',
  components: { TaskManager },
  props: {
    path: {
      type: String,
      required: true,
    },
    entries: {
      type: null,
      required: true,
    },
    readonly: {
      type: Boolean,
    },
  },
  data() {
    return {
      floatMenuShowing: false,

      taskManagerShowing: false,
      taskManagerButtonShowing: true,

      uploadStatus: { completed: 0, total: 0 },

      /**
       * @type {Array.<UploadTaskItem>}
       */
      tasks: [],

      dropZoneActive: false,
    }
  },
  created() {
    this.tasks = uploadManager.getTasks()
    this.updateTasksSummary()
    uploadManager.on('taskChanged', this.onTasksChanged)

    window.addEventListener('beforeunload', this.onWindowUnload)

    window.addEventListener('dragover', this.onDragEnter)
    window.addEventListener('dragleave', this.onDragLeave)
    window.addEventListener('drop', this.onItemsDropped)
  },
  beforeDestroy() {
    uploadManager.off('taskChanged', this.onTasksChanged)
    window.removeEventListener('beforeunload', this.onWindowUnload)

    window.removeEventListener('dragover', this.onDragEnter)
    window.removeEventListener('dragleave', this.onDragLeave)
    window.removeEventListener('drop', this.onItemsDropped)
  },
  methods: {
    onItemsDropped(e) {
      this.toggleDropZoneActive(false)
      e.preventDefault()
      const files = getFiles(e.dataTransfer)
      if (files) {
        this.submitUploadTasks(files)
      }
    },
    onFilesChosen() {
      const files = [...this.$refs.file.files]
      this.$refs.file.value = null
      this.submitUploadTasks(files)
    },
    async submitUploadTasks(files) {
      if (!files.length) return
      let applyAll, override
      for (const file of files) {
        if (this.entries && this.entries.find(e => e.name === file.name)) {
          if (!applyAll) {
            const { override: override_, all } = await this.confirmFileExists(
              file
            )
            applyAll = all
            override = override_
          }
        }
        if (override === false) continue
        uploadManager.submitTask({
          path: pathClean(pathJoin(this.path, file.name)),
          file,
          override,
        })
      }
      this.showTaskManager()
    },
    uploadFile() {
      this.$refs.file.click()
    },
    createEmptyFile() {
      this.$input({
        title: this.$t('p.new_entry.create_file'),
        validator: {
          pattern: /^[^/]+$/,
          message: this.$t('p.new_entry.invalid_filename'),
        },
        onOk: async text => {
          try {
            await uploadManager.upload(
              {
                path: pathClean(pathJoin(this.path, text)),
                file: '',
                override: false,
              },
              true
            )
            this.$emit('update')
          } catch (e) {
            this.$alert(e.message).catch(() => {})
            throw e
          }
        },
      })
    },
    createDir() {
      this.$input({
        title: this.$t('p.new_entry.create_folder'),
        validator: {
          pattern: /^[^/]+$/,
          message: this.$t('p.new_entry.invalid_folder_name'),
        },
        onOk: async text => {
          try {
            await makeDir(pathClean(pathJoin(this.path, text)))
            this.$emit('update')
          } catch (e) {
            this.$alert(e.message).catch(() => {})
            throw e
          }
        },
      })
    },
    onTasksChanged({ tasks, task }) {
      this.tasks = tasks
      this.updateTasksSummary()
      if (task && task.status === STATUS_COMPLETED) {
        if (this.path === dir(task.task.path)) {
          this.$emit('update')
        }
      }
    },
    startTask(task) {
      uploadManager.startTask(task.id)
    },
    pauseTask(task) {
      uploadManager.pauseTask(task.id)
    },
    async stopTask(task) {
      try {
        await this.$confirm(this.$t('p.new_entry.confirm_stop_task'))
      } catch {
        return
      }
      uploadManager.stopTask(task.id)
    },
    async removeTask(task) {
      try {
        await this.$confirm({
          message: this.$t('p.new_entry.confirm_remove_task'),
          confirmType: 'danger',
        })
      } catch {
        return
      }
      uploadManager.removeTask(task.id, true)
    },
    updateTasksSummary() {
      const completed = this.tasks.filter(t => t.status === STATUS_COMPLETED)
        .length
      this.uploadStatus = { completed, total: this.tasks.length }
    },
    async confirmFileExists(file) {
      try {
        const all = (
          await this.$dialog(FileExistsDialog, {
            title: this.$t('p.new_entry.file_exists'),
            message: this.$t('p.new_entry.file_exists_confirm', {
              m: file.name,
            }),
            confirmText: this.$t('p.new_entry.skip'),
            cancelText: this.$t('p.new_entry.override'),
            cancelType: 'danger',
            filename: file.name,
          })
        ).all
        return { all, override: false }
      } catch (e) {
        if (!e) return { all: false, override: false }
        return { all: e.all, override: true }
      }
    },
    newButtonClicked({ button }) {
      this[button.fn]()
    },
    showTaskManager() {
      this.taskManagerButtonShowing = false
      this.taskManagerShowing = true
    },
    taskManagerClosed() {
      this.taskManagerButtonShowing = true
    },
    hideTaskManager() {
      this.taskManagerShowing = false
    },
    onWindowUnload(e) {
      if (this.uploadStatus.completed < this.uploadStatus.total) {
        e.preventDefault()
        e.returnValue = ''
      }
    },
    onDragEnter(e) {
      if (this.readonly) return
      e.preventDefault()
      clearTimeout(this._dragLeaveTimeout)
      this.toggleDropZoneActive(true)
    },
    onDragLeave(e) {
      e.preventDefault()
      clearTimeout(this._dragLeaveTimeout)
      this._dragLeaveTimeout = setTimeout(() => {
        this.toggleDropZoneActive(false)
      }, 100)
    },
    toggleDropZoneActive(active) {
      this.dropZoneActive = active
    },
  },
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
    border-radius: 50%;
    margin: 5px;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
    background-color: var(--secondary-bg-color);

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
    background-color: var(--secondary-bg-color);
    color: var(--primary-text-color);

    border: none;
    box-shadow: 0 0 4px rgba(0, 0, 0, 0.1);
    cursor: pointer;
    user-select: none;
  }

  .task-manager {
    width: 50vw;
    max-width: 700px;
    min-width: 600px;
  }

  .drop-zone-indicator {
    position: fixed;
    top: 8px;
    left: 8px;
    right: 8px;
    bottom: 8px;
    z-index: 1000;
    border: solid 2px #66ccff;
    border-radius: 6px;
    pointer-events: none;
    background-color: rgba(102, 204, 255, 0.4);

    display: flex;
    justify-content: center;
    align-items: center;
    font-size: 24px;
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

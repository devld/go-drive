<template>
  <div class="new-entry-area">
    <float-button
      v-if="!readonly"
      v-model="floatMenuShowing"
      class="button-new-item"
      :title="t('p.new_entry.new_item')"
      :buttons="[
        {
          slot: 'new-file',
          title: t('p.new_entry.create_file'),
          fn: 'createEmptyFile',
        },
        {
          slot: 'upload-file',
          title: t('p.new_entry.upload_file'),
          fn: 'uploadFile',
        },
        {
          slot: 'new-folder',
          title: t('p.new_entry.create_folder'),
          fn: 'createDir',
        },
      ]"
      @click="newButtonClicked"
    >
      <span class="icon-new-item" :class="{ active: floatMenuShowing }">
        <i-icon svg="#icon-add1" />
      </span>
      <template #new-file><i-icon svg="#icon-new-file" /></template>
      <template #upload-file><i-icon svg="#icon-upload-file" /></template>
      <template #new-folder><i-icon svg="#icon-new-folder" /></template>
    </float-button>

    <dialog-view
      v-model:show="taskManagerShowing"
      :title="t('p.new_entry.upload_tasks')"
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
      v-if="taskManagerButtonShowing"
      class="button-task-manager"
      @click="showTaskManager"
    >
      {{
        t('p.new_entry.tasks_status', {
          p:
            uploadStatus && uploadStatus.total > 0
              ? `: ${uploadStatus.completed}/${uploadStatus.total}`
              : '',
        })
      }}
    </button>
    <input
      ref="fileEl"
      class="hidden-input-file"
      type="file"
      multiple
      @change="onFilesChosen"
    />

    <div v-if="dropZoneActive" class="drop-zone-indicator">
      {{ t('p.new_entry.drop_tip') }}
    </div>
  </div>
</template>
<script setup>
import TaskManager from './TaskManager/index.vue'
import { makeDir } from '@/api'
import { dir, pathClean, pathJoin } from '@/utils'
import { UploadManager } from '@/api/upload-manager'
// eslint-disable-next-line no-unused-vars
import { UploadTaskItem, STATUS_COMPLETED } from '@/api/upload-manager/task'
import { createDialog } from '@/utils/ui-utils/base-dialog'
import FileExistsDialogInner from './FileExistsConfirmDialog.vue'
import { alert, confirm, dialog, input } from '@/utils/ui-utils'
import { onBeforeMount, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'

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

const { t } = useI18n()

const props = defineProps({
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
})

const emit = defineEmits(['update'])

const floatMenuShowing = ref(false)

const taskManagerShowing = ref(false)
const taskManagerButtonShowing = ref(true)

const uploadStatus = ref({ completed: 0, total: 0 })

/**
 * @type {import('vue').Ref<import('vue').UnwrapRef<Array.<UploadTaskItem>>>}
 */
const tasks = ref([])

const dropZoneActive = ref(false)

const fileEl = ref(null)

const onItemsDropped = (e) => {
  toggleDropZoneActive(false)
  e.preventDefault()
  const files = getFiles(e.dataTransfer)
  if (files) {
    submitUploadTasks(files)
  }
}

const onFilesChosen = () => {
  const files = [...fileEl.value.files]
  fileEl.value.value = null
  submitUploadTasks(files)
}

const submitUploadTasks = async (files) => {
  if (!files.length) return
  let applyAll, override
  for (const file of files) {
    if (props.entries?.find((e) => e.name === file.name)) {
      if (!applyAll) {
        const { override: override_, all } = await confirmFileExists(file)
        applyAll = all
        override = override_
      }
    }
    if (override === false) continue
    uploadManager.submitTask({
      path: pathClean(pathJoin(props.path, file.name)),
      file,
      override,
    })
  }
  showTaskManager()
}

const uploadFile = () => {
  fileEl.value.click()
}

const createEmptyFile = () => {
  input({
    title: t('p.new_entry.create_file'),
    validator: {
      pattern: /^[^/]+$/,
      message: t('p.new_entry.invalid_filename'),
    },
    onOk: async (text) => {
      try {
        await uploadManager.upload(
          {
            path: pathClean(pathJoin(props.path, text)),
            file: '',
            override: false,
          },
          true
        )
        emit('update')
      } catch (e) {
        alert(e.message).catch(() => {})
        throw e
      }
    },
  })
}

const createDir = () => {
  input({
    title: t('p.new_entry.create_folder'),
    validator: {
      pattern: /^[^/]+$/,
      message: t('p.new_entry.invalid_folder_name'),
    },
    onOk: async (text) => {
      try {
        await makeDir(pathClean(pathJoin(props.path, text)))
        emit('update')
      } catch (e) {
        alert(e.message).catch(() => {})
        throw e
      }
    },
  })
}

const onTasksChanged = ({ tasks: tasks_, task }) => {
  tasks.value = tasks_
  updateTasksSummary()
  if (task && task.status === STATUS_COMPLETED) {
    if (props.path === dir(task.task.path)) {
      emit('update')
    }
  }
}

const startTask = (task) => {
  uploadManager.startTask(task.id)
}

const pauseTask = (task) => {
  uploadManager.pauseTask(task.id)
}

const stopTask = async (task) => {
  try {
    await confirm(t('p.new_entry.confirm_stop_task'))
  } catch {
    return
  }
  uploadManager.stopTask(task.id)
}

const removeTask = async (task) => {
  try {
    await confirm({
      message: t('p.new_entry.confirm_remove_task'),
      confirmType: 'danger',
    })
  } catch {
    return
  }
  uploadManager.removeTask(task.id, true)
}

const updateTasksSummary = () => {
  const completed = tasks.value.filter(
    (t) => t.status === STATUS_COMPLETED
  ).length
  uploadStatus.value = { completed, total: tasks.value.length }
}

const confirmFileExists = async (file) => {
  try {
    const all = (
      await dialog(FileExistsDialog, {
        title: t('p.new_entry.file_exists'),
        confirmText: t('p.new_entry.skip'),
        cancelText: t('p.new_entry.override'),
        cancelType: 'danger',
        filename: file.name,
      })
    ).all
    return { all, override: false }
  } catch (e) {
    if (!e) return { all: false, override: false }
    return { all: e.all, override: true }
  }
}

const newButtonClicked = ({ button }) => {
  ;({
    createDir,
    uploadFile,
    createEmptyFile,
  }[button.fn]())
}

const showTaskManager = () => {
  taskManagerButtonShowing.value = false
  taskManagerShowing.value = true
}

const taskManagerClosed = () => {
  taskManagerButtonShowing.value = true
}

const hideTaskManager = () => {
  taskManagerShowing.value = false
}

const onWindowUnload = (e) => {
  if (uploadStatus.value.completed < uploadStatus.value.total) {
    e.preventDefault()
    e.returnValue = ''
  }
}

let dragLeaveTimeout

const onDragEnter = (e) => {
  if (props.readonly) return
  e.preventDefault()
  clearTimeout(dragLeaveTimeout)
  toggleDropZoneActive(true)
}

const onDragLeave = (e) => {
  e.preventDefault()
  clearTimeout(dragLeaveTimeout)
  dragLeaveTimeout = setTimeout(() => {
    toggleDropZoneActive(false)
  }, 100)
}

const toggleDropZoneActive = (active) => {
  dropZoneActive.value = active
}

onBeforeUnmount(() => {
  uploadManager.off('taskChanged', onTasksChanged)
  window.removeEventListener('beforeunload', onWindowUnload)

  window.removeEventListener('dragover', onDragEnter)
  window.removeEventListener('dragleave', onDragLeave)
  window.removeEventListener('drop', onItemsDropped)
})

onBeforeMount(() => {
  tasks.value = uploadManager.getTasks()
  updateTasksSummary()
  uploadManager.on('taskChanged', onTasksChanged)

  window.addEventListener('beforeunload', onWindowUnload)

  window.addEventListener('dragover', onDragEnter)
  window.addEventListener('dragleave', onDragLeave)
  window.addEventListener('drop', onItemsDropped)
})
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

<template>
  <div class="new-entry-area">
    <FloatButton
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
        <Icon svg="#icon-add1" />
      </span>
      <template #new-file>
        <Icon svg="#icon-new-file" />
      </template>
      <template #upload-file>
        <Icon svg="#icon-upload-file" />
      </template>
      <template #new-folder>
        <Icon svg="#icon-new-folder" />
      </template>
    </FloatButton>

    <DialogView
      v-model:show="taskManagerShowing"
      :title="t('p.new_entry.upload_tasks')"
      esc-close
      overlay-close
      transition="tm-dialog"
      @closed="taskManagerClosed"
    >
      <TaskManager
        class="task-manager"
        :tasks="tasks"
        @navigate="hideTaskManager"
        @start="startTask"
        @pause="pauseTask"
        @stop="stopTask"
        @remove="removeTask"
      />
    </DialogView>

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
<script setup lang="ts">
import { makeDir } from '@/api'
import { UploadManager, UploadMangerEvents } from '@/api/upload-manager'
import { STATUS_COMPLETED, UploadTaskItem } from '@/api/upload-manager/task'
import { FloatButtonItem } from '@/components/FloatButton'
import { Entry } from '@/types'
import { dir, isParentPath, pathClean, pathJoin } from '@/utils'
import {
  getDataTransferFiles,
  getFileEntries,
  isDataTransferHasFiles,
  ResolvedEntry,
  resolveEntries,
  wrapFile,
} from '@/utils/file'
import { alert, confirm, input, loading } from '@/utils/ui-utils'
import { onBeforeMount, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import TaskManager from './TaskManager/index.vue'

const uploadManager = new UploadManager({ concurrent: 3 })

const { t } = useI18n()

const props = defineProps({
  path: {
    type: String,
    required: true,
  },
  entries: {
    type: Array as PropType<Entry[]>,
  },
  readonly: {
    type: Boolean,
  },
})

const emit = defineEmits<{ (e: 'update'): void }>()

const floatMenuShowing = ref(false)

const taskManagerShowing = ref(false)
const taskManagerButtonShowing = ref(true)

const uploadStatus = ref({ completed: 0, total: 0 })

const tasks = ref<UploadTaskItem[]>([])

const dropZoneActive = ref(false)

const fileEl = ref<HTMLInputElement | null>(null)

const onFilesChosen = () => {
  const files = Array.from(fileEl.value!.files!)
  fileEl.value!.value = ''
  submitUploadTasks(files.map(wrapFile))
}

const submitUploadTasks = async (files: ResolvedEntry[]) => {
  const flattenFiles = getFileEntries(files)
  if (!flattenFiles.length) return

  let override = true
  try {
    await confirm({
      message: t('p.new_entry.override_confirm'),
      confirmType: 'danger',
    })
  } catch {
    override = false
  }

  for (const file of flattenFiles) {
    uploadManager.submitTask({
      path: pathClean(pathJoin(props.path, file.path)),
      file: file.file,
      override,
    })
  }
  showTaskManager()
}

const handleDataTransfer = async (
  dt: DataTransfer,
  before?: () => Promise<void>
) => {
  try {
    const files = getDataTransferFiles(dt)

    await before?.()

    loading(true)
    let p: Promise<void> = Promise.resolve()

    const entries = await resolveEntries(files, (total) => {
      loading({
        text: t('p.new_entry.resolve_file', { n: total }),
        cancelType: 'warning',
        onCancel: () => {
          p = Promise.reject()
        },
      })
      return p
    })

    submitUploadTasks(entries)
  } catch {
    // ignore
  } finally {
    loading()
  }
}

const uploadFile = () => {
  fileEl.value!.click()
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
            file: new Blob([''], { type: 'text/plain' }),
            override: false,
          },
          true
        )
        emit('update')
      } catch (e: any) {
        alert(e.message).catch(() => {
          /* ignore */
        })
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
      } catch (e: any) {
        alert(e.message).catch(() => {
          /* ignore */
        })
        throw e
      }
    },
  })
}

const onTasksChanged = ({
  tasks: tasks_,
  task,
}: UploadMangerEvents['taskChanged']) => {
  tasks.value = tasks_
  updateTasksSummary()
  if (task?.status === STATUS_COMPLETED) {
    if (
      props.path === dir(task.task.path) ||
      (isParentPath(task.task.path, props.path) &&
        !props.entries?.find((e) => isParentPath(task.task.path, e.path)))
    ) {
      emit('update')
    }
  }
}

const startTask = (task: UploadTaskItem) => {
  uploadManager.startTask(task.id)
}

const pauseTask = (task: UploadTaskItem) => {
  uploadManager.pauseTask(task.id)
}

const stopTask = async (task: UploadTaskItem) => {
  try {
    await confirm(t('p.new_entry.confirm_stop_task'))
  } catch {
    return
  }
  uploadManager.stopTask(task.id)
}

const removeTask = async (task: UploadTaskItem) => {
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

const newButtonClicked = ({ button }: { button: FloatButtonItem }) => {
  ;((
    {
      createDir,
      uploadFile,
      createEmptyFile,
    } as any
  )[button.fn]())
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

const onWindowUnload = (e: BeforeUnloadEvent) => {
  if (uploadStatus.value.completed < uploadStatus.value.total) {
    e.preventDefault()
    e.returnValue = ''
  }
}

let dragLeaveTimeout: number

const onDragEnter = (e: DragEvent) => {
  if (props.readonly) return

  if (!e.dataTransfer) return
  if (!isDataTransferHasFiles(e.dataTransfer)) return

  e.preventDefault()
  clearTimeout(dragLeaveTimeout)
  toggleDropZoneActive(true)
}

const onDragLeave = (e: DragEvent) => {
  e.preventDefault()
  clearTimeout(dragLeaveTimeout)
  dragLeaveTimeout = setTimeout(() => {
    toggleDropZoneActive(false)
  }, 100) as unknown as number
}

const onItemsDropped = (e: DragEvent) => {
  toggleDropZoneActive(false)
  if (e.dataTransfer) {
    e.preventDefault()
    handleDataTransfer(e.dataTransfer)
  }
}

const onPaste = (e: ClipboardEvent) => {
  if (!e.clipboardData) return
  if (!isDataTransferHasFiles(e.clipboardData)) return
  e.preventDefault()

  handleDataTransfer(e.clipboardData, () =>
    confirm(t('p.new_entry.upload_clipboard'))
  )
}

const toggleDropZoneActive = (active: boolean) => {
  dropZoneActive.value = active
}

onBeforeUnmount(() => {
  uploadManager.off('taskChanged', onTasksChanged)
  window.removeEventListener('beforeunload', onWindowUnload)

  window.removeEventListener('dragover', onDragEnter)
  window.removeEventListener('dragleave', onDragLeave)
  window.removeEventListener('drop', onItemsDropped)

  document.removeEventListener('paste', onPaste)
})

onBeforeMount(() => {
  tasks.value = uploadManager.getTasks()
  updateTasksSummary()
  uploadManager.on('taskChanged', onTasksChanged)

  window.addEventListener('beforeunload', onWindowUnload)

  window.addEventListener('dragover', onDragEnter)
  window.addEventListener('dragleave', onDragLeave)
  window.addEventListener('drop', onItemsDropped)

  document.addEventListener('paste', onPaste)
})
</script>
<style lang="scss">
.new-entry-area {
  position: fixed;
  z-index: 10;

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

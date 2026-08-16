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
      <span
        class="icon-new-item glass-surface"
        :class="{ active: floatMenuShowing }"
        data-surface="glass"
      >
        <Icon name="plus" />
      </span>
      <template #new-file>
        <Icon name="new-file" />
      </template>
      <template #upload-file>
        <Icon name="upload-file" />
      </template>
      <template #new-folder>
        <Icon name="new-folder" />
      </template>
    </FloatButton>

    <DialogView
      v-model:show="taskManagerShowing"
      :title="t('p.new_entry.upload_tasks')"
      esc-close
      overlay-close
      @closed="taskManagerClosed"
    >
      <TaskManager
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
      class="button-task-manager glass-surface"
      data-ui="button"
      data-variant="plain"
      data-surface="glass"
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

    <div
      v-if="dropZoneActive"
      class="drop-zone-indicator"
      data-ui="drop-zone-indicator"
    >
      <div
        class="drop-zone-indicator__content glass-surface"
        data-ui="drop-zone"
        data-surface="glass"
      >
        <Icon name="upload-file" />
        <strong>
          {{
            t('p.new_entry.drop_tip', {
              target: currentPathLabel,
            })
          }}
        </strong>
      </div>
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
import { computed, onBeforeMount, onBeforeUnmount, ref } from 'vue'
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
const currentPathLabel = computed(() =>
  props.path ? `/${props.path}` : t('app.root_path')
)

const fileEl = ref<HTMLInputElement | null>(null)

const onFilesChosen = () => {
  const files = Array.from(fileEl.value!.files!)
  fileEl.value!.value = ''
  submitUploadTasks(files.map(wrapFile))
}

const submitUploadTasks = async (files: ResolvedEntry[]) => {
  const flattenFiles = getFileEntries(files)
  if (!flattenFiles.length) return

  for (const file of flattenFiles) {
    uploadManager.submitTask({
      path: pathClean(pathJoin(props.path, file.path)),
      file: file.file,
    })
  }
  showTaskManager()
}

const handleDataTransfer = async (
  dt: DataTransfer,
  before?: () => Promise<void>
) => {
  try {
    const droppedFiles = getDataTransferFiles(dt)
    if (
      droppedFiles.entries.length === 0 &&
      droppedFiles.files.length === 0
    ) {
      return
    }

    await before?.()

    loading(true)
    let p: Promise<void> = Promise.resolve()

    const entries = await resolveEntries(droppedFiles.entries, (total) => {
      loading({
        text: t('p.new_entry.resolve_file', { n: total }),
        cancelType: 'warning',
        onCancel: () => {
          p = Promise.reject()
        },
      })
      return p
    })

    submitUploadTasks([
      ...droppedFiles.files.map(wrapFile),
      ...entries,
    ])
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

const onDragEnter = (e: DragEvent) => {
  if (props.readonly) return
  if (!e.dataTransfer) return
  if (!isDataTransferHasFiles(e.dataTransfer)) return

  externalDragDepth += 1
  toggleDropZoneActive(true)
}

const onDragOver = (e: DragEvent) => {
  if (props.readonly) return
  if (!e.dataTransfer) return
  if (!isDataTransferHasFiles(e.dataTransfer)) return

  e.dataTransfer.dropEffect = 'copy'
  e.preventDefault()
  toggleDropZoneActive(true)
}

let externalDragDepth = 0

const onDragLeave = () => {
  if (!dropZoneActive.value) return
  externalDragDepth = Math.max(0, externalDragDepth - 1)
  if (externalDragDepth === 0) {
    toggleDropZoneActive(false)
  }
}

const onItemsDropped = (e: DragEvent) => {
  externalDragDepth = 0
  toggleDropZoneActive(false)
  if (props.readonly) return
  if (!e.dataTransfer) return
  if (!isDataTransferHasFiles(e.dataTransfer)) return

  e.preventDefault()
  handleDataTransfer(e.dataTransfer)
}

const onPaste = (e: ClipboardEvent) => {
  if (props.readonly) return

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

  window.removeEventListener('dragenter', onDragEnter)
  window.removeEventListener('dragover', onDragOver)
  window.removeEventListener('dragleave', onDragLeave)
  window.removeEventListener('drop', onItemsDropped)

  document.removeEventListener('paste', onPaste)
})

onBeforeMount(() => {
  tasks.value = uploadManager.getTasks()
  updateTasksSummary()
  uploadManager.on('taskChanged', onTasksChanged)

  window.addEventListener('beforeunload', onWindowUnload)

  window.addEventListener('dragenter', onDragEnter)
  window.addEventListener('dragover', onDragOver)
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
    box-shadow: var(--shadow-control);
    background-color: var(--color-bg-glass);

    .icon {
      display: block;
      transition: transform var(--motion-duration-normal)
        var(--motion-easing-standard);
      fill: var(--color-text-muted);
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
    background-color: var(--color-bg-glass);
    color: var(--color-text);

    border: none;
    box-shadow: var(--shadow-control);
    cursor: pointer;
    -webkit-user-select: none;
    user-select: none;
  }

  .drop-zone-indicator {
    position: fixed;
    inset: 0;
    z-index: 1000;
    pointer-events: none;
    background-color: var(--color-overlay);

    display: flex;
    justify-content: center;
    align-items: center;
    padding: 24px;
  }

  .drop-zone-indicator__content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    max-width: min(520px, calc(100vw - 64px));
    padding: 24px 32px;
    border-radius: 12px;
    background-color: var(--color-bg-glass);
    color: var(--color-text);
    box-shadow: var(--shadow-elevated);
    text-align: center;
    font-size: 20px;

    .icon {
      width: 56px;
      height: 56px;
    }
  }

  .hidden-input-file {
    display: none;
  }
}
</style>

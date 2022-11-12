import { arrayRemove } from '@/utils'
// eslint-disable-next-line no-unused-vars
import UploadTask, {
  UploadTaskItem,
  STATUS_UPLOADING,
  STATUS_CREATED,
  STATUS_ERROR,
  STATUS_COMPLETED,
  STATUS_MASK_PENDING,
  STATUS_MASK_FREEZED,
  STATUS_STARTING,
  TaskDef,
  TaskChangeEvent,
} from './task'
import DispatcherUploadTask from './upload-providers/dispatcher'

export type UploadTaskFinishedCallback = (
  this: UploadManager,
  task: UploadTask,
  data?: any
) => void

export interface UploadManagerConfig {
  concurrent: number
}

export type UploadManagerEventCallback = (
  this: UploadManager,
  ...args: any[]
) => void

export interface UploadMangerEvents {
  taskChanged: {
    task?: UploadTask
    tasks: UploadTaskItem[]
  }
}

export class UploadManager {
  private _config: UploadManagerConfig
  private _tasks: UploadTask[] = []
  private _taskMap: Record<number, UploadTask> = {}
  private _events: Record<string, UploadManagerEventCallback[]> = {}
  private _taskFinishedCallbacks: Record<number, UploadTaskFinishedCallback> =
    {}
  private _idSeq = 0

  constructor(config: Partial<UploadManagerConfig>) {
    this._config = { concurrent: config.concurrent ?? 3 }
    this._taskChanged = this._taskChanged.bind(this)
  }

  submitTask(taskDef: TaskDef) {
    const id = this._idSeq++
    const task = new DispatcherUploadTask(id, this._taskChanged, taskDef)
    this._putTask(task)
    return id
  }

  upload(taskDef: TaskDef, removeIfFinished?: boolean) {
    return new Promise<void>((resolve, reject) => {
      const id = this.submitTask(taskDef)
      this._taskFinishedCallbacks[id] = (task, e) => {
        if (task.status === STATUS_COMPLETED) {
          resolve()
        } else {
          reject(
            task.status === STATUS_ERROR
              ? e.data
              : { status: task.status, message: 'task stopped' }
          )
        }
        if (removeIfFinished) {
          this.removeTask(id)
        }
      }
      this.startTask(id)
    })
  }

  startTask(id: number) {
    this._taskMap[id] && this._taskMap[id].start()
  }

  pauseTask(id: number) {
    this._taskMap[id] && this._taskMap[id].pause()
  }

  stopTask(id: number) {
    this._taskMap[id] && this._taskMap[id].stop()
  }

  removeTask(id: number, force?: boolean) {
    const task = this._taskMap[id]
    if (!task) return false
    return this._removeTask(task, !!force)
  }

  private rescheduleTasks() {
    const uploading = this._tasks.filter(
      (t) => t.status === STATUS_UPLOADING || t.status === STATUS_STARTING
    ).length
    const needStart = this._config.concurrent - uploading
    if (needStart <= 0) return
    this._tasks
      .filter((t) => t.status === STATUS_CREATED)
      .slice(0, needStart)
      .forEach((t) => t.start())
  }

  getTasks() {
    return this._tasks.map((t) => new UploadTaskItem(t))
  }

  on(event: string, fn: UploadManagerEventCallback) {
    const events = this._events[event] || []
    events.push(fn)
    this._events[event] = events
  }

  off(event: string, fn: UploadManagerEventCallback) {
    const events = this._events[event]
    if (events) {
      arrayRemove(events, (e) => e === fn)
    }
  }

  private _emitEvent<T extends keyof UploadMangerEvents>(
    event: T,
    arg: UploadMangerEvents[T]
  ) {
    const events = this._events[event]
    if (events) {
      events.forEach((fn) => {
        fn.call(this, arg)
      })
    }
  }

  private _taskChanged(e: TaskChangeEvent) {
    const task = e.task
    if (!this._taskMap[task.id]) return

    this._emitTaskChanged(task)

    if (task.isStatus(STATUS_MASK_FREEZED)) {
      const cb = this._taskFinishedCallbacks[task.id]
      if (typeof cb === 'function') {
        cb.call(this, task, e)
        delete this._taskFinishedCallbacks[task.id]
      }
    }
  }

  private _putTask(task: UploadTask) {
    this._tasks.push(task)
    this._taskMap[task.id] = task
    this._emitTaskChanged(task)
  }

  private _removeTask(task: UploadTask, force: boolean) {
    if (!force && task.isStatus(STATUS_MASK_PENDING)) {
      return false
    }
    const index = this._tasks.findIndex((t) => t.id === task.id)
    if (index === -1) return false
    this._tasks.splice(index, 1)
    delete this._taskMap[task.id]
    this._emitTaskChanged()
    return true
  }

  private _emitTaskChanged(task?: UploadTask) {
    const tasks = this.getTasks()
    this._emitEvent('taskChanged', { task, tasks })

    if (task && task.status !== STATUS_UPLOADING) {
      setTimeout(() => {
        this.rescheduleTasks()
      }, 0)
    }
  }
}

const defaultUploadManager = new UploadManager({})

if (process.env.NODE_ENV === 'development') {
  ;(window as any).uploadManager = defaultUploadManager
}

export default defaultUploadManager

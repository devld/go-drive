import { IS_DEBUG } from '@/utils'
import http from '../http'

export interface TaskChangeEvent {
  task: UploadTask
  data?: O
}

export type TaskChangeListener = (
  this: UploadTask,
  event: TaskChangeEvent
) => void

export interface TaskDef {
  path: string
  file?: Blob
  size?: number
  override?: boolean
}

export interface UploadProgress {
  loaded: number
  total: number
}

function processTaskDef(task: TaskDef): TaskDef {
  let file = task.file
  if (typeof file !== 'string' && !(file instanceof Blob)) {
    throw new Error('invalid file')
  }
  if (typeof file === 'string') {
    file = new Blob([file], { type: 'text/plain' })
  }
  return { ...task, file, size: file.size }
}

const matchStatus = (mask: number, status: number) => !!(mask & status)

/** task created, but not started */
export const STATUS_CREATED = 1 << 0
/** task starting */
export const STATUS_STARTING = 1 << 1
/** task is uploading */
export const STATUS_UPLOADING = 1 << 2
/** task paused */
export const STATUS_PAUSED = 1 << 3
/** task completed successfully */
export const STATUS_COMPLETED = 1 << 4
/** task stopped */
export const STATUS_STOPPED = 1 << 5
/** task error */
export const STATUS_ERROR = 1 << 6

export const STATUS_MASK_PENDING =
  STATUS_CREATED | STATUS_STARTING | STATUS_PAUSED | STATUS_UPLOADING
export const STATUS_MASK_FREEZED =
  STATUS_COMPLETED | STATUS_ERROR | STATUS_STOPPED

export const STATUS_MASK_CAN_START =
  STATUS_CREATED | STATUS_PAUSED | STATUS_ERROR | STATUS_STOPPED
export const STATUS_MASK_CAN_PAUSE = STATUS_STARTING | STATUS_UPLOADING
export const STATUS_MASK_CAN_STOP =
  STATUS_STARTING | STATUS_UPLOADING | STATUS_PAUSED | STATUS_ERROR

export default abstract class UploadTask {
  private _progress?: UploadProgress
  private _error?: any

  private _status = STATUS_CREATED
  private _task: TaskDef

  constructor(
    private _id: number,
    private changeListener: TaskChangeListener,
    task: Readonly<TaskDef>,
    protected _config?: O
  ) {
    this._task = processTaskDef(task)
  }

  /** start or resume this task */
  async start(): Promise<false | void> {
    if (!this.isStatus(STATUS_MASK_CAN_START)) return false
  }

  /** pause this task */
  async pause(): Promise<false | void> {
    if (!this.isStatus(STATUS_MASK_CAN_PAUSE)) return false
  }

  /** stop (cancel) this task */
  async stop(): Promise<false | void> {
    if (!this.isStatus(STATUS_MASK_CAN_STOP)) return false
  }

  isStatus(mask: number) {
    return matchStatus(mask, this._status)
  }

  protected _onChange(status: number, data?: any) {
    if (IS_DEBUG) {
      console.debug('update status change:', status, data)
    }
    this._status = status
    if (status === STATUS_UPLOADING) {
      this._progress = Object.freeze({
        loaded: data!.loaded,
        total: data!.total,
      })
    }
    if (status === STATUS_COMPLETED) {
      this._task.file = undefined
    }
    if (status === STATUS_ERROR) {
      this._error = data
    }

    this.changeListener({ task: this, data })
  }

  protected uploadCallback<T = any>(data: O<string>): Promise<T> {
    return http.post(`/upload/${this.task.path}`, data, {
      params: { override: true, size: this.task.size },
    })
  }

  get id() {
    return this._id
  }
  get task(): Readonly<TaskDef> {
    return this._task
  }
  get status() {
    return this._status
  }
  get progress(): Readonly<UploadProgress> | undefined {
    return this._progress
  }
  get error(): any | undefined {
    return this._error
  }
}

export class UploadTaskItem {
  public readonly id: number
  public readonly task: TaskDef
  public readonly status: number
  public readonly progress?: UploadProgress
  public readonly error?: any

  constructor(task: UploadTask) {
    this.id = task.id
    this.task = task.task
    this.status = task.status
    this.progress = task.progress
    this.error = task.error
    Object.freeze(this)
  }

  isStatus(mask: number) {
    return matchStatus(mask, this.status)
  }
}

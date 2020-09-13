/**
 * @typedef TaskDef
 * @property {string} path file path
 * @property {any} file file payload
 * @property {number} [size] payload size (bytes)
 * @property {boolean} [override] override file if it exists
 */

/**
 * @typedef TaskChangeEvent
 * @property {UploadTask} task task
 * @property {any} [data] payload
 */

/**
 * @callback TaskChangeListener
 * @param {TaskChangeEvent} event
 */

/**
 * @param {TaskDef} task
 * @returns {TaskDef}
 */
function processTaskDef (task) {
  let file = task.file
  if (typeof (file) !== 'string' && !(file instanceof Blob)) {
    throw new Error('invalid file')
  }
  if (typeof (file) === 'string') {
    file = new Blob([file], { type: 'text/plain' })
  }
  return { ...task, file, size: file.size }
}

const matchStatus = (mask, status) => !!(mask & status)

/**
 * task created, but not started
 */
export const STATUS_CREATED = 1 << 0
/**
 * task starting
 */
export const STATUS_STARTING = 1 << 1
/**
 * task is uploading
 */
export const STATUS_UPLOADING = 1 << 2
/**
 * task paused
 */
export const STATUS_PAUSED = 1 << 3
/**
 * task completed successfully
 */
export const STATUS_COMPLETED = 1 << 4
/**
 * task stopped
 */
export const STATUS_STOPPED = 1 << 5
/**
 * task error
 */
export const STATUS_ERROR = 1 << 6

export const STATUS_MASK_PENDING = STATUS_CREATED | STATUS_STARTING | STATUS_PAUSED | STATUS_UPLOADING
export const STATUS_MASK_FREEZED = STATUS_COMPLETED | STATUS_ERROR | STATUS_STOPPED

export const STATUS_MASK_CAN_START = STATUS_CREATED | STATUS_PAUSED | STATUS_ERROR | STATUS_STOPPED
export const STATUS_MASK_CAN_PAUSE = STATUS_STARTING | STATUS_UPLOADING
export const STATUS_MASK_CAN_STOP = STATUS_STARTING | STATUS_UPLOADING | STATUS_PAUSED

export default class UploadTask {
  /**
   * @type {number}
   */
  _id

  /**
   * @type {TaskDef}
   */
  _task

  /**
   * @type {any}
   */
  _config

  /**
   * @type {number}
   */
  _status

  /**
   * @type {{loaded: number, total: number}|undefined}
   */
  _progress

  /**
   * @type {TaskChangeListener}
   */
  _changeListener

  /**
   * @param {number} id task id
   * @param {TaskChangeListener} changeListener task changed listener
   * @param {TaskDef} task task definition
   * @param {any} [config] task specified config
   */
  constructor (id, changeListener, task, config) {
    if (new.target === UploadTask) {
      throw new Error('Cannot construct abstract UploadTask')
    }
    this._id = id
    this._changeListener = changeListener
    this._status = STATUS_CREATED
    this._task = processTaskDef(task)
    this._config = config
  }

  /**
   * start or resume this task
   * @returns {boolean}
   */
  start () {
    if (!this.isStatus(STATUS_MASK_CAN_START)) return false
  }

  /**
   * pause this task
   * @returns {boolean}
   */
  pause () {
    if (!this.isStatus(STATUS_MASK_CAN_PAUSE)) return false
  }

  /**
   * stop (cancel) this task
   */
  stop () {
    if (!this.isStatus(STATUS_MASK_CAN_STOP)) return false
  }

  get id () { return this._id }
  get status () { return this._status }
  get progress () { return this._progress }
  get task () { return this._task }

  isStatus (mask) {
    return matchStatus(mask, this._status)
  }

  /**
   * @param {string} status
   * @param {any} data
   */
  _onChange (status, data) {
    this._status = status
    if (status === STATUS_UPLOADING) {
      this._progress = Object.freeze({ loaded: data.loaded, total: data.total })
    }
    if (status === STATUS_COMPLETED) {
      this._task.file = null
    }

    this._changeListener({ task: this, data })
  }
}

export class UploadTaskItem {
  /**
   * @type {number}
   */
  id
  /**
   * @type {TaskDef}
   */
  task
  /**
   * @type {number}
   */
  status
  /**
   * @type {{ loaded: number, total: number }}
   */
  progress
  /**
   * @param {UploadTask} task
   */
  constructor (task) {
    this.id = task.id
    this.task = task.task
    this.status = task.status
    this.progress = task.progress
    Object.freeze(this)
  }

  isStatus (mask) {
    return matchStatus(mask, this.status)
  }
}

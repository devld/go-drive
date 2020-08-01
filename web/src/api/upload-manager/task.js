/**
 * @typedef TaskDef
 * @property {string} path file path
 * @property {any} file file payload
 * @property {number} [size] payload size (bytes)
 * @property {boolean} [overwrite] overwrite file if it exists
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
 * task is uploading
 */
export const STATUS_UPLOADING = 1 << 1
/**
 * task paused
 */
export const STATUS_PAUSED = 1 << 2
/**
 * task completed successfully
 */
export const STATUS_COMPLETED = 1 << 3
/**
 * task stopped
 */
export const STATUS_STOPPED = 1 << 4
/**
 * task error
 */
export const STATUS_ERROR = 1 << 5

export const STATUS_MASK_PENDING = STATUS_CREATED | STATUS_PAUSED | STATUS_UPLOADING
export const STATUS_MASK_FREEZED = STATUS_COMPLETED | STATUS_ERROR | STATUS_STOPPED

export const STATUS_MASK_CAN_START = STATUS_CREATED | STATUS_PAUSED | STATUS_ERROR | STATUS_STOPPED
export const STATUS_MASK_CAN_PAUSE = STATUS_UPLOADING
export const STATUS_MASK_CAN_STOP = STATUS_UPLOADING | STATUS_PAUSED

export default class UploadTask {
  /**
   * @type {number}
   */
  _id

  /**
   * @type {UploadManager}
   */
  _manager

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
   * @param {number} id task id
   * @param {UploadManager} manager upload manager
   * @param {TaskDef} task task definition
   * @param {any} [config] task specified config
   */
  constructor (id, manager, task, config) {
    if (new.target === UploadTask) {
      throw new Error('Cannot construct abstract UploadTask')
    }
    this._id = id
    this._manager = manager
    this._status = STATUS_CREATED
    this._task = processTaskDef(task)
    this._config = config
    this._taskInit(task, config)
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

  /**
   * task init
   * @param {TaskDef} task
   * @param {any} config
   */
  _taskInit (task, config) {
  }

  get id () { return this._id }
  get status () { return this._status }
  get progress () { return this._progress }
  get task () { return this._task }

  isStatus (mask) {
    return matchStatus(mask, this._status)
  }

  _onStart () {
    this._status = STATUS_UPLOADING
    this._manager._taskChanged(this)
  }

  _onPause () {
    this._status = STATUS_PAUSED
    this._manager._taskChanged(this)
  }

  _onStop () {
    this._status = STATUS_STOPPED
    this._manager._taskChanged(this)
  }

  /**
   * @param {number} loaded uploaded size
   * @param {number} total total size
   */
  _onProgress (loaded, total) {
    this._status = STATUS_UPLOADING
    this._progress = Object.freeze({ loaded, total })
    this._manager._taskChanged(this)
  }

  _onError (e) {
    this._status = STATUS_ERROR
    this._manager._taskChanged(this, e)
  }

  _onComplete () {
    this._task.file = null
    this._status = STATUS_COMPLETED
    this._manager._taskChanged(this)
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

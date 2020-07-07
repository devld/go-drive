// eslint-disable-next-line no-unused-vars
import UploadManager from '.'

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

export default class UploadTask {
  /**
   * task created, but not started
   */
  static STATUS_CREATED = 0
  /**
   * task is uploading
   */
  static STATUS_UPLOADING = 1
  /**
   * task paused
   */
  static STATUS_PAUSED = 2
  /**
   * task completed successfully
   */
  static STATUS_COMPLETED = 3
  /**
   * task stopped
   */
  static STATUS_STOPPED = 4
  /**
   * task error
   */
  static STATUS_ERROR = 5

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
    this._status = UploadTask.STATUS_CREATED
    this._task = processTaskDef(task)
    this._config = config
    this._taskInit(task, config)
  }

  /**
   * start or resume this task
   */
  start () {
    if (this.status === UploadTask.STATUS_COMPLETED) {
      throw new Error('[UploadTask] task completed')
    }
  }

  /**
   * pause this task
   */
  pause () {
  }

  /**
   * stop (cancel) this task
   */
  stop () {
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

  get isFinished () {
    return this._status === UploadTask.STATUS_COMPLETED ||
      this._status === UploadTask.STATUS_ERROR ||
      this._status === UploadTask.STATUS_STOPPED
  }

  get isPending () {
    return this._status === UploadTask.STATUS_PAUSED ||
      this._status === UploadTask.STATUS_UPLOADING
  }

  _onStart () {
    this._status = UploadTask.STATUS_UPLOADING
    this._manager._taskChanged(this)
  }

  _onPause () {
    this._status = UploadTask.STATUS_PAUSED
    this._manager._taskChanged(this)
  }

  _onStop () {
    this._status = UploadTask.STATUS_STOPPED
    this._manager._taskChanged(this)
  }

  /**
   * @param {number} loaded uploaded size
   * @param {number} total total size
   */
  _onProgress (loaded, total) {
    this._status = UploadTask.STATUS_UPLOADING
    this._progress = Object.freeze({ loaded, total })
    this._manager._taskChanged(this)
  }

  _onError (e) {
    this._status = UploadTask.STATUS_ERROR
    this._manager._taskChanged(this, e)
  }

  _onComplete () {
    this._task.file = null
    this._status = UploadTask.STATUS_COMPLETED
    this._manager._taskChanged(this)
  }
}

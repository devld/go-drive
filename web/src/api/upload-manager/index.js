import { arrayRemove } from '@/utils'
// eslint-disable-next-line no-unused-vars
import UploadTask, {
  UploadTaskItem, STATUS_UPLOADING, STATUS_CREATED,
  STATUS_ERROR, STATUS_COMPLETED, STATUS_MASK_PENDING, STATUS_MASK_FREEZED
} from './task'
import { getUploadConfig } from '@/api'

import LocalUploadTask from './upload-providers/local'

/**
 * @type {Object.<string, typeof UploadTask>}
 */
const TASK_PROVIDERS = {
  local: LocalUploadTask
}

/**
 * @callback UploadManagerEventCallback
 * @this UploadManager
 */

/**
 * @callback UploadTaskFinishedCallback
 * @param {UploadTask} task
 * @param {any} [e]
 * @this UploadManager
 */

/**
 * @typedef UploadManagerConfig
 * @property {number} concurrent max concurrent uploading tasks
 */

export class UploadManager {
  /**
   * @type {number}
   */
  _idSeq

  /**
   * @type {Array.<UploadTask>}
   */
  _tasks

  /**
   * @type {Object.<number, UploadTask>}
   */
  _taskMap
  /**
   * @type {Object.<string, Array.<Function>>}
   */
  _events

  /**
   * @type {Object.<number, UploadTaskFinishedCallback>}
   */
  _taskFinishedCallbacks

  /**
   * @type {UploadManagerConfig}
   */
  _config

  /**
   * @param {UploadManagerConfig} config
   */
  constructor (config) {
    this._tasks = []
    this._taskMap = {}
    this._events = {}
    this._taskFinishedCallbacks = {}
    this._idSeq = 0
    this._config = { ...config }
  }

  /**
   * submit an upload task
   * @param {import('./task').TaskDef} taskDef
   * @returns {Promise.<number>} task id if successfully added
   */
  async submitTask (taskDef) {
    const uploadConfig = await getUploadConfig(taskDef.path, taskDef.overwrite)
    const ConcreteUploadTask = TASK_PROVIDERS[uploadConfig.provider]
    if (!ConcreteUploadTask) {
      throw new Error(`provider '${uploadConfig.provider}' not supported`)
    }
    const id = this._idSeq++
    const task = new ConcreteUploadTask(id, this, taskDef, uploadConfig.config)
    this._putTask(task)
    return id
  }

  /**
   * submit and start task then wait for it finish
   * @param {import('./task').TaskDef} taskDef
   * @param {boolean} [removeIfFinished]
   * @returns {Promise.<void>}
   */
  upload (taskDef, removeIfFinished) {
    return new Promise((resolve, reject) => {
      this.submitTask(taskDef).then(id => {
        this._taskFinishedCallbacks[id] = (task, e) => {
          if (task.status === STATUS_COMPLETED) {
            resolve()
          } else {
            // eslint-disable-next-line prefer-promise-reject-errors
            reject({
              status: task.status,
              message: task.status === STATUS_ERROR ? e.message : 'task stopped'
            })
          }
          if (removeIfFinished) {
            this.removeTask(id)
          }
        }
        this.startTask(id)
      }).catch(reject)
    })
  }

  /**
   * start a task
   * @param {number} id task id
   */
  startTask (id) {
    this._taskMap[id] && this._taskMap[id].start()
  }

  /**
   * pause a task
   * @param {number} id task id
   */
  pauseTask (id) {
    this._taskMap[id] && this._taskMap[id].pause()
  }

  /**
   * stop a task
   * @param {number} id task id
   */
  stopTask (id) {
    this._taskMap[id] && this._taskMap[id].stop()
  }

  /**
   * remove a finished task
   * @param {number} id task id
   * @param {boolean} [force] force remove (will stop the task)
   * @returns {boolean}
   */
  removeTask (id, force) {
    const task = this._taskMap[id]
    if (!task) return false
    return this._removeTask(task, force)
  }

  rescheduleTasks () {
    const uploading = this._tasks.filter(t => t.status === STATUS_UPLOADING).length
    const needStart = this._config.concurrent - uploading
    if (needStart <= 0) return
    this._tasks.filter(t => t.status === STATUS_CREATED)
      .slice(0, needStart).forEach(t => t.start())
  }

  /**
   * @returns {UploadTaskItem}
   */
  getTasks () {
    return this._tasks.map(t => new UploadTaskItem(t))
  }

  /**
   * @param {string} event event name
   * @param {UploadManagerEventCallback} fn event handler
   */
  on (event, fn) {
    const events = this._events[event] || []
    events.push(fn)
    this._events[event] = events
  }

  /**
 * @param {string} event event name
 * @param {UploadManagerEventCallback} fn event handler
 */
  off (event, fn) {
    const events = this._events[event]
    if (events) {
      arrayRemove(events, e => e === fn)
    }
  }

  _emitEvent (event, argsArray) {
    const events = this._events[event]
    if (events) {
      events.forEach(fn => {
        fn.apply(this, argsArray)
      })
    }
  }

  /**
   * @param {UploadTask} task the task
   * @param {any} e the error when task failed
   */
  _taskChanged (task, e) {
    if (!this._taskMap[task.id]) return

    this._emitTaskChanged(task)

    if (task.isStatus(STATUS_MASK_FREEZED)) {
      const cb = this._taskFinishedCallbacks[task.id]
      if (typeof (cb) === 'function') {
        cb.call(this, task, e)
        delete this._taskFinishedCallbacks[task.id]
      }
    }
  }

  /**
   * @param {UploadTask} task
   */
  _putTask (task) {
    this._tasks.push(task)
    this._taskMap[task.id] = task
    this._emitTaskChanged(task)
  }

  /**
   * @param {UploadTask} task
   * @param {boolean} force
   * @returns {boolean}
   */
  _removeTask (task, force) {
    if (task.isStatus(STATUS_MASK_PENDING)) {
      if (force) task.stop()
      else return false
    }
    const index = this._tasks.findIndex(t => t.id === task.id)
    if (index === -1) return false
    this._tasks.splice(index, 1)
    delete this._taskMap[task.id]
    this._emitTaskChanged()
    return true
  }

  /**
   * @param {UploadTask} task
   */
  _emitTaskChanged (task) {
    const tasks = this.getTasks()
    this._emitEvent('taskChanged', [{ task, tasks }])
    if (task && task.status !== STATUS_UPLOADING) {
      setTimeout(() => { this.rescheduleTasks() }, 0)
    }
  }
}

const defaultUploadManager = new UploadManager({ concurrent: 3 })

if (process.env.NODE_ENV === 'development') {
  window.uploadManager = defaultUploadManager
}

export default defaultUploadManager

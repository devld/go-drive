import UploadTask, { STATUS_STARTING, STATUS_ERROR } from '../task'
import { getUploadConfig } from '@/api'

import LocalUploadTask from './local'

/**
 * @type {Object.<string, typeof UploadTask>}
 */
const TASK_PROVIDERS = {
  local: LocalUploadTask
}

class DispatcherUploadTask extends UploadTask {
  /**
   * @type {UploadTask}
   */
  _targetTask

  start () {
    if (this._targetTask) return this._targetTask.start()
    if (this._started) return false
    this._initTask()
  }

  pause () {
    if (this._status === STATUS_STARTING) return false
    if (!this._targetTask) return false
    this._targetTask.pause()
  }

  stop () {
    if (!this._targetTask) return false
    this._targetTask.stop()
  }

  async _initTask () {
    this._started = true
    this._onChange(STATUS_STARTING)
    let uploadConfig
    try {
      uploadConfig = await getUploadConfig(this._task.path, this._task.overwrite)
    } catch (e) {
      this._started = false
      this._onChange(STATUS_ERROR, e)
      return
    }
    const ConcreteUploadTask = TASK_PROVIDERS[uploadConfig.provider]
    if (!ConcreteUploadTask) {
      this._onChange(STATUS_ERROR, new Error(`provider '${uploadConfig.provider}' not supported`))
      return
    }
    this._targetTask = new ConcreteUploadTask(this._id,
      this._dispatcherOnTaskChanged.bind(this), this._task, uploadConfig.config)
    this._targetTask.start()
  }

  /**
   * @param {import('../task').TaskChangeEvent} e
   */
  _dispatcherOnTaskChanged (e) {
    this._onChange(this._targetTask._status, e.data)
  }
}

export default DispatcherUploadTask

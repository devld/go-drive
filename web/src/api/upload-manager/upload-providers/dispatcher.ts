import http from '@/api/http'
import UploadTask, {
  STATUS_ERROR,
  STATUS_STARTING,
  TaskChangeEvent,
} from '../task'
import LocalUploadTask from './local'
import LocalChunkUploadTask from './local-chunk'
import S3UploadTask from './s3'
import OneDriveUploadTask from './onedrive'

const TASK_PROVIDERS: O<{
  new (...args: ConstructorParameters<typeof UploadTask>): UploadTask
}> = {
  local: LocalUploadTask,
  localChunk: LocalChunkUploadTask,
  s3: S3UploadTask,
  onedrive: OneDriveUploadTask,
}

class DispatcherUploadTask extends UploadTask {
  private _targetTask?: UploadTask
  private _started?: boolean

  override async start() {
    if (this._targetTask) return this._targetTask.start()
    if (this._started) return false
    this._initTask()
  }

  override async pause() {
    if (this.status === STATUS_STARTING) return false
    if (!this._targetTask) return false
    return this._targetTask.pause()
  }

  override async stop() {
    if (!this._targetTask) return false
    const r = await this._targetTask.stop()
    this._targetTask = undefined
    this._started = false
    return r
  }

  private async _initTask() {
    this._started = true
    this._onChange(STATUS_STARTING)
    let uploadConfig: any
    try {
      uploadConfig = await http.post(
        `/upload/${this.task.path}`,
        {},
        {
          params: { override: this.task.override, size: this.task.size },
        }
      )
    } catch (e: any) {
      this._started = false
      this._onChange(STATUS_ERROR, e)
      return
    }
    const ConcreteUploadTask = TASK_PROVIDERS[uploadConfig.provider]
    if (!ConcreteUploadTask) {
      this._onChange(
        STATUS_ERROR,
        new Error(`provider '${uploadConfig.provider}' not supported`)
      )
      return
    }
    this._targetTask = new ConcreteUploadTask(
      this.id,
      this._dispatcherOnTaskChanged.bind(this),
      this.task,
      uploadConfig.config
    )
    this._targetTask.start()
  }

  private _dispatcherOnTaskChanged(e: TaskChangeEvent) {
    this._onChange(this._targetTask!.status, e.data)
  }
}

export default DispatcherUploadTask

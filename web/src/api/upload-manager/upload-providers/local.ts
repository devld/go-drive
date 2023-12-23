/// local storage provider

import { deleteTask } from '@/api'
import http from '@/api/http'
import { Task } from '@/types'
import { taskDone } from '@/utils'
import { RequestTask } from '@/utils/http'
import UploadTask, {
  STATUS_COMPLETED,
  STATUS_ERROR,
  STATUS_STOPPED,
  STATUS_UPLOADING,
} from '../task'

/**
 * local upload task provider
 */
export default class LocalUploadTask extends UploadTask {
  private _httpTask?: RequestTask<Task<any>>
  private _waitingTask?: Task<any>

  override async start() {
    if ((await super.start()) === false) return false

    const task = http.put<Task>(`/content/${this.task.path}`, this.task.file, {
      params: { override: this.task.override ? '1' : '' },
      transformRequest: (d) => d,
      onUploadProgress: ({ loaded, total }) => {
        this._onChange(STATUS_UPLOADING, { loaded, total })
      },
    })
    this._httpTask = task
    task.then(
      (t) => {
        this._waitingTask = t
      },
      () => {
        // ignore
      }
    )

    return taskDone(task, (t) => {
      this._waitingTask = t
    })
      .then(
        () => {
          this._onChange(STATUS_COMPLETED)
        },
        (e) => {
          if (this.status === STATUS_STOPPED) return
          this._onChange(STATUS_ERROR, e)
        }
      )
      .then(() => {
        this._httpTask = undefined
      })
  }

  override async pause() {
    console.warn('[LocalUploadTask] cannot pause task')
  }

  override async stop() {
    if (this._httpTask) {
      this._onChange(STATUS_STOPPED)
      this._httpTask.cancel()
    }
    if (this._waitingTask) {
      try {
        await deleteTask(this._waitingTask.id)
      } catch {
        // ignore
      }
    }
  }
}

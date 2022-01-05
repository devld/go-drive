/// local storage provider

import axios, { ApiError } from '@/api/axios'
import { taskDone } from '@/utils'
import Axios from 'axios'
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
  start() {
    if (super.start() === false) return false

    this._axiosSource = Axios.CancelToken.source()

    return taskDone(
      axios.put(`/content/${this._task.path}`, this._task.file, {
        params: { override: this._task.override ? '1' : '' },
        cancelToken: this._axiosSource.token,
        transformRequest: null,
        onUploadProgress: ({ loaded, total }) => {
          this._onChange(STATUS_UPLOADING, { loaded, total })
        },
      }),
      () => {}
    ).then(
      () => {
        this._onChange(STATUS_COMPLETED)
      },
      (e) => {
        if (this._status === STATUS_STOPPED) return
        this._onChange(STATUS_ERROR, ApiError.from(e))
      }
    )
  }

  pause() {
    console.warn('[LocalUploadTask] cannot pause task')
  }

  stop() {
    if (this._axiosSource) {
      this._onChange(STATUS_STOPPED)
      this._axiosSource.cancel()
    }
  }
}

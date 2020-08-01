/// local storage provider

import axios from '@/api/axios'
import Axios from 'axios'
import UploadTask, { STATUS_STOPPED, STATUS_UPLOADING, STATUS_COMPLETED, STATUS_ERROR } from '../task'

/**
 * local upload task provider
 */
export default class LocalUploadTask extends UploadTask {
  /**
   * @type {import('axios').CancelTokenSource}
   */
  _axiosSource

  start () {
    if (super.start() === false) return false

    this._axiosSource = Axios.CancelToken.source()

    const formData = new FormData()
    formData.append('file', this._task.file)
    axios.put(`/content/${this._task.path}`, formData, {
      cancelToken: this._axiosSource.token,
      onUploadProgress: ({ loaded, total }) => {
        this._onChange(STATUS_UPLOADING, { loaded, total })
      }
    }).then(() => {
      this._onChange(STATUS_COMPLETED)
    }, e => {
      if (this._status === STATUS_STOPPED) return
      this._onChange(STATUS_ERROR, e)
    })
  }

  pause () {
    console.warn('[LocalUploadTask] cannot pause task')
  }

  stop () {
    if (this._axiosSource) {
      this._onChange(STATUS_STOPPED)
      this._axiosSource.cancel()
    }
  }
}

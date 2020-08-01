/// local storage provider

import axios from '@/api/axios'
import Axios from 'axios'
import UploadTask, { STATUS_STOPPED } from '../task'

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
        this._onProgress(loaded, total)
      }
    }).then(() => {
      this._onComplete()
    }, e => {
      if (this._status === STATUS_STOPPED) return
      this._onError(e)
    })
  }

  pause () {
    console.warn('[LocalUploadTask] cannot pause task')
  }

  stop () {
    if (this._axiosSource) {
      this._onStop()
      this._axiosSource.cancel()
    }
  }
}

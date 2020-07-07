/// local storage provider

import axios from '@/api/axios'
import Axios from 'axios'
import UploadTask from '../task'

/**
 * local upload task provider
 */
export default class LocalUploadTask extends UploadTask {
  /**
   * @type {import('axios').CancelTokenSource}
   */
  _axiosSource

  start () {
    super.start()
    if (this.status === UploadTask.STATUS_UPLOADING) {
      return
    }
    if (this.status === UploadTask.STATUS_CREATED ||
      this.status === UploadTask.STATUS_STOPPED ||
      this.status === UploadTask.STATUS_ERROR) {
      this._axiosSource = Axios.CancelToken.source()
    }
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
      this._onError(e)
    })
  }

  pause () {
    console.warn('[LocalUploadTask] cannot pause task')
  }

  stop () {
    if (this._axiosSource) {
      this._axiosSource.cancel()
      this._onStop()
    }
  }
}

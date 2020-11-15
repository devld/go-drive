import Axios from 'axios'
import axios from '@/api/axios'
import ChunkUploadTask from '../chunk-task'
import { STATUS_COMPLETED } from '../task'

const CHUNK_SIZE = 4 * 1024 * 1024

export default class OneDriveUploadTask extends ChunkUploadTask {
  /**
   * @type {number}
   */
  _chunkSize

  /**
   * @type {string}
   */
  _url

  _finishedResponse

  /**
   * prepare upload
   * @returns {Promise.<number} chunks count
   */
  async _prepare () {
    this._url = this._config.url
    const size = this._task.size
    this._chunkSize = CHUNK_SIZE
    this._maxConcurrent = 1
    return Math.ceil(size / CHUNK_SIZE)
  }

  /**
   * @param {number} seq seq, start from 0
   * @param {Blob} blob  chunk
   * @param {Function} onProgress progress
   */
  async _chunkUpload (seq, blob, onProgress) {
    const startByte = seq * this._chunkSize
    const endByte = Math.min((seq + 1) * this._chunkSize, this._task.size) - 1
    const resp = await this._request({
      method: 'PUT',
      url: this._url,
      data: blob,
      headers: {
        'Content-Type': 'application/octet-stream',
        'Content-Range': `bytes ${startByte}-${endByte}/${this._task.size}`
      },
      transformRequest: null,
      onUploadProgress: ({ loaded, total }) => {
        onProgress({ loaded, total })
      }
    })
    if (resp.status === 201) {
      this._finishedResponse = resp.data
    }
    return resp
  }

  /**
   * @returns {Promise.<any>} upload result
   */
  async _completeUpload () {
    if (!this._finishedResponse) throw new Error('unexpected undefined finishedResponse')
    await axios.post(`/upload/${this._task.path}`, { action: 'CompleteUpload' })
    return this._finishedResponse
  }

  /**
   * @param {number} seq chunk seq
   * @returns {Blob} chunk
   */
  _getChunk (seq) {
    return this._task.file.slice(seq * this._chunkSize, (seq + 1) * this._chunkSize)
  }

  _cleanup () {
    super._cleanup()
    if (!this.isStatus(STATUS_COMPLETED)) {
      if (this._url) {
        Axios.delete(this._url).catch(() => { })
      }
    }
  }
}

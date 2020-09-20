
import axios from '@/api/axios'
import ChunkUploadTask from '../chunk-task'
import { STATUS_COMPLETED } from '../task'

export default class LocalChunkUploadTask extends ChunkUploadTask {
  /**
   * @type {string}
   */
  _uploadId

  /**
   * @type {number}
   */
  _chunkSize

  /**
   * prepare upload
   * @returns {Promise.<number} chunks count
   */
  async _prepare () {
    const size = this._task.size
    const data = await this._request({
      method: 'POST',
      url: '/chunk',
      params: { size, chunk_size: 5 * 1024 * 1024 }
    }, axios)
    this._uploadId = data.id
    this._chunkSize = data.chunk_size
    return data.chunks
  }

  /**
   * @param {number} seq seq, start from 0
   * @param {Blob} blob  chunk
   * @param {Function} onProgress progress
   */
  async _chunkUpload (seq, blob, onProgress) {
    return this._request({
      method: 'PUT',
      url: `/chunk/${this._uploadId}/${seq}`,
      data: blob,
      headers: { 'Content-Type': 'application/octet-stream' },
      transformRequest: null,
      onUploadProgress: ({ loaded, total }) => {
        onProgress({ loaded, total })
      }
    }, axios)
  }

  /**
   * @returns {Promise.<any>} upload result
   */
  async _completeUpload () {
    return axios.post(`/chunk-content/${this._task.path}`, null, {
      params: { id: this._uploadId }
    })
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
      axios.delete(`/chunk/${this._uploadId}`)
    }
  }
}

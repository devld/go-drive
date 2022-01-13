import axios from '@/api/axios'
import ChunkUploadTask from '../chunk-task'
import { STATUS_COMPLETED } from '../task'

const PART_SIZE = 6 * 1024 * 1024 // 6M

const UNSIGNED_PAYLOAD = {
  'x-amz-content-sha256': 'UNSIGNED-PAYLOAD',
}

export default class S3UploadTask extends ChunkUploadTask {
  /**
   * @param {number} id task id
   * @param {TaskChangeListener} changeListener task changed listener
   * @param {TaskDef} task task definition
   * @param {any} [config] task specified config
   */
  constructor(id, changeListener, task, config) {
    super(id, changeListener, task, config)
    /**
     * @type {{url: string, multipart?: boolean}}
     */
    this._config = config
    this._partSize = PART_SIZE

    /**
     * @type {string} the multipart upload id
     */
    this._uploadId = undefined

    /**
     * ETag of uploaded parts
     * @type {Array.<string>}
     */
    this._uploadedParts = undefined
  }

  async _prepare() {
    if (!this._config.multipart) return 1 // PutObject directly

    // CreateMultipartUpload
    const r = await this._request({
      method: 'POST',
      url: this._config.url,
      headers: { ...UNSIGNED_PAYLOAD },
    })
    const matched = /<UploadId>(.+)<\/UploadId>/.exec(r.data)
    if (!matched) throw new Error('invalid response from aws s3')
    // multipart upload
    this._uploadId = matched[1]
    const parts = Math.ceil(this._task.size / this._partSize)

    this._uploadedParts = []
    for (let i = 0; i < parts; i++) {
      this._uploadedParts.push('_EMPTY_')
    }

    return parts
  }

  /**
   * @param {number} seq seq, start from 0
   * @param {Blob} blob  chunk
   * @param {Function} onProgress progress
   */
  async _chunkUpload(seq, blob, onProgress) {
    if (!this._uploadId) {
      // PutObject
      return this._request({
        method: 'PUT',
        url: this._config.url,
        data: blob,
        headers: {
          'Content-Type': 'application/octet-stream',
          ...UNSIGNED_PAYLOAD,
        },
        transformRequest: null,
        onUploadProgress: (e) =>
          onProgress({ loaded: e.loaded, total: e.total }),
      })
    }

    // request for presigned UploadPart url
    const r = await this._request(
      {
        method: 'POST',
        url: `/upload/${this._task.path}`,
        data: { action: 'UploadPart', uploadId: this._uploadId, seq: `${seq}` },
      },
      axios
    )
    const url = r.config.url

    const resp = await this._request({
      method: 'PUT',
      url,
      data: blob,
      headers: {
        'Content-Type': 'application/octet-stream',
        ...UNSIGNED_PAYLOAD,
      },
      transformRequest: null,
      onUploadProgress: (e) => onProgress({ loaded: e.loaded, total: e.total }),
    })

    const etag = resp.headers.etag
    this._uploadedParts[seq] = etag
  }

  /**
   * @returns {Promise.<any>} upload result
   */
  async _completeUpload() {
    if (!this._uploadId) {
      return axios.post(`/upload/${this._task.path}`, {
        action: 'CompletePutObject',
      })
    }
    return axios.post(`/upload/${this._task.path}`, {
      action: 'CompleteMultipartUpload',
      uploadId: this._uploadId,
      parts: this._uploadedParts.join(';'),
    })
  }

  /**
   * @param {number} seq chunk seq
   * @returns {Blob} chunk
   */
  _getChunk(seq) {
    return this._task.file.slice(
      seq * this._partSize,
      (seq + 1) * this._partSize
    )
  }

  _cleanup() {
    super._cleanup()
    if (!this.isStatus(STATUS_COMPLETED)) {
      axios
        .post(`/upload/${this._task.path}`, {
          action: 'AbortMultipartUpload',
          uploadId: this._uploadId,
        })
        .catch(() => {})
    }
    this._uploadedParts = undefined
  }
}

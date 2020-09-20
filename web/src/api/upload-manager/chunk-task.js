import { arrayRemove } from '@/utils'
/// large file task

import Axios from 'axios'
import { ApiError } from '../axios'
import UploadTask, { STATUS_COMPLETED, STATUS_CREATED, STATUS_ERROR, STATUS_PAUSED, STATUS_STOPPED, STATUS_UPLOADING } from './task'

/**
 * large file task
 */
export default class ChunkUploadTask extends UploadTask {
  /**
   * @type {number}
   */
  _maxConcurrent = 3

  /**
   * @type {Array.<import('axios').CancelTokenSource>}
   */
  _axiosSources = []

  /**
   * @type {Array.<number>}
   */
  _queue = []

  /**
   * @type {number}
   */
  _chunks

  _totalProgress = { loaded: 0, total: 0 }
  /**
   * @type {Object.<number, number>}
   */
  _uploadingChunkProgress = {}

  /**
 * @param {number} id task id
 * @param {TaskChangeListener} changeListener task changed listener
 * @param {TaskDef} task task definition
 * @param {any} [config] task specified config
 */
  constructor (id, changeListener, task, config) {
    super(id, changeListener, task, config)
    if (new.target === ChunkUploadTask) {
      throw new Error('Cannot construct abstract ChunkUploadTask')
    }
    this._totalProgress.total = this._task.size
  }

  start () {
    if (super.start() === false) return false
    if (this.isStatus(STATUS_CREATED) || this.isStatus(STATUS_STOPPED)) {
      this._start()
    } else {
      this._onChange(STATUS_UPLOADING, this._sumProgress())
      this._chunkUploadLoop()
    }
  }

  pause () {
    if (super.pause() === false) return false
    this._pause()
  }

  stop () {
    if (super.stop() === false) return false
    this._onChange(STATUS_STOPPED)
    this._abort()
    this._cleanup()
  }

  async _start () {
    this._onChange(STATUS_UPLOADING, this._sumProgress())
    try {
      this._chunks = await this._prepare()
    } catch (e) {
      this._abort(e)
      return
    }
    if (typeof (this._chunks) !== 'number' || this._chunks <= 0) throw new Error('invalid chunk size')
    this._queue.splice(0)
    for (let i = 0; i < this._chunks; i++) {
      this._queue.push(i)
    }
    this._chunkUploadLoop()
  }

  _pause () {
    this._axiosSources.forEach(t => { t.cancel() })
    this._onChange(STATUS_PAUSED)
  }

  _abort (e) {
    this._axiosSources.forEach(t => { t.cancel() })
    if (this.isStatus(STATUS_PAUSED) || this.isStatus(STATUS_STOPPED | STATUS_ERROR)) return
    this._onChange(STATUS_STOPPED)
    if (e) {
      this._onChange(STATUS_ERROR, ApiError.from(e))
    }
  }

  _sumProgress () {
    const total = this._totalProgress.total
    let loaded = this._totalProgress.loaded
    Object.values(this._uploadingChunkProgress).forEach(l => { loaded += l })
    return { loaded, total }
  }

  _chunkUploadLoop () {
    while (true) {
      if (!this.isStatus(STATUS_UPLOADING)) return
      const uploadingChunks = Object.keys(this._uploadingChunkProgress).length
      if (this._queue.length === 0) {
        if (uploadingChunks === 0) {
          this._doCompleteUpload()
        }
        return
      }
      if (uploadingChunks >= this._maxConcurrent) return
      const seq = this._queue.shift()
      this._doChunkUpload(seq)
        .then(
          () => { this._chunkUploadLoop() },
          e => {
            this._queue.push(seq)
            this._abort(e)
          }
        )
    }
  }

  async _doChunkUpload (seq) {
    const chunk = this._getChunk(seq)
    this._uploadingChunkProgress[seq] = 0
    try {
      await this._chunkUpload(seq, chunk, ({ loaded }) => {
        this._uploadingChunkProgress[seq] = loaded
        this._onChange(STATUS_UPLOADING, this._sumProgress())
      })
      this._totalProgress.loaded += this._uploadingChunkProgress[seq]
    } finally {
      delete this._uploadingChunkProgress[seq]
    }
  }

  async _doCompleteUpload () {
    let data
    try {
      data = await this._completeUpload()
    } catch (e) {
      this._abort(e)
      return
    }
    this._onChange(STATUS_COMPLETED, data)
    this._cleanup()
  }

  /**
   * make a request
   * @param {import('axios').AxiosRequestConfig} config
   * @param {import('axios').AxiosInstance} [axios]
   * @returns {Promise.<import('axios').AxiosResponse<any>>}
   */
  async _request (config, axios) {
    if (!axios) axios = Axios
    const cancelToken = Axios.CancelToken.source()
    this._axiosSources.push(cancelToken)
    try {
      return await axios({
        ...config,
        cancelToken: cancelToken.token
      })
    } finally {
      arrayRemove(this._axiosSources, e => e === cancelToken)
    }
  }

  /**
   * prepare upload
   * @returns {Promise.<number} chunks count
   */
  async _prepare () {
    throw new Error('not implemented')
  }

  /**
   * @param {number} seq seq, start from 0
   * @param {Blob} blob  chunk
   * @param {Function} onProgress progress
   */
  async _chunkUpload (seq, blob, onProgress) {
    throw new Error('not implemented')
  }

  /**
   * @returns {Promise.<any>} upload result
   */
  async _completeUpload () {
    throw new Error('not implemented')
  }

  /**
   * @param {number} seq chunk seq
   * @returns {Blob} chunk
   */
  _getChunk (seq) {
    throw new Error('not implemented')
  }

  _cleanup () {
    this._totalProgress.loaded = 0
    this._uploadingChunkProgress = {}
    this._chunks = undefined
    this._queue.splice(0)
  }
}

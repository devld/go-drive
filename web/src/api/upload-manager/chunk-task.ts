/// large file task
import { arrayRemove } from '@/utils'
import defaultHttp from '@/utils/http'
import { Http, HttpRequestConfig } from '@/utils/http/types'
import { ApiError, RequestTask } from '@/utils/http/utils'
import UploadTask, {
  STATUS_COMPLETED,
  STATUS_CREATED,
  STATUS_ERROR,
  STATUS_PAUSED,
  STATUS_STOPPED,
  STATUS_UPLOADING,
  TaskChangeListener,
  TaskDef,
  UploadProgress,
} from './task'

function insertSeq(arr: number[], seq: number) {
  let i = 0
  for (; i < arr.length; i++) {
    if ((i === 0 || arr[i - 1] < seq) && arr[i] > seq) break
  }
  arr.splice(i, 0, seq)
}

/**
 * large file task
 */
export default abstract class ChunkUploadTask extends UploadTask {
  protected _maxConcurrent = 3
  private _requests: RequestTask[] = []
  private _queue: number[] = []
  private _chunks: number | undefined
  private _totalProgress: UploadProgress = { loaded: 0, total: 0 }
  private _uploadingChunkProgress: Record<number, number> = {}
  private _prepareFailed: boolean | undefined

  constructor(
    id: number,
    changeListener: TaskChangeListener,
    task: TaskDef,
    config?: O
  ) {
    super(id, changeListener, task, config)
    this._totalProgress.total = this.task.size ?? 0
  }

  override async start() {
    if ((await super.start()) === false) return false
    if (
      this.isStatus(STATUS_CREATED) ||
      this.isStatus(STATUS_STOPPED) ||
      this._prepareFailed
    ) {
      this._prepareFailed = false
      this._start()
    } else {
      this._onChange(STATUS_UPLOADING, this._sumProgress())
      this._chunkUploadLoop()
    }
  }

  override async pause() {
    if ((await super.pause()) === false) return false
    this._pause()
  }

  override async stop() {
    if ((await super.stop()) === false) return false
    this._onChange(STATUS_STOPPED)
    this._abort()
    this._cleanup()
  }

  protected async _start() {
    this._onChange(STATUS_UPLOADING, this._sumProgress())
    try {
      this._chunks = await this._prepare()
    } catch (e: any) {
      this._prepareFailed = true
      this._abort(e)
      return
    }
    if (typeof this._chunks !== 'number' || this._chunks <= 0) {
      throw new Error('invalid chunk size')
    }
    this._queue.splice(0)
    for (let i = 0; i < this._chunks; i++) {
      this._queue.push(i)
    }
    this._chunkUploadLoop()
  }

  protected _pause() {
    this._requests.forEach((t) => {
      t.cancel()
    })
    this._onChange(STATUS_PAUSED)
  }

  protected _abort(e?: any) {
    this._requests.forEach((t) => {
      t.cancel()
    })
    if (
      this.isStatus(STATUS_PAUSED) ||
      this.isStatus(STATUS_STOPPED | STATUS_ERROR)
    ) {
      return
    }
    this._onChange(STATUS_STOPPED)
    if (e) {
      this._onChange(STATUS_ERROR, ApiError.from(e))
    }
  }

  private _sumProgress() {
    const total = this._totalProgress.total
    let loaded = this._totalProgress.loaded
    Object.values(this._uploadingChunkProgress).forEach((l) => {
      loaded += l
    })
    return { loaded, total }
  }

  private _chunkUploadLoop() {
    // eslint-disable-next-line no-constant-condition
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
      const seq = this._queue.shift()!
      this._doChunkUpload(seq).then(
        () => {
          this._chunkUploadLoop()
        },
        (e) => {
          insertSeq(this._queue, seq)
          this._abort(e)
        }
      )
    }
  }

  private async _doChunkUpload(seq: number) {
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

  private async _doCompleteUpload() {
    let data
    try {
      data = await this._completeUpload()
    } catch (e: any) {
      this._abort(e)
      return
    }
    this._onChange(STATUS_COMPLETED, data)
    this._cleanup()
  }

  protected async _request(config: HttpRequestConfig, http_?: Http) {
    if (!http_) http_ = defaultHttp
    const task = http_(config)
    this._requests.push(task)
    try {
      return await task
    } finally {
      arrayRemove(this._requests, (e) => e === task)
    }
  }

  /** prepare upload */
  protected abstract _prepare(): Promise<number>

  protected abstract _chunkUpload(
    seq: number,
    blob: Blob,
    onProgress: (p: UploadProgress) => void
  ): Promise<any>

  /** returns upload result */
  protected abstract _completeUpload(): Promise<any>

  protected abstract _getChunk(seq: number): Blob

  protected _cleanup() {
    this._totalProgress.loaded = 0
    this._uploadingChunkProgress = {}
    this._chunks = undefined
    this._queue.splice(0)
  }
}

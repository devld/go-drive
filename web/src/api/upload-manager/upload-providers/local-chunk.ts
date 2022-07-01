import { deleteTask } from '@/api'
import http from '@/api/http'
import { Task } from '@/types'
import { taskDone } from '@/utils'
import ChunkUploadTask from '../chunk-task'
import { STATUS_COMPLETED, STATUS_UPLOADING, UploadProgress } from '../task'

export default class LocalChunkUploadTask extends ChunkUploadTask {
  private _uploadId?: string
  private _chunkSize?: number
  private _mergeLock?: boolean
  private _mergeTask?: Task

  override async _prepare() {
    const size = this.task.size
    const data: any = await this._request(
      {
        method: 'post',
        url: '/chunk',
        params: { size, chunkSize: 5 * 1024 * 1024 },
      },
      http
    )
    this._uploadId = data.id
    this._chunkSize = data.chunkSize
    return data.chunks as number
  }

  override async _chunkUpload(
    seq: number,
    blob: Blob,
    onProgress: (p: UploadProgress) => void
  ) {
    return this._request(
      {
        method: 'put',
        url: `/chunk/${this._uploadId}/${seq}`,
        data: blob,
        headers: { 'Content-Type': 'application/octet-stream' },
        transformRequest: (d) => d,
        onUploadProgress: ({ loaded, total }) => {
          onProgress({ loaded, total })
        },
      },
      http
    )
  }

  override async _completeUpload() {
    // complete upload is not cancelable
    if (this._mergeLock) return
    this._mergeLock = true
    try {
      return await taskDone(
        http.post(`/chunk-content/${this.task.path}`, null, {
          params: { id: this._uploadId },
        }),
        (task) => {
          this._mergeTask = task
          this._onChange(STATUS_UPLOADING, task.progress)
        }
      )
    } finally {
      this._mergeTask = undefined
      this._mergeLock = undefined
    }
  }

  override _pause() {
    if (this._mergeLock) return
    super._pause()
  }

  override async stop() {
    if (this._mergeTask) {
      deleteTask(this._mergeTask.id).catch(() => {})
    }
    return super.stop()
  }

  override _getChunk(seq: number) {
    return this.task.file!.slice(
      seq * this._chunkSize!,
      (seq + 1) * this._chunkSize!
    )
  }

  override _cleanup() {
    super._cleanup()
    if (!this.isStatus(STATUS_COMPLETED)) {
      http.delete(`/chunk/${this._uploadId}`)
    }
  }
}

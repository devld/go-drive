import http from '@/api/http'
import defaultHttp from '@/utils/http'
import ChunkUploadTask from '../chunk-task'
import { STATUS_COMPLETED, UploadProgress } from '../task'

const CHUNK_SIZE = 4 * 1024 * 1024

export default class OneDriveUploadTask extends ChunkUploadTask {
  private _paused?: boolean
  private _url?: string
  private _chunkSize?: number
  private _finishedResponse?: any

  override _pause() {
    this._paused = true
    super._pause()
  }

  override async _prepare() {
    this._url = this._config!.url
    const size = this.task.size!
    this._chunkSize = CHUNK_SIZE
    this._maxConcurrent = 1
    return Math.ceil(size / CHUNK_SIZE)
  }

  override async _chunkUpload(
    seq: number,
    blob: Blob,
    onProgress: (p: UploadProgress) => void
  ) {
    let startByte = seq * this._chunkSize!
    const endByte = Math.min((seq + 1) * this._chunkSize!, this.task.size!) - 1

    if (this._paused) {
      // we need to recalculate nextExpectedRanges
      // because OneDrive's api does not allow us to upload chunks already received
      const resp = await this._request({ method: 'get', url: this._url })
      if (
        resp.data &&
        resp.data.nextExpectedRanges &&
        resp.data.nextExpectedRanges.length
      ) {
        const nextExpectedRanges =
          +resp.data.nextExpectedRanges[0].split('-')[0]
        if (
          !(
            nextExpectedRanges >= startByte && nextExpectedRanges <= endByte + 1
          )
        ) {
          throw new Error(
            `unexpected nextExpectedRanges: ${nextExpectedRanges}`
          )
        }
        startByte = nextExpectedRanges
        blob = this.task.file!.slice(startByte, endByte + 1)
      }
      this._paused = undefined
      if (blob.size === 0) return
    }

    const resp = await this._request({
      method: 'put',
      url: this._url,
      data: blob,
      headers: {
        'Content-Type': 'application/octet-stream',
        'Content-Range': `bytes ${startByte}-${endByte}/${this.task.size}`,
      },
      transformRequest: (d) => d,
      onUploadProgress: ({ loaded, total }) => {
        onProgress({ loaded, total })
      },
    })
    // 201: new file created
    // 200: existing file overridden
    if (resp.status === 201 || resp.status === 200) {
      this._finishedResponse = resp.data
    }
    return resp
  }

  override async _completeUpload() {
    if (!this._finishedResponse) {
      throw new Error('unexpected undefined finishedResponse')
    }
    await http.post(`/upload/${this.task.path}`, { action: 'CompleteUpload' })
    return this._finishedResponse
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
      if (this._url) {
        defaultHttp.delete(this._url).catch(() => {
          // ignore
        })
      }
    }
  }
}

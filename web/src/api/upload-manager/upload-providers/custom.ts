import http from '@/utils/http'
import http_ from '@/api/http'
import { API_PATH } from '@/api/http'
import ChunkUploadTask from '../chunk-task'
import { UploadProgress } from '../task'

interface CustomUploader {
  prepare?(): Promise<number>
  getChunk?(seq: number): Blob
  upload(
    data: Blob,
    seq: number,
    onProgress: (p: UploadProgress) => void
  ): Promise<any>
  complete?(): Promise<any>

  onCleanup?(): void
}

export default class CustomUploadTask extends ChunkUploadTask {
  private uploader?: CustomUploader
  private singleUploadResult: any

  protected async _prepare(): Promise<number> {
    const uploaderName = this._config?.uploader
    if (!uploaderName) throw new Error('invalid upload config')

    const resp = await http.get(API_PATH + `/drive-uploader/${uploaderName}.js`)
    let scriptContent: string = resp.data
    if (!scriptContent) throw new Error('invalid uploader code')
    scriptContent = scriptContent.replace(/;+$/, '')

    const scriptThis = Object.defineProperties(
      {},
      {
        config: { value: this._config },
        request: { value: this._request.bind(this) },
        maxConcurrent: {
          get: () => this._maxConcurrent,
          set: (v: number) => (this._maxConcurrent = v),
        },
        http: { value: http_ },
        task: { value: this.task },
        uploadCallback: { value: this.uploadCallback.bind(this) },
      }
    )

    this.uploader = eval(`(${scriptContent})`).call(undefined, scriptThis)

    if (this.uploader!.prepare) {
      return this.uploader!.prepare()
    }

    // upload directly
    return 1
  }

  protected async _chunkUpload(
    seq: number,
    blob: Blob,
    onProgress: (p: UploadProgress) => void
  ): Promise<any> {
    const res = await this.uploader!.upload(blob, seq, onProgress)
    if (this.uploader!.complete) {
      this.singleUploadResult = res
    }
    return res
  }

  protected _completeUpload(): Promise<any> {
    if (this.uploader!.complete) {
      return this.uploader!.complete()
    }
    return this.singleUploadResult
  }

  protected _getChunk(seq: number): Blob {
    if (this.uploader!.getChunk) {
      return this.uploader!.getChunk(seq)
    }
    return this.task.file!
  }

  protected _cleanup(): void {
    if (this.uploader!.onCleanup) {
      this.uploader!.onCleanup()
    }
    super._cleanup()
  }
}

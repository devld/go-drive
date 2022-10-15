import http from '@/api/http'
import ChunkUploadTask from '../chunk-task'
import { STATUS_COMPLETED, UploadProgress } from '../task'

const PART_SIZE = 6 * 1024 * 1024 // 6M

const UNSIGNED_PAYLOAD = {
  'x-amz-content-sha256': 'UNSIGNED-PAYLOAD',
}

export default class S3UploadTask extends ChunkUploadTask {
  private _partSize = PART_SIZE
  private _uploadId?: string
  private _uploadedParts?: string[]

  override async _prepare() {
    if (!this._config!.multipart) return 1 // PutObject directly

    // CreateMultipartUpload
    const r = await this._request({
      method: 'post',
      url: this._config!.url,
      headers: { ...UNSIGNED_PAYLOAD },
    })
    const matched = /<UploadId>(.+)<\/UploadId>/.exec(r.data)
    if (!matched) throw new Error('invalid response from aws s3')
    // multipart upload
    this._uploadId = matched[1]
    const parts = Math.ceil(this.task.size! / this._partSize)

    this._uploadedParts = []
    for (let i = 0; i < parts; i++) {
      this._uploadedParts.push('_EMPTY_')
    }

    return parts
  }

  override async _chunkUpload(
    seq: number,
    blob: Blob,
    onProgress: (p: UploadProgress) => void
  ) {
    if (!this._uploadId) {
      // PutObject
      return this._request({
        method: 'put',
        url: this._config!.url,
        data: blob,
        headers: {
          'Content-Type': 'application/octet-stream',
          ...UNSIGNED_PAYLOAD,
        },
        transformRequest: (d) => d,
        onUploadProgress: (e) =>
          onProgress({ loaded: e.loaded, total: e.total }),
      })
    }

    // request for presigned UploadPart url
    const r = await this._request(
      {
        method: 'post',
        url: `/upload/${this.task.path}`,
        data: { action: 'UploadPart', uploadId: this._uploadId, seq: `${seq}` },
      },
      http
    )
    const url = r.config.url

    const resp = await this._request({
      method: 'put',
      url,
      data: blob,
      headers: {
        'Content-Type': 'application/octet-stream',
        ...UNSIGNED_PAYLOAD,
      },
      transformRequest: (d) => d,
      onUploadProgress: (e) => onProgress({ loaded: e.loaded, total: e.total }),
    })

    const etag = resp.headers.etag
    this._uploadedParts![seq] = etag
  }

  override async _completeUpload() {
    if (!this._uploadId) {
      return http.post(`/upload/${this.task.path}`, {
        action: 'CompletePutObject',
      })
    }
    return http.post(`/upload/${this.task.path}`, {
      action: 'CompleteMultipartUpload',
      uploadId: this._uploadId,
      parts: this._uploadedParts!.join(';'),
    })
  }

  override _getChunk(seq: number) {
    return this.task.file!.slice(
      seq * this._partSize,
      (seq + 1) * this._partSize
    )
  }

  _cleanup() {
    super._cleanup()
    if (!this.isStatus(STATUS_COMPLETED)) {
      http
        .post(`/upload/${this.task!.path}`, {
          action: 'AbortMultipartUpload',
          uploadId: this._uploadId,
        })
        .catch(() => {
          // ignore
        })
    }
    this._uploadedParts = undefined
  }
}

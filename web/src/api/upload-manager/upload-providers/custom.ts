import http from '@/utils/http'
import http_ from '@/api/http'
import { API_PATH } from '@/api/http'
import { transformErrorResponse, transformTextResponse } from '@/utils/http/transformers'
import ChunkUploadTask from '../chunk-task'
import { UploadProgress } from '../task'

const UPLOADER_NAME_RE = /^[A-Za-z0-9_-]+$/
const MAX_CONCURRENT = 16

const scriptCache = new Map<string, Promise<string>>()

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

function loadUploaderScript(name: string, version?: string): Promise<string> {
  const key = version ? `${name}@${version}` : name
  let pending = scriptCache.get(key)
  if (!pending) {
    const query = version ? `?v=${encodeURIComponent(version)}` : ''
    pending = http
      .get(`${API_PATH}/drive-uploader/${name}${query}`, {
        transformResponse: [transformTextResponse([]), transformErrorResponse],
      })
      .then((resp) => {
        const scriptContent =
          typeof resp === 'string'
            ? resp
            : typeof resp?.data === 'string'
              ? resp.data
              : ''
        if (!scriptContent) throw new Error('invalid uploader code')
        return scriptContent.replace(/};+$/, '}')
      })
    scriptCache.set(key, pending)
    pending.catch(() => {
      scriptCache.delete(key)
    })
  }
  return pending
}

function createUploaderFactory(scriptContent: string): (ctx: unknown) => unknown {
  let factory: unknown
  try {
    factory = new Function(`return (${scriptContent})`)()
  } catch {
    throw new Error('invalid uploader code')
  }
  if (typeof factory !== 'function') {
    throw new Error('uploader factory is not a function')
  }
  return factory as (ctx: unknown) => unknown
}

function isCustomUploader(value: unknown): value is CustomUploader {
  return (
    !!value &&
    typeof value === 'object' &&
    typeof (value as CustomUploader).upload === 'function'
  )
}

export default class CustomUploadTask extends ChunkUploadTask {
  private uploader?: CustomUploader
  private singleUploadResult: any
  private cleanedUp = false

  protected async _prepare(): Promise<number> {
    this.cleanedUp = false
    const uploaderName = this._config?.uploader
    if (
      typeof uploaderName !== 'string' ||
      !UPLOADER_NAME_RE.test(uploaderName)
    ) {
      throw new Error('invalid upload config')
    }

    const scriptContent = await loadUploaderScript(
      uploaderName,
      this._config?.uploaderVersion
    )
    const factory = createUploaderFactory(scriptContent)

    const scriptThis = Object.defineProperties(
      {},
      {
        config: { value: this._config },
        request: { value: this._request.bind(this) },
        maxConcurrent: {
          get: () => this._maxConcurrent,
          set: (v: number) => {
            const n = Math.floor(Number(v))
            if (!Number.isFinite(n)) return
            this._maxConcurrent = Math.min(MAX_CONCURRENT, Math.max(1, n))
          },
        },
        http: { value: http_ },
        task: { value: this.task },
        uploadCallback: { value: this.uploadCallback.bind(this) },
      }
    )

    const uploader = factory.call(undefined, scriptThis)
    if (!isCustomUploader(uploader)) {
      throw new Error('uploader must provide an upload function')
    }
    this.uploader = uploader

    if (this.uploader.prepare) {
      return this.uploader.prepare()
    }

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
    if (!this.cleanedUp) {
      this.cleanedUp = true
      try {
        this.uploader?.onCleanup?.()
      } catch {
        // ignore cleanup errors so they do not mask the original failure
      }
    }
    super._cleanup()
  }
}

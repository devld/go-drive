import http from '@/utils/http'
import { API_PATH } from '@/api/http'
import { transformErrorResponse, transformTextResponse } from '@/utils/http/transformers'
import ChunkUploadTask from '../chunk-task'
import { UploadProgress } from '../task'

const UPLOADER_NAME_RE = /^[A-Za-z0-9_-]+$/
const MAX_CONCURRENT = 16

const scriptCache = new Map<string, Promise<string>>()

interface UploaderSpec {
  chunkSize?: number
  maxConcurrent?: number
  start?(ctx: UploaderRuntimeContext): Promise<unknown>
  upload(
    ctx: UploaderRuntimeContext,
    args: {
      blob: Blob
      seq: number
      session: unknown
      onProgress: (p: UploadProgress) => void
    }
  ): Promise<unknown>
  complete?(
    ctx: UploaderRuntimeContext,
    args: { session: unknown; parts: unknown[] }
  ): Promise<unknown>
  abort?(
    ctx: UploaderRuntimeContext,
    args: { session: unknown }
  ): void | Promise<void>
  getChunk?(ctx: UploaderRuntimeContext, seq: number): Blob
}

interface UploaderRuntimeContext {
  readonly config: Record<string, string>
  readonly request: ChunkUploadTask['_request']
  maxConcurrent: number
  readonly task: ChunkUploadTask['task']
  readonly uploadCallback: ChunkUploadTask['uploadCallback']
  chunks: number
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
        return scriptContent
      })
    scriptCache.set(key, pending)
    pending.catch(() => {
      scriptCache.delete(key)
    })
  }
  return pending
}

function loadUploaderSpec(scriptContent: string): UploaderSpec {
  let spec: unknown
  try {
    spec = new Function(
      `'use strict';
      var __uploaderSpec;
      function defineUploader(s) { __uploaderSpec = s; }
      ${scriptContent}
      return __uploaderSpec;`
    )()
  } catch {
    throw new Error('invalid uploader code')
  }
  if (
    !spec ||
    typeof spec !== 'object' ||
    typeof (spec as UploaderSpec).upload !== 'function'
  ) {
    throw new Error('uploader must call defineUploader with an upload function')
  }
  return spec as UploaderSpec
}

export default class CustomUploadTask extends ChunkUploadTask {
  private uploader?: {
    prepare(): Promise<number>
    getChunk(seq: number): Blob
    upload(
      data: Blob,
      seq: number,
      onProgress: (p: UploadProgress) => void
    ): Promise<unknown>
    complete(): Promise<unknown>
    onCleanup(): void
  }
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
    const spec = loadUploaderSpec(scriptContent)

    const ctx: UploaderRuntimeContext = Object.defineProperties(
      { chunks: 1 } as UploaderRuntimeContext,
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
        task: { value: this.task },
        uploadCallback: { value: this.uploadCallback.bind(this) },
        chunks: { writable: true, value: 1 },
      }
    )

    if (spec.maxConcurrent != null) {
      ctx.maxConcurrent = spec.maxConcurrent
    }

    const file = this.task.file
    const chunkSize = spec.chunkSize
    const chunks =
      !file || !chunkSize || chunkSize <= 0
        ? 1
        : Math.max(1, Math.ceil(file.size / chunkSize))
    ctx.chunks = chunks

    let session: unknown
    let completed = false
    const parts: unknown[] = []

    this.uploader = {
      prepare: async () => {
        if (spec.start) {
          session = await spec.start(ctx)
        }
        return chunks
      },
      getChunk: (seq) => {
        if (spec.getChunk) return spec.getChunk(ctx, seq)
        if (!file || !chunkSize || chunks === 1) return file!
        return file.slice(seq * chunkSize, (seq + 1) * chunkSize)
      },
      upload: async (blob, seq, onProgress) => {
        const res = await spec.upload(ctx, {
          blob,
          seq,
          session,
          onProgress,
        })
        parts[seq] = res
        return res
      },
      complete: async () => {
        let res: unknown
        if (spec.complete) {
          res = await spec.complete(ctx, { session, parts })
        }
        completed = true
        await this.uploadCallback({ action: 'Completed' })
        return res
      },
      onCleanup: () => {
        if (completed) return
        try {
          Promise.resolve(spec.abort?.(ctx, { session })).catch(() => undefined)
        } catch {
          // ignore cleanup errors so they do not mask the original failure
        }
      },
    }

    return this.uploader.prepare()
  }

  protected async _chunkUpload(
    seq: number,
    blob: Blob,
    onProgress: (p: UploadProgress) => void
  ): Promise<unknown> {
    return this.uploader!.upload(blob, seq, onProgress)
  }

  protected _completeUpload(): Promise<unknown> {
    return this.uploader!.complete()
  }

  protected _getChunk(seq: number): Blob {
    return this.uploader!.getChunk(seq)
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

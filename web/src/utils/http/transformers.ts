import { isPlainObject } from '..'
import {
  HttpRequestTransformer,
  HttpResponse,
  HttpResponseTransformer,
} from './types'
import { HttpError } from './utils'

const CONTENT_TYPE_JSON = 'application/json'

const matchContentTypes = (
  contentTypes?: string[],
  contentType?: string | null
) =>
  !contentTypes?.length ||
  !!(contentType && contentTypes.some((t) => contentType.includes(t)))

export const transformJSONRequest: HttpRequestTransformer = (config) => {
  if (config.headers?.['content-type']) return config

  if (typeof config.data === 'undefined') return config
  const type = typeof config.data
  if (
    isPlainObject(config.data) ||
    Array.isArray(config.data) ||
    type === 'number' ||
    type === 'boolean'
  ) {
    return {
      ...config,
      headers: { ...config.headers, 'content-type': CONTENT_TYPE_JSON },
      data: JSON.stringify(config.data),
    }
  }
  return config
}

export const transformBlobResponse: (
  contentTypes?: string[]
) => HttpResponseTransformer<HttpResponse<Blob>> =
  (contentTypes) =>
  async (error, resp): Promise<HttpResponse<Blob>> => {
    if (error) throw error
    if (resp.data instanceof Blob) return resp

    const contentType = resp.headers['content-type']
    if (!matchContentTypes(contentTypes, contentType)) return resp

    if (resp.data instanceof ReadableStream) {
      const reader = resp.data.getReader()
      const chunks: Uint8Array[] = []
      // eslint-disable-next-line no-constant-condition
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        chunks.push(value)
      }
      return { ...resp, data: new Blob(chunks) }
    }
    throw new Error('unable to transform to blob response')
  }

export const transformTextResponse: (
  contentTypes: string[]
) => HttpResponseTransformer<HttpResponse<string>> = (contentTypes) => {
  const blobTransformer = transformBlobResponse(contentTypes)

  return async (error, resp): Promise<HttpResponse> => {
    if (error) throw error
    if (typeof resp.data === 'string') return resp

    const blobResp = await blobTransformer(undefined, resp)
    if (!(blobResp.data instanceof Blob)) return blobResp
    return { ...blobResp, data: await blobResp.data?.text() }
  }
}

const jsonTransformTextResponse = transformTextResponse([CONTENT_TYPE_JSON])

export const transformJSONResponse: HttpResponseTransformer = async (
  error,
  resp
): Promise<HttpResponse> => {
  if (error) throw error
  const textResp = await jsonTransformTextResponse(undefined, resp)
  if (typeof textResp.data !== 'string') return textResp
  if (
    !matchContentTypes([CONTENT_TYPE_JSON], textResp.headers['content-type'])
  ) {
    return textResp
  }
  return { ...resp, data: JSON.parse(textResp.data) }
}

export const transformErrorResponse: HttpResponseTransformer = async (
  error,
  resp
): Promise<HttpResponse> => {
  if (error) throw error
  if (resp.status >= 200 && resp.status < 300) return resp
  let message: string
  if (typeof resp.data === 'string') message = resp.data
  else if (isPlainObject(resp.data) && typeof resp.data.message) {
    message = resp.data.message
  } else {
    message = `Request failed with status ${resp.status}`
  }
  throw new HttpError(resp.status, message, resp)
}

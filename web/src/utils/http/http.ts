import { buildURL } from '..'
import {
  Http,
  HttpBase,
  HttpExtend,
  HttpRequestBaseConfig,
  HttpRequestConfig,
  HttpRequestTransformer,
  HttpResponse,
  HttpResponseTransformer,
} from './types'
import {
  HttpError,
  RequestTaskImpl,
  normalizeHttpHeader,
  normalizeXHRHttpHeader,
} from './utils'

const processHeaders = (...headers: (O | undefined)[]) => {
  const merged: O = {}
  for (const header of headers) {
    if (!header) continue
    Object.keys(header).forEach((key) => {
      const value = header[key]
      key = key.toLowerCase()
      const mergedValue = merged[key]
      if (Array.isArray(value)) {
        merged[key] = value
        return
      }

      if (typeof mergedValue === 'undefined' || mergedValue === null) {
        merged[key] = value
      } else {
        if (!Array.isArray(mergedValue)) merged[key] = [mergedValue]
        merged[key].push(value)
      }
    })
  }
  return merged
}

const mergeArrayValues = <T>(...values: (T | T[] | undefined)[]) => {
  const merged: T[] = []
  for (const value of values) {
    if (!value) continue
    if (Array.isArray(value)) merged.push(...value)
    else merged.push(value)
  }
  return merged
}

const mergeURL = (baseURL?: string, url?: string) => {
  if (!url) return baseURL
  if (!baseURL) return url
  if (url.match(/^https?:\/\//i)) return url
  return (
    baseURL + (baseURL.endsWith('/') || url.startsWith('/') ? '' : '/') + url
  )
}

const mergeConfig = (
  baseConfig: HttpRequestBaseConfig,
  config: HttpRequestConfig
): HttpRequestConfig => {
  const baseURL = config.baseURL || baseConfig.baseURL
  return {
    baseURL,
    headers: processHeaders(baseConfig.headers, config.headers),
    timeout: config.timeout || baseConfig.timeout,
    transformRequest: mergeArrayValues(
      config.transformRequest,
      baseConfig.transformRequest
    ),
    transformResponse: mergeArrayValues(
      baseConfig.transformResponse,
      config.transformResponse
    ),
    url: buildURL(
      mergeURL(baseURL, config.url) || '',
      Object.assign({}, baseConfig.params, config.params)
    ),
    method: config.method || 'get',
    data: config.data,
    params: config.params,
    onUploadProgress: config.onUploadProgress,
    context: Object.assign({}, baseConfig.context, config.context),
  }
}

const wrapConfig = <T>(
  config: HttpRequestConfig | undefined,
  http: Http<T>
) => ({ ...config, context: { ...config?.context, __initiator: http } })

const transformRequest = async (
  config: HttpRequestConfig,
  transformers: HttpRequestTransformer[]
) => {
  for (const transformer of transformers) {
    config = await transformer(config)
  }
  return config
}

const transformResponse = async (
  response: HttpResponse,
  transformers: HttpResponseTransformer[],
  onStep: (data: any) => void
) => {
  let data: any = response
  let error: any
  for (const transformer of transformers) {
    try {
      data = await transformer(error, data)
      onStep(data)
      error = undefined
    } catch (e) {
      error = e
    }
  }
  if (error) throw error
  return data
}

type RequestParams = Pick<
  HttpRequestConfig,
  'data' | 'headers' | 'method' | 'url' | 'onUploadProgress'
> & { signal: AbortSignal; request: HttpRequestConfig }

const requestWithFetch = async (req: RequestParams): Promise<HttpResponse> => {
  const resp = await fetch(req.url || '', {
    method: req.method,
    body: req.data,
    headers: req.headers,
    signal: req.signal,
  })
  return {
    status: resp.status,
    headers: normalizeHttpHeader(resp.headers),
    data: resp.body,
    request: req.request,
  }
}

const requestWithXHR = (req: RequestParams) => {
  return new Promise<HttpResponse<Blob>>((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open(req.method || 'get', req.url || '')
    xhr.onreadystatechange = () => {
      if (xhr.readyState !== 4) return
      resolve({
        status: xhr.status,
        headers: normalizeXHRHttpHeader(xhr.getAllResponseHeaders()),
        data: xhr.response,
        request: req.request,
      })
    }
    xhr.onabort = () => {
      reject(new HttpError(0, 'aborted', undefined, true))
    }
    req.signal.onabort = () => xhr.abort()
    xhr.upload.onprogress = (e) => {
      req.onUploadProgress &&
        req.onUploadProgress.call(undefined, {
          loaded: e.loaded,
          total: e.total,
        })
    }

    const headers = req.headers
    if (headers) {
      Object.keys(headers).forEach((key) => {
        xhr.setRequestHeader(key, headers[key])
      })
    }
    xhr.responseType = 'blob'
    xhr.send(req.data)
  })
}

export const createHttp = <T = HttpResponse>(
  baseConfig: HttpRequestBaseConfig
): Http<T> => {
  const http: HttpBase<T> = function (config) {
    const mergedConfig = wrapConfig(
      config.context?.__initiator === fullHttp
        ? config
        : mergeConfig(baseConfig, config),
      fullHttp
    )

    const aborter = new AbortController()

    let rawResponse: HttpResponse
    let finalResponse: any
    let timedOut = false
    const timeoutTimer =
      mergedConfig.timeout && mergedConfig.timeout > 0
        ? setTimeout(() => {
            timedOut = true
            aborter.abort()
          }, mergedConfig.timeout)
        : null

    const promise: Promise<any> = transformRequest(
      mergedConfig,
      mergeArrayValues(mergedConfig.transformRequest)
    )
      .then<RequestParams>((config) => ({
        data: config.data,
        headers: config.headers,
        method: config.method,
        url: config.url,
        onUploadProgress: config.onUploadProgress,
        request: config,
        signal: aborter.signal,
      }))
      .then<HttpResponse>((config) =>
        config.onUploadProgress
          ? requestWithXHR(config)
          : requestWithFetch(config)
      )
      .then((resp) => {
        if (timeoutTimer) clearTimeout(timeoutTimer)

        rawResponse = resp
        finalResponse = resp
        return transformResponse(
          resp,
          mergeArrayValues(mergedConfig.transformResponse),
          (resp) => {
            finalResponse = resp
          }
        )
      })
      .catch((err) => {
        if (timeoutTimer) clearTimeout(timeoutTimer)

        if (err instanceof HttpError) return Promise.reject(err)

        if (!timedOut && err.name === 'AbortError') {
          return Promise.reject(
            new HttpError(
              rawResponse?.status || 0,
              err.message,
              finalResponse,
              true
            )
          )
        }
        return Promise.reject(
          new HttpError(
            rawResponse?.status || 0,
            err.message,
            finalResponse,
            false
          )
        )
      })

    return RequestTaskImpl.from(promise, aborter)
  }

  const extend: HttpExtend<T> = {
    head: (url, config) => http({ ...config, url, method: 'head' }),
    get: (url, config) => http({ ...config, url, method: 'get' }),
    post: (url, data, config) => http({ ...config, url, data, method: 'post' }),
    put: (url, data, config) => http({ ...config, url, data, method: 'put' }),
    delete: (url, config) => http({ ...config, url, method: 'delete' }),
  }

  const fullHttp = Object.assign(http, extend)
  return fullHttp
}

import { getLang } from '@/i18n'
import { waitPromise } from '@/utils'
import http, {
  Http,
  HttpError,
  HttpRequestBaseConfig,
  HttpRequestConfig,
  HttpResponse,
} from '@/utils/http'
import { createHttp } from '@/utils/http/http'
import {
  transformErrorResponse,
  transformJSONRequest,
  transformJSONResponse,
} from '@/utils/http/transformers'

export const AUTH_PARAM = 'token'

const AUTH_HEADER = 'Authorization'
const TOKEN_KEY = 'token'
const MAX_RETRY = 1

let apiPath = window.___config___.api
if (!/^https?:\/\//.test(apiPath)) {
  apiPath = location.origin + apiPath
}
export const API_PATH = apiPath

const BASE_CONFIG: HttpRequestBaseConfig = {
  baseURL: API_PATH,
  timeout: 60000,
}

function setToken(token: string) {
  return localStorage.setItem(TOKEN_KEY, token)
}

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

const doAuth = waitPromise(async () => {
  const data = (await http.post('/auth/init', null, BASE_CONFIG)).data
  const token = data.token
  setToken(token)
  return token
})

async function processConfig(config: HttpRequestConfig) {
  if (!config.context) config.context = {}
  if (config.context._t === undefined) config.context._t = -1
  config.context._t++
  if (config.context._t > MAX_RETRY)
    throw new HttpError(-1, 'max retry reached')

  if (!config.headers) config.headers = {}

  const token = getToken() ?? (await doAuth())
  config.headers[AUTH_HEADER] = token
  config.context._tokenUsing = token

  config.headers['Accept-Language'] = getLang()

  return config
}

async function handlerError(e: any) {
  const response = e.response as HttpResponse
  if (!response) throw e
  const status = response.status

  const config = response.request
  if (status === 401) {
    if (getToken() === config.context?._tokenUsing) {
      // if expired token was not replaced with a new one
      await doAuth()
    }
    const originalHttp = config.context?.__initiator as Http | undefined
    if (!originalHttp) throw e
    return originalHttp(config)
  }

  throw e
}

export default createHttp({
  ...BASE_CONFIG,
  transformRequest: [processConfig, transformJSONRequest],
  transformResponse: [
    transformJSONResponse,
    transformErrorResponse,
    (error, resp) => {
      if (error) return handlerError(error)
      return resp.data
    },
  ],
})

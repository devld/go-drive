import { getLang } from '@/i18n'
import { waitPromise } from '@/utils'
import { ApiError, wrapAxios } from '@/utils/http/utils'
import Axios, { AxiosRequestConfig } from 'axios'

const AUTH_HEADER = 'Authorization'
const TOKEN_KEY = 'token'
const MAX_RETRY = 1

let apiPath = window.___config___.api
if (!/^https?:\/\//.test(apiPath)) {
  apiPath = location.origin + apiPath
}
export const API_PATH = apiPath

const BASE_CONFIG: AxiosRequestConfig = {
  baseURL: API_PATH,
  timeout: 60000,
}

function setToken(token: string) {
  return localStorage.setItem(TOKEN_KEY, token)
}

function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

const doAuth = waitPromise(async () => {
  const data = (await Axios.post('/auth/init', null, BASE_CONFIG)).data
  const token = data.token
  setToken(token)
  return token
})

const axios = Axios.create(BASE_CONFIG)

async function processConfig(config: AxiosRequestConfig) {
  if (config._t === undefined) config._t = -1
  config._t++
  if (config._t > MAX_RETRY) throw new ApiError(-1, 'max retry reached')

  if (!config.headers) config.headers = {}

  const token = getToken() ?? (await doAuth())
  config.headers[AUTH_HEADER] = token
  config._tokenUsing = token

  config.headers['Accept-Language'] = getLang()

  return config
}

async function handlerError(e: any) {
  if (Axios.isCancel(e)) {
    throw new ApiError(-1, e.message || 'canceled', null, true)
  }

  if (!e.response) throw e
  const status = e.response.status

  const config = e.config
  if (status === 401) {
    if (getToken() === config._tokenUsing) {
      // if expired token was not replaced with a new one
      await doAuth()
    }
    return (config._axios || axios)(config)
  }

  throw ApiError.from(e)
}

axios.interceptors.request.use(processConfig)
axios.interceptors.response.use((resp) => resp.data, handlerError)

export default wrapAxios(axios)

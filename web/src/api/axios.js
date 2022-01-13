import { waitPromise } from '@/utils'
import Axios from 'axios'
import { getLang } from '@/i18n'

const AUTH_HEADER = 'Authorization'
const TOKEN_KEY = 'token'
const MAX_RETRY = 1

let apiPath = window.___config___.api
if (!/^https?:\/\//.test(apiPath)) {
  apiPath = location.origin + apiPath
}
export const API_PATH = apiPath

const BASE_CONFIG = {
  baseURL: API_PATH,
  timeout: 60000,
}

/**
 * @callback RequestTaskWrapCallback
 * @param {import('axios').CancelTokenSource}
 * @returns {any}
 */

export class RequestTask {
  /**
   * @param {any} v
   * @param {import('axios').CancelTokenSource} [source]
   */
  static from(v, source) {
    const t = new RequestTask(source)
    t._setPromise(v)
    return t
  }

  /**
   * @param {RequestTaskWrapCallback} fn
   */
  static wrap(fn) {
    const task = new RequestTask()
    task._setPromise(fn(task))
    return task
  }

  /**
   * @param {import('axios').CancelTokenSource} [axiosSource]
   */
  constructor(axiosSource) {
    /**
     * @type {Promise.<any>}
     */
    this._promise = undefined
    this._axiosSource = axiosSource || Axios.CancelToken.source()
    this.then = this.then.bind(this)
    this.catch = this.catch.bind(this)
    this.finally = this.finally.bind(this)
  }

  _setPromise(v) {
    if (this._promise) throw new Error('_setPromise already called')
    if (!v || typeof v.then !== 'function') {
      v = Promise.resolve(v)
    }
    this._promise = v
  }

  get promise() {
    return this._promise
  }

  get token() {
    return this._axiosSource.token
  }

  then(resolve, reject) {
    return RequestTask.from(
      this._promise.then(resolve, reject),
      this._axiosSource
    )
  }

  catch(handler) {
    return RequestTask.from(this._promise.catch(handler), this._axiosSource)
  }

  finally(handler) {
    return RequestTask.from(this._promise.finally(handler), this._axiosSource)
  }

  /**
   * @param {string} [message]
   */
  cancel(message) {
    this._axiosSource.cancel(message)
  }
}

/**
 * @param {RequestTask} task
 * @param {import('axios').AxiosRequestConfig} [config]
 */
function wrapConfig(task, config, axios) {
  if (!config) config = {}
  config.cancelToken = task.token
  config._axios = axios
  return config
}

/**
 * @param {import('axios').AxiosInstance} axios
 */
function wrapAxios(axios) {
  /**
   * @param {import('axios').AxiosRequestConfig} config
   */
  const axiosWrapper = function (config) {
    return RequestTask.wrap((t) => axios(wrapConfig(t, config, axiosWrapper)))
  }

  /**
   * @param {import('axios').AxiosRequestConfig} [config]
   */
  axiosWrapper.getUri = function (config) {
    return axios.getUri(config)
  }
  /**
   * @param {import('axios').AxiosRequestConfig} config
   */
  axiosWrapper.request = function (config) {
    return RequestTask.wrap((t) =>
      axios.request(wrapConfig(t, config, axiosWrapper))
    )
  }
  /**
   * @param {string} url
   * @param {import('axios').AxiosRequestConfig} [config]
   */
  axiosWrapper.head = function (url, config) {
    return RequestTask.wrap((t) =>
      axios.head(url, wrapConfig(t, config, axiosWrapper))
    )
  }
  /**
   * @param {string} url
   * @param {import('axios').AxiosRequestConfig} [config]
   */
  axiosWrapper.get = function (url, config) {
    return RequestTask.wrap((t) =>
      axios.get(url, wrapConfig(t, config, axiosWrapper))
    )
  }
  /**
   * @param {string} url
   * @param {any} [data]
   * @param {import('axios').AxiosRequestConfig} [config]
   */
  axiosWrapper.post = function (url, data, config) {
    return RequestTask.wrap((t) =>
      axios.post(url, data, wrapConfig(t, config, axiosWrapper))
    )
  }
  /**
   * @param {string} url
   * @param {import('axios').AxiosRequestConfig} [config]
   */
  axiosWrapper.delete = function (url, config) {
    return RequestTask.wrap((t) =>
      axios.delete(url, wrapConfig(t, config, axiosWrapper))
    )
  }
  /**
   * @param {string} url
   * @param {import('axios').AxiosRequestConfig} [config]
   */
  axiosWrapper.options = function (url, config) {
    return RequestTask.wrap((t) =>
      axios.options(url, wrapConfig(t, config, axiosWrapper))
    )
  }
  /**
   * @param {string} url
   * @param {any} [data]
   * @param {import('axios').AxiosRequestConfig} [config]
   */
  axiosWrapper.put = function (url, data, config) {
    return RequestTask.wrap((t) =>
      axios.put(url, data, wrapConfig(t, config, axiosWrapper))
    )
  }
  /**
   * @param {string} url
   * @param {any} [data]
   * @param {import('axios').AxiosRequestConfig} [config]
   */
  axiosWrapper.patch = function (url, data, config) {
    return RequestTask.wrap((t) =>
      axios.patch(url, data, wrapConfig(t, config, axiosWrapper))
    )
  }
  return axiosWrapper
}

export class ApiError extends Error {
  static from(e) {
    if (!e.response) return e
    const status = e.response.status
    const res = e.response.data
    return new ApiError(status, res.message, res.data)
  }

  constructor(status, message, data, isCancel) {
    super(message)
    this.status = status
    this.data = data
    this._isCancel = isCancel
  }

  get isCancel() {
    return this._isCancel
  }
}

function setToken(token) {
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

async function processConfig(config) {
  if (config._t === undefined) config._t = -1
  config._t++
  if (config._t > MAX_RETRY) throw new ApiError(-1, 'max retry reached')

  if (!config._noAuth) {
    let token = getToken()
    if (!token) token = await doAuth()
    config.headers[AUTH_HEADER] = token
    config._tokenUsing = token
  }

  config.headers['Accept-Language'] = getLang()

  return config
}

async function handlerError(e) {
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

export default axios
export const axiosWrapper = wrapAxios(axios)

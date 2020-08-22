import Axios from 'axios'

const AUTH_HEADER = 'Authorization'
const TOKEN_KEY = 'token'
const MAX_RETRY = 1
export const API_PATH = window.__api_path__ || process.env.VUE_APP_API

const BASE_CONFIG = {
  baseURL: API_PATH
}

class ApiError extends Error {
  constructor (status, message, data) {
    super(message)
    this.status = status
    this.data = data
  }
}

function setToken (token) {
  return localStorage.setItem(TOKEN_KEY, token)
}

function getToken () {
  return localStorage.getItem(TOKEN_KEY)
}

async function doAuth () {
  const data = await axios.post('/auth/init')
  const token = data.token
  setToken(token)
  return token
}

const axios = Axios.create(BASE_CONFIG)

async function processConfig (config) {
  if (config._t === undefined) config._t = -1
  config._t++
  if (config._t > MAX_RETRY) throw new ApiError(-1, 'max retry reached')

  let token = getToken()
  if (!token) token = await doAuth()
  config.headers[AUTH_HEADER] = token
  return config
}

async function handlerError (e) {
  if (!e.response) throw e

  const status = e.response.status
  const res = e.response.data

  if (status === 401) {
    await doAuth()
    return axios(e.config)
  }

  throw new ApiError(status, res.msg, res.data)
}

axios.interceptors.request.use(processConfig)
axios.interceptors.response.use(resp => resp.data, handlerError)

export default axios

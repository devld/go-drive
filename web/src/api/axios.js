import Axios from 'axios'

export const API_PATH = window.__api_path__ || process.env.VUE_APP_API

const axios = Axios.create({
  baseURL: API_PATH
})

class ApiError extends Error {
  constructor (status, message) {
    super(message)
    this.status = status
  }
}

axios.interceptors.response.use(resp => {
  return resp.data
}, e => {
  if (e.response) {
    return Promise.reject(new ApiError(e.response.status, e.response.data))
  }
  return Promise.reject(e)
})

export default axios

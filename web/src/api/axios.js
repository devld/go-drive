import Axios from 'axios'

const axios = Axios.create({
  baseURL: process.env.VUE_APP_API
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

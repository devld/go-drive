import Axios from 'axios'

const axios = Axios.create({
  baseURL: '/api'
})

axios.interceptors.response.use(resp => {
  return resp.data
})

export default axios


import axios from './axios'

export function listEntries (path) {
  return axios.get(`/entries${path}`)
}

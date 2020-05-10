
import axios from './axios'

export function listEntries (path) {
  return axios.get(`/entries${path}`)
}

export function getContent (path) {
  return axios.get(`/content${path}`)
}

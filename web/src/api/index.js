
import axios, { API_PATH } from './axios'

export function listEntries (path) {
  return axios.get(`/entries${path}`)
}

export function entry (path) {
  return axios.get(`/entry${path}`)
}

export function getContent (path) {
  return axios.get(`/content${path}`, {
    transformResponse: []
  })
}

export function fileUrl (path) {
  return `${API_PATH}/content${path}`
}

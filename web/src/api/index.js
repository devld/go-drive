
import axios, { API_PATH } from './axios'

export function listEntries (path) {
  return axios.get(`/entries/${path}`)
}

export function entry (path) {
  return axios.get(`/entry/${path}`)
}

export function getContent (path, noCache) {
  const params = {}
  if (noCache) {
    params.r = Math.random()
  }
  return axios.get(`/content/${path}`, {
    transformResponse: [],
    params
  })
}

export function makeDir (path) {
  return axios.post(`/mkdir/${path}`)
}

export function deleteEntry (path) {
  return axios.delete(`/entry/${path}`)
}

export function getUploadConfig (path, size, overwrite) {
  return axios.post(`/upload/${path}`, null, {
    params: { overwrite, size }
  })
}

export function fileUrl (path) {
  return `${API_PATH}/content/${path}`
}

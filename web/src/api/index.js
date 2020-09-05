
import axios, { API_PATH } from './axios'

export function listEntries (path) {
  return axios.get(`/entries/${path}`)
}

export function entry (path) {
  return axios.get(`/entry/${path}`)
}

export function getContent (path, accessKey, noCache) {
  const params = {}
  if (noCache) {
    params.r = Math.random()
  }
  return axios.get(_fileUrl(path, accessKey), {
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

function _fileUrl (path, accessKey) {
  return `/content/${path}${accessKey ? `?k=${encodeURIComponent(accessKey)}` : ''}`
}

export function fileUrl (path, accessKey) {
  return `${API_PATH}${_fileUrl(path, accessKey)}`
}

/// auth

export function login (username, password) {
  return axios.post('/auth/login', {
    username, password
  })
}

export function logout () {
  return axios.post('/auth/logout')
}

export function getUser () {
  return axios.get('/auth/user')
}

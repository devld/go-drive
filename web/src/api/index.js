import axios, { API_PATH, axiosWrapper } from './axios'

const ACCESS_KEY = '_k'

export function listEntries(path) {
  return axiosWrapper.get(`/entries/${path}`)
}

export function getEntry(path) {
  return axiosWrapper.get(`/entry/${path}`)
}

export function searchEntries(path, q, next) {
  return axiosWrapper.get(`/search/${path}`, {
    params: { q, next },
  })
}

export function getContent(path, accessKey, noCache) {
  const params = {}
  if (noCache) {
    params.r = Math.random()
  }
  return axiosWrapper.get(_fileUrl(path, accessKey), {
    transformResponse: [],
    params,
    _noAuth: true,
  })
}

export function makeDir(path) {
  return axios.post(`/mkdir/${path}`)
}

export function deleteEntry(path) {
  return axios.delete(`/entry/${path}`)
}

export function copyEntry(from, to, override) {
  return axios.post('/copy', null, {
    params: { from, to, override: override ? '1' : '' },
  })
}

export function moveEntry(from, to, override) {
  return axios.post('/move', null, {
    params: { from, to, override: override ? '1' : '' },
  })
}

export function getTasks(group) {
  return axiosWrapper.get('/tasks', {
    params: { group },
  })
}

export function getTask(id) {
  return axiosWrapper.get(`/task/${id}`)
}

export function deleteTask(id) {
  return axios.delete(`/task/${id}`)
}

function _fileUrl(path, accessKey) {
  return `/content/${path}${
    accessKey ? `?${ACCESS_KEY}=${encodeURIComponent(accessKey)}` : ''
  }`
}

export function fileUrl(path, accessKey) {
  return `${API_PATH}${_fileUrl(path, accessKey)}`
}

export function fileThumbnail(path, accessKey) {
  return `${API_PATH}/thumbnail/${path}${
    accessKey ? `?${ACCESS_KEY}=${encodeURIComponent(accessKey)}` : ''
  }`
}

/// auth

export function login(username, password) {
  return axios.post('/auth/login', {
    username,
    password,
  })
}

export function logout() {
  return axios.post('/auth/logout')
}

export function getUser() {
  return axiosWrapper.get('/auth/user')
}

export function getConfig() {
  return axiosWrapper.get('/config')
}

import axios from './axios'

export function getUsers() {
  return axios.get('/admin/users')
}

export function getUser(username) {
  return axios.get(`/admin/user/${username}`)
}

export function createUser(user) {
  return axios.post('/admin/user', user)
}

export function updateUser(username, user) {
  return axios.put(`/admin/user/${username}`, user)
}

export function deleteUser(username) {
  return axios.delete(`/admin/user/${username}`)
}

export function getGroups() {
  return axios.get('/admin/groups')
}

export function getGroup(name) {
  return axios.get(`/admin/group/${name}`)
}

export function createGroup(group) {
  return axios.post('/admin/group', group)
}

export function updateGroup(name, group) {
  return axios.put(`/admin/group/${name}`, group)
}

export function deleteGroup(name) {
  return axios.delete(`/admin/group/${name}`)
}

export function getDriveFactories() {
  return axios.get('/admin/drive-factories')
}

export function getDrives() {
  return axios.get('/admin/drives')
}

export function createDrive(drive) {
  return axios.post('/admin/drive', drive)
}

export function updateDrive(name, drive) {
  return axios.put(`/admin/drive/${name}`, drive)
}

export function deleteDrive(name) {
  return axios.delete(`/admin/drive/${name}`)
}

export function getDriveInitConfig(name) {
  return axios.get(`/admin/drive/${name}/init`)
}

export function initDrive(name, data) {
  return axios.post(`/admin/drive/${name}/init`, data)
}

export function reloadDrives() {
  return axios.post('/admin/drives/reload')
}

export function getPermissions(path) {
  return axios.get(`/admin/path-permissions/${path}`)
}

export function savePermissions(path, permissions) {
  return axios.put(`/admin/path-permissions/${path}`, permissions)
}

export function mountPaths(pathTo, mounts) {
  return axios.post(`/admin/mount/${pathTo}`, mounts)
}

export function cleanPermissionsAndMounts() {
  return axios.post('/admin/clean-permissions-mounts')
}

export function cleanDriveCache(name) {
  return axios.delete(`/admin/drive-cache/${name}`)
}

export function loadStats() {
  return axios.get('/admin/stats')
}

export function searchIndex(path) {
  return axios.post(`/admin/search/index/${path}`)
}

export function setOptions(options) {
  return axios.put('/admin/options', options)
}

export function getOption(key) {
  return axios.get(`/admin/options/${encodeURIComponent(key)}`)
}

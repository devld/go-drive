import axios from './axios'

export function getUsers () {
  return axios.get('/admin/users')
}

export function getUser (username) {
  return axios.get(`/admin/user/${username}`)
}

export function createUser (user) {
  return axios.post('/admin/user', user)
}

export function updateUser (username, user) {
  return axios.put(`/admin/user/${username}`, user)
}

export function deleteUser (username) {
  return axios.delete(`/admin/user/${username}`)
}

export function getGroups () {
  return axios.get('/admin/groups')
}

export function getGroup (name) {
  return axios.get(`/admin/group/${name}`)
}

export function createGroup (group) {
  return axios.post('/admin/group', group)
}

export function updateGroup (name, group) {
  return axios.put(`/admin/group/${name}`, group)
}

export function deleteGroup (name) {
  return axios.delete(`/admin/group/${name}`)
}

export function getDrives () {
  return axios.get('/admin/drives')
}

export function updateDrives (drives) {
  return axios.put('/admin/drives', drives)
}

export function reloadDrives () {
  return axios.post('/admin/drives/reload')
}

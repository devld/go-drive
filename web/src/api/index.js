import axios, { axiosWrapper } from './axios'

export * from './drive'

/// task

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


import axios from './axios'

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
  return `${process.env.VUE_APP_API}/content${path}`
}

import { ACCESS_KEY_API_KEY, DRIVE_API_CTX_SHARED_KEY } from '@/config'
import { buildURL } from '@/utils'
import axios, { API_PATH, axiosWrapper } from './axios'

function extraCtxData(ctx) {
  if (!ctx) return
  return { [DRIVE_API_CTX_SHARED_KEY]: ctx.sharedId }
}

export function listEntries(ctx, path) {
  return axiosWrapper.get(`/entries/${path}`, {
    params: extraCtxData(ctx),
  })
}

export function getEntry(ctx, path) {
  return axiosWrapper.get(`/entry/${path}`, {
    params: extraCtxData(ctx),
  })
}

export function searchEntries(ctx, path, q, next) {
  return axiosWrapper.get(`/search/${path}`, {
    params: { ...extraCtxData(ctx), q, next },
  })
}

export function getContent(ctx, path, accessKey, noCache) {
  const params = {}
  if (noCache) {
    params.r = Math.random()
  }
  return axiosWrapper.get(_fileUrl(ctx, path, accessKey), {
    transformResponse: [],
    params,
    _noAuth: true,
  })
}

export function makeDir(ctx, path) {
  return axios.post(`/mkdir/${path}`, null, {
    params: extraCtxData(ctx),
  })
}

export function deleteEntry(ctx, path) {
  return axios.delete(`/entry/${path}`, {
    params: extraCtxData(ctx),
  })
}

export function copyEntry(ctx, from, to, override) {
  return axios.post('/copy', null, {
    params: { ...extraCtxData(ctx), from, to, override: override ? '1' : '' },
  })
}

export function moveEntry(ctx, from, to, override) {
  return axios.post('/move', null, {
    params: { ...extraCtxData(ctx), from, to, override: override ? '1' : '' },
  })
}

function _fileUrl(ctx, path, accessKey) {
  return buildURL(`/content/${path}`, {
    ...extraCtxData(ctx),
    [ACCESS_KEY_API_KEY]: accessKey,
  })
}

export function fileUrl(ctx, path, accessKey) {
  return `${API_PATH}${_fileUrl(ctx, path, accessKey)}`
}

export function fileThumbnail(ctx, path, accessKey) {
  return buildURL(`${API_PATH}/thumbnail/${path}`, {
    ...extraCtxData(ctx),
    [ACCESS_KEY_API_KEY]: accessKey,
  })
}

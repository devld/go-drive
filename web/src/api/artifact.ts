import { EntryMeta, Task } from '@/types'
import type { HttpResponse } from '@/utils/http'
import { buildURL } from '@/utils'
import http, {
  ACCESS_KEY,
  API_PATH,
  pathPasswordHeaders,
  RETURN_RESPONSE_CONTEXT_KEY,
} from './http'

export interface ArchiveEntry {
  path: string
  name: string
  type: 'dir' | 'file'
  size: number
  modTime: number
  mimeType?: string
}

export interface ArtifactInfo {
  type: string
  name?: string
  mimeType?: string
  size: number
}

export type ArtifactPrepareResult =
  | { info: ArtifactInfo }
  | { task: Task<ArtifactInfo> }

function artifactQuery(
  path: string,
  meta: EntryMeta,
  type: string,
  args?: string
) {
  const query = { path, type } as O<any>
  if (args) query.args = args
  if (meta?.accessKey) query[ACCESS_KEY] = meta.accessKey
  return query
}

export function fileThumbnailUrl(path: string, meta: EntryMeta) {
  return buildURL(`${API_PATH}/artifact`, artifactQuery(path, meta, 'thumbnail'))!
}

export function artifactUrl(
  path: string,
  meta: EntryMeta,
  type: string,
  args?: string
) {
  return buildURL(`${API_PATH}/artifact`, artifactQuery(path, meta, type, args))!
}

export function prepareArtifact(
  path: string,
  meta: EntryMeta,
  type: string,
  args?: string
) {
  return http
    .post<HttpResponse<ArtifactInfo | Task<ArtifactInfo>>>('/artifact', null, {
      headers: pathPasswordHeaders(path),
      params: artifactQuery(path, meta, type, args),
      context: { [RETURN_RESPONSE_CONTEXT_KEY]: true },
    })
    .then((response) => {
      if (response.status === 202) {
        return { task: response.data as Task<ArtifactInfo> }
      }
      return { info: response.data as ArtifactInfo }
    })
}

export function getArchiveIndex(path: string, meta: EntryMeta) {
  return http.get<ArchiveEntry[]>('/artifact', {
    headers: pathPasswordHeaders(path),
    params: artifactQuery(path, meta, 'archive-index'),
  })
}

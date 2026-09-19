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
  name?: string
  mimeType?: string
  size: number
}

export type ArtifactPrepareResult =
  | { info: ArtifactInfo }
  | { task: Task<ArtifactInfo> }

export const ARTIFACT_THUMBNAIL = 'thumbnail'
export const ARTIFACT_ARCHIVE = 'archive'
export const ARCHIVE_ARGS_INDEX = 'index'

export function archiveContentArgs(path: string) {
  return `content:${path}`
}

function artifactQuery(path: string, meta: EntryMeta, args?: string) {
  const query = { path } as O<any>
  if (args) query.args = args
  if (meta?.accessKey) query[ACCESS_KEY] = meta.accessKey
  return query
}

function artifactPath(handler: string) {
  return `${API_PATH}/artifact/${handler}`
}

export function fileThumbnailUrl(path: string, meta: EntryMeta) {
  return buildURL(artifactPath(ARTIFACT_THUMBNAIL), artifactQuery(path, meta))!
}

export function artifactUrl(
  path: string,
  meta: EntryMeta,
  handler: string,
  args?: string
) {
  return buildURL(artifactPath(handler), artifactQuery(path, meta, args))!
}

export function prepareArtifact(
  path: string,
  meta: EntryMeta,
  handler: string,
  args?: string
) {
  return http
    .post<HttpResponse<ArtifactInfo | Task<ArtifactInfo>>>(
      `/artifact/${handler}`,
      null,
      {
        headers: pathPasswordHeaders(path),
        params: artifactQuery(path, meta, args),
        context: { [RETURN_RESPONSE_CONTEXT_KEY]: true },
      }
    )
    .then((response) => {
      if (response.status === 202) {
        return { task: response.data as Task<ArtifactInfo> }
      }
      return { info: response.data as ArtifactInfo }
    })
}

export function getArchiveIndex(path: string, meta: EntryMeta) {
  return http.get<ArchiveEntry[]>('/artifact/' + ARTIFACT_ARCHIVE, {
    headers: pathPasswordHeaders(path),
    params: artifactQuery(path, meta, ARCHIVE_ARGS_INDEX),
  })
}

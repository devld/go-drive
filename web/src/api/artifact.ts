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
  ref: string
}

export type ArtifactPrepareResult =
  | { info: ArtifactInfo }
  | { task: Task<ArtifactInfo> }

export const ARTIFACT_THUMBNAIL = 'thumbnail'
export const ARTIFACT_ARCHIVE = 'archive'
export const ARTIFACT_ZIP = 'zip'
export const ARCHIVE_ARGS_INDEX = 'index'

export function archiveContentArgs(path: string) {
  return `content:${path}`
}

export function archivePackArgs(members: string[]) {
  return `pack:${JSON.stringify(members)}`
}

export function zipSelectionArgs(dir: string, paths: string[]) {
  const root = dir.replace(/\/+$/, '')
  const prefix = root ? `${root}/` : ''
  const relative = paths
    .map((path) =>
      prefix && (path === root || path.startsWith(prefix))
        ? path.slice(prefix.length)
        : path
    )
    .filter((path) => path !== '')
  return JSON.stringify({ paths: relative })
}

function artifactQuery(
  path: string,
  meta: EntryMeta,
  extra?: { args?: string; ref?: string }
) {
  const query = { path } as O<any>
  if (extra?.args) query.args = extra.args
  if (extra?.ref) query.ref = extra.ref
  if (meta?.accessKey) query[ACCESS_KEY] = meta.accessKey
  return query
}

function artifactPath(handler: string) {
  return `${API_PATH}/artifact/${handler}`
}

export function artifactRefUrl(
  path: string,
  meta: EntryMeta,
  handler: string,
  ref: string
) {
  return buildURL(artifactPath(handler), artifactQuery(path, meta, { ref }))!
}

function preparedArtifact(
  response: HttpResponse<ArtifactInfo | Task<ArtifactInfo>>
): ArtifactPrepareResult {
  if (response.status === 202) {
    return { task: response.data as Task<ArtifactInfo> }
  }
  return { info: response.data as ArtifactInfo }
}

export function prepareArtifact(
  path: string,
  meta: EntryMeta,
  handler: string,
  args = ''
) {
  return http
    .post<HttpResponse<ArtifactInfo | Task<ArtifactInfo>>>(
      `/artifact/${handler}`,
      args,
      {
        headers: {
          ...pathPasswordHeaders(path),
          'content-type': 'text/plain',
        },
        params: artifactQuery(path, meta),
        context: { [RETURN_RESPONSE_CONTEXT_KEY]: true },
      }
    )
    .then(preparedArtifact)
}

export function getArchiveIndex(path: string, meta: EntryMeta, ref: string) {
  return http.get<ArchiveEntry[]>('/artifact/' + ARTIFACT_ARCHIVE, {
    headers: pathPasswordHeaders(path),
    params: artifactQuery(path, meta, { ref }),
  })
}

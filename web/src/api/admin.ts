import {
  Drive,
  DriveFactoryConfig,
  DriveInitConfig,
  DriveScriptContent,
  DriveScriptList,
  FileBucket,
  Group,
  Job,
  JobDefinitions,
  JobExecution,
  PathMeta,
  PathMountSource,
  PathPermission,
  ServiceStatsItem,
  Task,
  User,
} from '@/types'
import http, { StreamHttpResponse, streamHttp } from './http'

export function getUsers() {
  return http.get<User[]>('/admin/users')
}

export function getUser(username: string) {
  return http.get<User>(`/admin/users/${username}`)
}

export function createUser(user: Partial<User>) {
  return http.post<User>('/admin/users', user)
}

export function updateUser(username: string, user: Partial<User>) {
  return http.put<void>(`/admin/users/${username}`, user)
}

export function deleteUser(username: string) {
  return http.delete<void>(`/admin/users/${username}`)
}

export function getGroups() {
  return http.get<Group[]>('/admin/groups')
}

export function getGroup(name: string) {
  return http.get<Group>(`/admin/groups/${name}`)
}

export function createGroup(group: Partial<Group>) {
  return http.post<Group>('/admin/groups', group)
}

export function updateGroup(name: string, group: Partial<Group>) {
  return http.put<void>(`/admin/groups/${name}`, group)
}

export function deleteGroup(name: string) {
  return http.delete<void>(`/admin/groups/${name}`)
}

export function getDriveFactories() {
  return http.get<DriveFactoryConfig[]>('/admin/drive-factories')
}

export function getDrives() {
  return http.get<Drive[]>('/admin/drives')
}

export function createDrive(drive: Partial<Drive>) {
  return http.post<Drive>('/admin/drives', drive)
}

export function updateDrive(name: string, drive: Partial<Drive>) {
  return http.put<void>(`/admin/drives/${name}`, drive)
}

export function deleteDrive(name: string) {
  return http.delete<void>(`/admin/drives/${name}`)
}

export function getDriveInitConfig(name: string) {
  return http.post<DriveInitConfig>(`/admin/drives/${name}/init-config`)
}

export function initDrive(name: string, data: O<string>) {
  return http.post(`/admin/drives/${name}/init`, data)
}

export function reloadDrives() {
  return http.post<void>('/admin/drives/reload')
}

export function getPermissions(path: string) {
  return http.get<PathPermission[]>('/admin/path-permissions', {
    params: { path },
  })
}

export function savePermissions(path: string, permissions: O[]) {
  return http.put<void>('/admin/path-permissions', permissions, {
    params: { path },
  })
}

export function getAllPathMeta() {
  return http.get<PathMeta[]>('/admin/path-metadata')
}

export function savePathMeta(path: string, data: Partial<PathMeta>) {
  return http.put<void>('/admin/path-metadata', data, { params: { path } })
}

export function deletePathMeta(path: string) {
  return http.delete<void>('/admin/path-metadata', { params: { path } })
}

export function mountPaths(pathTo: string, mounts: PathMountSource[]) {
  return http.put<void>('/admin/path-mounts', mounts, {
    params: { path: pathTo },
  })
}

export function cleanPermissionsAndMounts() {
  return http.post<number>('/admin/maintenance/path-rules/cleanup')
}

export function cleanDriveCache(name: string) {
  return http.delete<void>(`/admin/drives/${name}/cache`)
}

export function loadStats() {
  return http.get<ServiceStatsItem[]>('/admin/stats')
}

export function searchIndex(path: string) {
  return http.put<Task<void>>('/admin/search-indexes', null, {
    params: { path },
  })
}

export function setOptions(options: O<string>) {
  return http.put<void>('/admin/options', options)
}

export function getOptions(...keys: string[]) {
  return http.get<O<string>>(
    `/admin/options/${encodeURIComponent(keys.join(','))}`
  )
}

export function getJobDefinitions() {
  return http.get<JobDefinitions>('/admin/job-definitions')
}

export function getJobs() {
  return http.get<Job[]>('/admin/jobs')
}

export function createJob(job: Partial<Job>) {
  return http.post<Job>('/admin/jobs', job)
}

export function updateJob(id: number, job: Partial<Job>) {
  return http.put<void>(`/admin/jobs/${id}`, job)
}

export function deleteJob(id: number) {
  return http.delete<void>(`/admin/jobs/${id}`)
}

export function getJobExecutions(jobId: number) {
  return http.get<JobExecution[]>('/admin/job-executions', {
    params: { jobId },
  })
}

export function executeJobSync(jobId: number) {
  return streamHttp.post<StreamHttpResponse<Task>>(
    '/admin/job-executions',
    null,
    { params: { jobId } }
  )
}

export function jobScriptEvalSync(code: string) {
  return streamHttp.post<StreamHttpResponse<Task>>(
    '/admin/job-script-evaluations',
    code,
    { headers: { 'content-type': 'text/plain' } }
  )
}

export function cancelJobExecution(id: number) {
  return http.post<void>(`/admin/job-executions/${id}/cancel`)
}

export function deleteJobExecution(id: number) {
  return http.delete<void>(`/admin/job-executions/${id}`)
}

export function deleteJobExecutions(jobId: number) {
  return http.delete<void>('/admin/job-executions', {
    params: { jobId },
  })
}

export function listDriveScripts() {
  return http.get<DriveScriptList>('/admin/drive-scripts')
}

export function syncDriveScriptsRepository() {
  return http.post<Task<void>>('/admin/drive-scripts/sync')
}

export function installDriveScript(name: string) {
  return http.put<void>(`/admin/drive-scripts/${encodeURIComponent(name)}`)
}

export function uninstallDriveScript(name: string) {
  return http.delete<void>(`/admin/drive-scripts/${encodeURIComponent(name)}`)
}

export function getDriveScriptContent(name: string) {
  return http.get<DriveScriptContent>(
    `/admin/drive-scripts/${encodeURIComponent(name)}/content`
  )
}

export function saveDriveScriptContent(
  name: string,
  content: Partial<DriveScriptContent>
) {
  return http.put(
    `/admin/drive-scripts/${encodeURIComponent(name)}/content`,
    content
  )
}

export function getAllFileBuckets() {
  return http.get<FileBucket[]>('/admin/file-buckets')
}

export function createFileBucket(bucket: Partial<FileBucket>) {
  return http.post<FileBucket>('/admin/file-buckets', bucket)
}

export function updateFileBucket(name: string, bucket: Partial<FileBucket>) {
  return http.put<void>(`/admin/file-buckets/${name}`, bucket)
}

export function deleteFileBucket(name: string) {
  return http.delete<void>(`/admin/file-buckets/${name}`)
}

import {
  AvailableDriveScript,
  Drive,
  DriveFactoryConfig,
  DriveInitConfig,
  DriveScriptContent,
  Group,
  InstalledDriveScript,
  Job,
  JobDefinition,
  JobExecution,
  PathMountSource,
  PathPermission,
  ServiceStatsItem,
  Task,
  User,
} from '@/types'
import http from './http'

export function getUsers() {
  return http.get<User[]>('/admin/users')
}

export function getUser(username: string) {
  return http.get<User>(`/admin/user/${username}`)
}

export function createUser(user: Partial<User>) {
  return http.post<User>('/admin/user', user)
}

export function updateUser(username: string, user: Partial<User>) {
  return http.put<void>(`/admin/user/${username}`, user)
}

export function deleteUser(username: string) {
  return http.delete<void>(`/admin/user/${username}`)
}

export function getGroups() {
  return http.get<Group[]>('/admin/groups')
}

export function getGroup(name: string) {
  return http.get<Group>(`/admin/group/${name}`)
}

export function createGroup(group: Partial<Group>) {
  return http.post<Group>('/admin/group', group)
}

export function updateGroup(name: string, group: Partial<Group>) {
  return http.put<void>(`/admin/group/${name}`, group)
}

export function deleteGroup(name: string) {
  return http.delete<void>(`/admin/group/${name}`)
}

export function getDriveFactories() {
  return http.get<DriveFactoryConfig[]>('/admin/drive-factories')
}

export function getDrives() {
  return http.get<Drive[]>('/admin/drives')
}

export function createDrive(drive: Partial<Drive>) {
  return http.post<Drive>('/admin/drive', drive)
}

export function updateDrive(name: string, drive: Partial<Drive>) {
  return http.put<void>(`/admin/drive/${name}`, drive)
}

export function deleteDrive(name: string) {
  return http.delete<void>(`/admin/drive/${name}`)
}

export function getDriveInitConfig(name: string) {
  return http.get<DriveInitConfig>(`/admin/drive/${name}/init`)
}

export function initDrive(name: string, data: O<string>) {
  return http.post(`/admin/drive/${name}/init`, data)
}

export function reloadDrives() {
  return http.post<void>('/admin/drives/reload')
}

export function getPermissions(path: string) {
  return http.get<PathPermission[]>(`/admin/path-permissions/${path}`)
}

export function savePermissions(path: string, permissions: O[]) {
  return http.put<void>(`/admin/path-permissions/${path}`, permissions)
}

export function mountPaths(pathTo: string, mounts: PathMountSource[]) {
  return http.post<void>(`/admin/mount/${pathTo}`, mounts)
}

export function cleanPermissionsAndMounts() {
  return http.post<number>('/admin/clean-permissions-mounts')
}

export function cleanDriveCache(name: string) {
  return http.delete<void>(`/admin/drive-cache/${name}`)
}

export function loadStats() {
  return http.get<ServiceStatsItem[]>('/admin/stats')
}

export function searchIndex(path: string) {
  return http.post<Task<void>>(`/admin/search/index/${path}`)
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
  return http.get<JobDefinition[]>('/admin/jobs/definitions')
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

export function getJobExecutions(jobId?: number) {
  return http.get<JobExecution[]>('/admin/jobs/execution', {
    params: { jobId },
  })
}

export function cancelJobExecution(id: number) {
  return http.put<void>(`/admin/jobs/execution/${id}/cancel`)
}

export function deleteJobExecution(id: number) {
  return http.delete<void>(`/admin/jobs/execution/${id}`)
}

export function deleteJobExecutions(jobId: number) {
  return http.delete<void>('/admin/jobs/execution', {
    params: { jobId },
  })
}

export function listAvailableDriveScripts(force?: boolean) {
  return http.get<AvailableDriveScript[]>('/admin/scripts/available', {
    params: { force },
  })
}

export function listInstalledDriveScripts() {
  return http.get<InstalledDriveScript[]>('/admin/scripts/installed')
}

export function installDriveScript(s: AvailableDriveScript) {
  return http.post<void>('/admin/scripts/install', s)
}

export function uninstallDriveScript(name: string) {
  return http.delete<void>('/admin/scripts/uninstall', {
    params: { name },
  })
}

export function getDriveScriptContent(name: string) {
  return http.get<DriveScriptContent>(`/admin/scripts/content/${name}`)
}

export function saveDriveScriptContent(
  name: string,
  content: Partial<DriveScriptContent>
) {
  return http.put(`/admin/scripts/content/${name}`, content)
}

import { FormItem } from '..'

export interface DriveFactoryConfig {
  type: string
  displayName: string
  readme: string
  configForm: FormItem[]
}

export interface Drive {
  name: string
  enabled: boolean
  type: string
  config: string
}

export interface DriveInitOAuth {
  url: string
  text: string

  principal?: string
}

export interface DriveInitConfig {
  configured: boolean

  oauth?: DriveInitOAuth

  form?: FormItem[]
  value?: O<string>
}

export enum PathPermissionPolicy {
  ACCEPTED = 1,
  REJECTED = 0,
}

export enum PathPermissionPerm {
  Empty = 0,
  Read = 1 << 0,
  Write = 1 << 1,
  ReadWrite = PathPermissionPerm.Read | PathPermissionPerm.Write,
}

export interface PathPermission {
  path: string
  subject: string
  permission: PathPermissionPerm
  policy: PathPermissionPolicy
}

export interface PathMountSource {
  path: string
  name: string
}

export interface ServiceStatsItem {
  name: string
  data: O<string>
}

export interface JobDefinition {
  name: string
  displayName: string
  description?: string
  paramsForm: FormItem[]
}

export interface Job {
  id: number
  description: string
  job: string
  params: string
  schedule: string
  enabled: boolean

  nextRun?: string
}

export enum JobExecutionStatus {
  Running = 'running',
  Success = 'success',
  Failed = 'failed',
}

export interface JobExecution {
  id: number
  jobId: number
  startedAt: number
  completedAt?: number
  status: JobExecutionStatus
  logs?: string
  errorMsg?: string
}

export interface AvailableDriveScript {
  name: string
  driveUrl: string
  driveUploaderUrl?: string
}

export interface InstalledDriveScript {
  name: string
  displayName: string
  description?: string
}

export interface DriveScriptContent {
  drive: string
  uploader?: string
}

import type { FormItem } from '..'

export interface SearchConfig {
  enabled: boolean
  examples: string[]
}

export interface ThumbnailConfig {
  extensions: O<boolean>
}

export interface VersionConfig {
  buildAt: string
  rev: string
  version: string
}

export interface AuthProvider {
  provider: string
  displayName: string
  type: 'form'
  form: FormItem[]
}

export interface AuthConfig {
  providers: AuthProvider[]
}

export interface WOPIConfig {
  enabled: boolean
  extensions: Record<string, { view: boolean; edit: boolean }>
}

export interface Config {
  auth: AuthConfig
  version: VersionConfig
  thumbnail: ThumbnailConfig
  options: O

  search?: SearchConfig
  wopi?: WOPIConfig
}

export interface ExternalFilePreviewer {
  exts: string[]
  name: string
  url: string
}

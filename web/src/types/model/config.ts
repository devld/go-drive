import type { FormItem } from '..'

export interface SearchConfig {
  enabled: boolean
  examples: string[]
}

export interface ArtifactHandlerConfig {
  extensions: string[]
  maxSize?: number
}

export type ArtifactConfig = O<ArtifactHandlerConfig>

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

export interface Config {
  auth: AuthConfig
  version: VersionConfig
  artifact: ArtifactConfig
  options: O

  search?: SearchConfig
}

export interface RawConfig extends Omit<Config, 'artifact'> {
  artifact: O<{
    /** Comma-separated file extensions returned by the backend. */
    extensions: string
    maxSize?: number
  }>
}

export interface ExternalFilePreviewer {
  exts: string[]
  name: string
  url: string
}

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

export interface Config {
  version: VersionConfig
  thumbnail: ThumbnailConfig
  options: O

  search?: SearchConfig
}

export interface ExternalFilePreviewer {
  exts: string[]
  name: string
  url: string
}

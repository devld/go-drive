import {
  DEFAULT_AUDIO_FILE_EXTS,
  DEFAULT_EXTERNAL_FILE_PREVIEWERS,
  DEFAULT_IMAGE_FILE_EXTS,
  DEFAULT_TEXT_FILE_EXTS,
  DEFAULT_VIDEO_FILE_EXTS,
} from '@/config'
import { ExternalFilePreviewer } from '@/types'
import { stringSplitN } from '@/utils'

export interface OptionsItem<T> {
  defaultValue?: string
  process: (v?: string) => T
}

const stringList = (v?: string) => {
  v = v?.trim()
  if (!v) return []
  return Object.freeze((v || '').split(',').map((e) => e.trim()))
}

const externalFileViewers = (v?: string): readonly ExternalFilePreviewer[] => {
  if (!v) return []
  return Object.freeze(
    v
      .split('\n')
      .map((e) => e.trim())
      .filter((e) => e && !e.startsWith('#'))
      .map((e) => stringSplitN(e, /\s+/, 3))
      .filter((e) => e.length === 3)
      .flatMap((parts) => {
        const exts = parts[0].toLowerCase().split(',')
        const viewer: ExternalFilePreviewer = {
          exts,
          name: parts[2],
          url: parts[1],
        }
        return viewer
      })
      .filter(Boolean)
  )
}

export const configOptions = {
  'web.textFileExts': {
    defaultValue: DEFAULT_TEXT_FILE_EXTS,
    process: stringList,
  },
  'web.imageFileExts': {
    defaultValue: DEFAULT_IMAGE_FILE_EXTS,
    process: stringList,
  },
  'web.audioFileExts': {
    defaultValue: DEFAULT_AUDIO_FILE_EXTS,
    process: stringList,
  },
  'web.videoFileExts': {
    defaultValue: DEFAULT_VIDEO_FILE_EXTS,
    process: stringList,
  },
  'web.monacoEditorExts': {
    process: stringList,
  },
  'web.externalFileViewers': {
    defaultValue: DEFAULT_EXTERNAL_FILE_PREVIEWERS,
    process: externalFileViewers,
  },
}

export type ConfigOptionsMap = {
  [K in keyof typeof configOptions]: ReturnType<
    typeof configOptions[K]['process']
  >
}

export const ConfigOptions: O<OptionsItem<any>> = configOptions

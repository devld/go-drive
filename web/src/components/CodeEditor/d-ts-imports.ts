import type { JavaScriptLibItem } from '../../../monaco-editor/src/types'

import { filename, mapOf } from '@/utils'
import serverGlobal from '../../../../docs/scripts/global.d.ts?raw'
const serverLibs = import.meta.glob('../../../../docs/scripts/libs/*.d.ts', {
  query: '?raw',
  import: 'default',
  eager: true,
})
const serverEnvs = import.meta.glob('../../../../docs/scripts/env/*.d.ts', {
  query: '?raw',
  import: 'default',
  eager: true,
})

const getName = (path: string) => filename(path).replace(/\.d\.ts$/, '')

export const D_SERVER_GLOBAL: JavaScriptLibItem = {
  name: 'global',
  filePath: 'docs/scripts/global.d.ts',
  content: serverGlobal,
}

export const D_SERVER_LIBS: JavaScriptLibItem[] = Object.entries(
  serverLibs as unknown as Record<string, string>
).map(([path, content]) => {
  const name = getName(path)
  return { name, filePath: `docs/scripts/libs/${name}.d.ts`, content }
})

export const D_SERVER_ENVS_MAP: Record<string, JavaScriptLibItem> = mapOf(
  Object.entries(serverEnvs as unknown as Record<string, string>).map(
    ([path, content]) => {
      const name = getName(path)
      return { name, filePath: `docs/scripts/env/${name}.d.ts`, content }
    }
  ),
  (e) => e.name
)

import driveUploaderEnv from '../../../../docs/drive-uploaders/types.d.ts?raw'

export const D_BROWSER_ENVS_MAP: Record<string, JavaScriptLibItem> = {
  uploader: {
    name: 'uploader',
    filePath: 'docs/drive-uploaders/types.d.ts',
    content: driveUploaderEnv,
  },
}

import type {
  JavaScriptLibItem,
  JavaScriptSetupOptions,
} from '../../../monaco-editor/src/types'
import {
  D_SERVER_ENVS_MAP,
  D_SERVER_GLOBAL,
  D_SERVER_LIBS,
  D_BROWSER_ENVS_MAP,
} from './d-ts-imports'

export const serverBaseOptions = (
  libs: JavaScriptLibItem[]
): JavaScriptSetupOptions => ({
  target: 'es5',
  lib: ['es5'],
  extraLibs: [...D_SERVER_LIBS, D_SERVER_GLOBAL, ...libs],
})

export const browserBaseOptions = (
  libs?: JavaScriptLibItem[]
): JavaScriptSetupOptions => ({
  target: 'latest',
  lib: ['es2020', 'dom'],
  extraLibs: libs,
})

export const getEnv = (name: string) => {
  if (name.startsWith('server-')) {
    name = name.substring(7)
    const env = D_SERVER_ENVS_MAP[name]
    if (!env) {
      console.warn('[CodeEditor] unknown env: ' + name)
      return
    }
    return serverBaseOptions([env])
  }
  const env = D_BROWSER_ENVS_MAP[name]
  if (!env) {
    console.warn('[CodeEditor] unknown env: ' + name)
    return
  }
  return browserBaseOptions([env])
}

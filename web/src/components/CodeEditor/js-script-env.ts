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

export const serverConsoleLib: JavaScriptLibItem = {
  name: 'console',
  content: `interface Console {
    debug(...message?: any[]): void;
    error(...message?: any[]): void;
    info(...message?: any[]): void;
    log(...message?: any[]): void;
    warn(...message?: any[]): void;
  }
  declare const console: Console;`,
}

export const serverBaseOptions = (
  libs: JavaScriptLibItem[]
): JavaScriptSetupOptions => ({
  target: 'es5',
  lib: ['es5'],
  extraLibs: [serverConsoleLib, ...D_SERVER_LIBS, D_SERVER_GLOBAL, ...libs],
})

export const browserBaseOptions = (
  libs?: JavaScriptLibItem[]
): JavaScriptSetupOptions => ({ extraLibs: libs })

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

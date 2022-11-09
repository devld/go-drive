import type {
  JavaScriptLibItem,
  JavaScriptSetupOptions,
} from '../../../monaco-editor/src/types'

export const consoleLib: JavaScriptLibItem = {
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

const commonLibs: Readonly<JavaScriptLibItem[]> = Object.freeze([
  consoleLib,
  ...Object.keys(JS_DECLARATIONS.libs).map((name) => ({
    name,
    content: JS_DECLARATIONS.libs[name],
  })),
  { name: 'global', content: JS_DECLARATIONS.global },
])

export const baseOptions = (
  libs: JavaScriptLibItem[]
): JavaScriptSetupOptions => ({
  target: 'es5',
  lib: ['es5'],
  extraLibs: [...commonLibs, ...libs],
})

export const getEnv = (name?: string) => {
  const content = name ? JS_DECLARATIONS.env[name] : undefined
  if (typeof content !== 'string') {
    if (name) {
      console.warn('[CodeEditor] unknown env: ' + name)
    }
    return
  }
  return baseOptions([{ name: name!, content: content! }])
}

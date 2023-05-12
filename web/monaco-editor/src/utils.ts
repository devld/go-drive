import * as monaco from 'monaco-editor'
import {
  EditorOutMessageTypes,
  JavaScriptSetupOptions,
  MessageHandler,
  MESSAGE_KEY_PREFIX,
} from './types'

// https://github.com/go-gitea/gitea/pull/21734/commits/cea7458c79f74805f384ea721c2fd2a7517284a0
monaco.languages.register({ id: 'vs.editor.nullLanguage' })
monaco.languages.setLanguageConfiguration('vs.editor.nullLanguage', {})

export const queries = parseQueries()

const messageKey = MESSAGE_KEY_PREFIX + queries['id']

export function createEditor(language: string) {
  const container = document.querySelector(
    '.editor-container'
  ) as HTMLDivElement

  const editor = monaco.editor.create(container, {
    language,
  })

  window.addEventListener('resize', () => {
    editor.layout()
  })

  return editor
}

const JsTargets: Record<string, monaco.languages.typescript.ScriptTarget> = {
  es5: monaco.languages.typescript.ScriptTarget.ES5,
  es6: monaco.languages.typescript.ScriptTarget.ES2015,
  latest: monaco.languages.typescript.ScriptTarget.Latest,
}

export function setupJavaScript(opt: JavaScriptSetupOptions) {
  monaco.languages.typescript.javascriptDefaults.setDiagnosticsOptions({
    noSemanticValidation: true,
    noSyntaxValidation: false,
  })
  monaco.languages.typescript.javascriptDefaults.setCompilerOptions({
    allowNonTsExtensions: true,
    allowJs: true,
    lib: opt.lib,
    target:
      JsTargets[opt.target ?? 'latest'] ||
      monaco.languages.typescript.ScriptTarget.Latest,
  })
  if (opt.extraLibs) {
    monaco.languages.typescript.javascriptDefaults.setExtraLibs(
      opt.extraLibs.map((item) => ({
        content: item.content,
        filePath: `${item.name}.d.ts`,
      }))
    )
    opt.extraLibs.forEach((item) => {
      monaco.editor.createModel(
        item.content,
        'typescript',
        monaco.Uri.parse(`${item.name}.d.ts`)
      )
    })
  }
}

export function setupDataExchanging(handlers: Record<string, MessageHandler>) {
  window.addEventListener('message', (e) => {
    let data = e.data
    if (
      data &&
      typeof data === 'object' &&
      (data = data[messageKey]) &&
      Array.isArray(data) &&
      typeof data[0] === 'string'
    ) {
      const fn = handlers[data[0]]
      if (fn) {
        fn(data[1])
      }
    }
  })
}

export function emit<K extends keyof EditorOutMessageTypes>(
  fn: K,
  data: EditorOutMessageTypes[K]
) {
  window.parent.postMessage({
    [messageKey]: [fn, data],
  })
}

function parseQueries(): Record<string, string> {
  let i = location.href.indexOf('?')
  if (i === -1) return {}
  let qs = location.href.substring(i + 1)
  i = qs.indexOf('#')
  if (i >= 0) qs = qs.substring(0, i)
  return qs
    .split('&')
    .map((item) => {
      const t = item.split('=', 2)
      return [decodeURIComponent(t[0]), decodeURIComponent(t[1])]
    })
    .reduce((a, c) => {
      const [k, v] = c
      a[k] = v
      return a
    }, {} as Record<string, string>)
}

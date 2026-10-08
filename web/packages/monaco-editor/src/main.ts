import './main.scss'
import { EditorInMessageHandlers } from './types'
import {
  createEditor,
  emit,
  queries,
  setupDataExchanging,
  setupJavaScript,
} from './utils'
import './workers'
import * as monaco from 'monaco-editor'

const language = queries['lang']

const editor = createEditor(language)
emit('ready', undefined)

let editorTheme: 'vs' | 'vs-dark' = 'vs'
let editorBackground = '#ffffff'
const updateTheme = () => {
  const themeName = `go-drive-${editorTheme}`
  monaco.editor.defineTheme(themeName, {
    base: editorTheme,
    inherit: true,
    rules: [],
    colors: {
      'editor.background': editorBackground,
      'editorGutter.background': editorBackground,
    },
  })
  editor.updateOptions({ theme: themeName })
}

editor.getModel()!.onDidChangeContent(() => {
  emit('change', editor.getValue())
})
editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
  emit('save', undefined)
})

const messageHandlers: EditorInMessageHandlers = {
  setValue: (data) => {
    editor.setValue(data)
  },
  setupJs: (data) => {
    setupJavaScript(data)
  },
  setDisabled: (disabled) => {
    editor.updateOptions({ readOnly: disabled })
  },
  setTheme: (theme) => {
    if (theme !== 'vs' && theme !== 'vs-dark') return
    editorTheme = theme
    updateTheme()
  },
  setBackground: (background) => {
    if (!background.startsWith('#')) return
    editorBackground = background
    updateTheme()
  },
}

setupDataExchanging(messageHandlers)

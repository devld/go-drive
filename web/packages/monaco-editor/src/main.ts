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
  setTheme: ({ base, background }) => {
    if (base !== 'vs' && base !== 'vs-dark') return
    if (!background.startsWith('#')) return
    const themeName = `go-drive-${base}`
    monaco.editor.defineTheme(themeName, {
      base,
      inherit: true,
      rules: [],
      colors: {
        'editor.background': background,
        'editorGutter.background': background,
      },
    })
    editor.updateOptions({ theme: themeName })
  },
}

setupDataExchanging(messageHandlers)

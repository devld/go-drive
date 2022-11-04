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

const language = queries['lang']

const editor = createEditor(language)
emit('ready', undefined)

editor.getModel()!.onDidChangeContent(() => {
  emit('change', editor.getValue())
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
    editor.updateOptions({ theme })
  },
}

setupDataExchanging(messageHandlers)

export const MESSAGE_KEY_PREFIX = 'monaco-editor-data-'

export type MessageHandler<T = any> = (data: T) => void

export interface JavaScriptLibItem {
  name: string
  content: string
}
export interface JavaScriptSetupOptions {
  target: string
  lib: string[]
  extraLibs: JavaScriptLibItem[]
}

export interface EditorInMessageTypes {
  setValue: string
  setupJs: JavaScriptSetupOptions
  setDisabled: boolean
  setTheme: string
}

export interface EditorOutMessageTypes {
  ready: undefined
  change: string
  save: undefined
}

export type EditorInMessageHandlers = {
  [K in keyof EditorInMessageTypes]: MessageHandler<EditorInMessageTypes[K]>
}

export type EditorOutMessageHandlers = {
  [K in keyof EditorOutMessageTypes]: MessageHandler<EditorOutMessageTypes[K]>
}

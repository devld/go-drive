import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker'
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import typescriptWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker'

self.MonacoEnvironment = {
  getWorker: (_id: string, label: string) => {
    switch (label) {
      case 'css':
      case 'scss':
      case 'less':
        return new cssWorker()
      case 'html':
        return new htmlWorker()
      case 'json':
        return new jsonWorker()
      case 'typescript':
      case 'javascript':
        return new typescriptWorker()
      case 'editorWorkerService':
        return new editorWorker()
    }
    throw new Error('unsupported: ' + label)
  },
}

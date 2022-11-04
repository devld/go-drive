import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'

self.MonacoEnvironment = {
  getWorker: (_id: string, label: string) => {
    switch (label) {
      case 'css':
      case 'scss':
      case 'less':
        return import(
          'monaco-editor/esm/vs/language/css/css.worker?worker'
        ).then((g) => new g.default())
      case 'html':
        return import(
          'monaco-editor/esm/vs/language/html/html.worker?worker'
        ).then((g) => new g.default())
      case 'json':
        return import(
          'monaco-editor/esm/vs/language/json/json.worker?worker'
        ).then((g) => new g.default())
      case 'typescript':
      case 'javascript':
        return import(
          'monaco-editor/esm/vs/language/typescript/ts.worker?worker'
        ).then((g) => new g.default())
      case 'editorWorkerService':
        return new editorWorker()
    }
    throw new Error('unsupported: ' + label)
  },
}

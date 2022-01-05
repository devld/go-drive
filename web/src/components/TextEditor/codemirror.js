import CodeMirror from 'codemirror'
import 'codemirror/mode/meta'

window.CodeMirror = CodeMirror

export async function loadMode(mode) {
  const name = mode.mode
  await import(
    /* @vite-ignore */ `../../../node_modules/codemirror/mode/${name}/${name}.js`
  )
}

export default CodeMirror

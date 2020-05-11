<template>
  <div ref="editor" class="text-editor" />
</template>
<script>
import CodeMirror from './codemirror'
import { filenameExt } from '../../utils'

const THEME_NAME = 'github-light'

export default {
  name: 'TextEditor',
  props: {
    value: {
      type: String
    },
    filename: {
      type: String
    },
    lineNumbers: {
      type: Boolean
    }
  },
  watch: {
    filename: {
      immediate: true,
      handler () {
        if (this.filename) {
          this.setEditorModeByFilename(this.filename)
        }
      }
    },
    value: {
      immediate: true,
      handler () {
        this.setEditorContent(this.value)
      }
    },
    lineNumbers (val) {
      this.setEditorOption('lineNumbers', val)
    }
  },
  mounted () {
    this.initEditor()
  },
  methods: {
    initEditor () {
      this.editor = CodeMirror(this.$refs.editor, {
        theme: THEME_NAME, value: this.content || '',
        lineNumbers: this.lineNumbers
      })
      this.editor.on('change', () => {
        this.content = this.editor.getValue()
        this.$emit('input', this.content)
      })
    },
    async setEditorModeByFilename (filename) {
      const ext = filenameExt(filename)
      const mode = CodeMirror.findModeByExtension(ext)
      if (mode) {
        this.editor.setOption('mode', mode.mode)
        CodeMirror.autoLoadMode(this.editor, mode.mode)
      } else {
        console.warn(`[CodeMirror] language mode of '${ext}' not found`)
      }
    },
    setEditorContent (content) {
      if (this.content === content) return
      this.content = content
      if (this.editor) {
        this.editor.setValue(this.content)
      }
    },
    setEditorOption (name, value) {
      this.editor.setOption(name, value)
    }
  }
}
</script>
<style lang="scss">
@import url("~codemirror/lib/codemirror.css");
@import url("~codemirror-github-light/lib/codemirror-github-light-theme.css");

.text-editor {
  .CodeMirror {
    height: unset;
  }

  .CodeMirror-scroll {
    min-height: 300px;
  }
}
</style>

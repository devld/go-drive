<template>
  <div ref="editor" class="text-editor" />
</template>
<script>
import CodeMirror from './codemirror'
import { filenameExt } from '@/utils'
import {
  addPreferColorListener,
  isDarkMode,
  removePreferColorListener,
} from '@/utils/theme'

function getThemeName() {
  return isDarkMode() ? 'material-darker' : 'github-light'
}

export default {
  name: 'TextEditor',
  props: {
    value: {
      type: String,
    },
    filename: {
      type: String,
    },
    lineNumbers: {
      type: Boolean,
    },
    disabled: {
      type: Boolean,
    },
  },
  watch: {
    filename() {
      if (this.filename) {
        this.setEditorMode()
      }
    },
    value: {
      immediate: true,
      handler() {
        this.setEditorContent(this.value)
      },
    },
    lineNumbers(val) {
      this.setEditorOption('lineNumbers', val)
    },
    disabled(val) {
      this.setEditorOption('readOnly', val ? 'nocursor' : false)
    },
  },
  created() {
    addPreferColorListener(this.prefersColorChanged)
  },
  beforeDestroy() {
    removePreferColorListener(this.prefersColorChanged)
  },
  mounted() {
    this.initEditor()
  },
  methods: {
    initEditor() {
      this.editor = CodeMirror(this.$refs.editor, {
        theme: getThemeName(),
        value: this.content || '',
        lineNumbers: this.lineNumbers,
        readOnly: this.disabled ? 'nocursor' : false,
      })
      this.setEditorMode()
      this.editor.on('change', () => {
        this.content = this.editor.getValue()
        this.$emit('input', this.content)
      })
    },
    async setEditorMode() {
      const ext = filenameExt(this.filename)
      const mode = CodeMirror.findModeByExtension(ext)
      if (mode) {
        this.setEditorOption('mode', mode.mode)
        CodeMirror.autoLoadMode(this.editor, mode.mode)
      } else {
        console.warn(`[CodeMirror] language mode of '${ext}' not found`)
      }
    },
    setEditorContent(content) {
      if (this.content === content) return
      this.content = content
      if (this.editor) {
        this.editor.setValue(this.content)
      }
    },
    setEditorOption(name, value) {
      if (this.editor) {
        this.editor.setOption(name, value)
      }
    },
    prefersColorChanged() {
      this.setEditorOption('theme', getThemeName())
    },
  },
}
</script>
<style lang="scss">
@import url('~codemirror/lib/codemirror.css');

@import url('~codemirror-github-light/lib/codemirror-github-light-theme.css');
@import '~codemirror/theme/material-darker.css';

.text-editor {
  .CodeMirror {
    height: unset;
  }

  .CodeMirror-scroll {
    min-height: 300px;
  }
}
</style>

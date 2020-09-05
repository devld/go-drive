<template>
  <div class="text-edit-view" @keydown="onKeyDown">
    <h1 class="filename">
      <button
        class="header-button simple-button save-button"
        v-if="!readonly"
        @click="saveFile"
      >{{ saving ? 'Saving...' : 'Save' }}</button>
      <span :title="filename">{{ filename }}</span>
      <button class="header-button close-button plain-button" title="Close" @click="$emit('close')">
        <i-icon svg="#icon-close" />
      </button>
    </h1>
    <text-editor
      v-if="!error"
      v-model="content"
      :filename="filename"
      line-numbers
      :disabled="readonly"
    />
    <error-view v-else :status="error.status" :message="error.message" />
    <div v-if="!inited" class="loading">Loading...</div>
  </div>
</template>
<script>
import { filename } from '@/utils'
import { entry, getContent } from '@/api'
import TextEditor from '@/components/TextEditor'
import uploadManager from '@/api/upload-manager'

export default {
  name: 'TextEditView',
  components: { TextEditor },
  props: {
    entry: {
      type: Object,
      required: true
    },
    entries: { type: Array }
  },
  data () {
    return {
      error: null,
      inited: false,

      file: null,
      content: '',

      saving: false
    }
  },
  computed: {
    filename () {
      return filename(this.path)
    },
    path () {
      return this.entry.path
    },
    readonly () {
      return !this.entry.meta.can_write
    }
  },
  created () {
    this.loadFile()
    window.addEventListener('beforeunload', this.onWindowUnload)
  },
  beforeDestroy () {
    window.removeEventListener('beforeunload', this.onWindowUnload)
  },
  watch: {
    content () {
      this.changeSaveState(false)
    }
  },
  methods: {
    async loadFile () {
      this.inited = false
      const path = this.path
      try {
        this.file = await entry(path)
        return await this.loadFileContent()
      } catch (e) {
        this.error = e
      } finally {
        this.inited = true
      }
    },
    async loadFileContent () {
      this.content = await getContent(this.path, this.entry.meta.access_key, true)
      this.$nextTick(() => {
        this.changeSaveState(true)
      })
      return this.content
    },
    async saveFile () {
      if (this.saving) {
        return
      }
      this.saving = true
      try {
        await uploadManager.upload({
          path: this.path,
          file: this.content,
          overwrite: true
        }, true)
        this.changeSaveState(true)
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.saving = false
      }
    },
    changeSaveState (saved) {
      this.saved = saved
      this.$emit('save-state', saved)
    },
    onKeyDown (e) {
      if (e.key === 's' && e.ctrlKey && !this.readonly) {
        e.preventDefault()
        this.saveFile()
      }
    },
    onWindowUnload (e) {
      if (!this.saved) {
        e.preventDefault()
        e.returnValue = ''
      }
    }
  }
}
</script>
<style lang="scss">
@import url("~codemirror/lib/codemirror.css");
@import url("~codemirror-github-light/lib/codemirror-github-light-theme.css");

.text-edit-view {
  position: relative;
  width: 800px;
  height: calc(100vh - 64px);
  padding-top: 60px;
  background-color: #fff;
  overflow: hidden;
  box-sizing: border-box;
  box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);

  .filename {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    margin: 0;
    text-align: center;
    border-bottom: 1px solid #eaecef;
    padding: 10px 4em;
    font-size: 28px;
    font-weight: normal;
    z-index: 10;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .header-button {
    position: absolute;
    top: 50%;
    transform: translateY(-50%);
  }

  .save-button {
    left: 2em;
  }

  .close-button {
    right: 1em;
  }

  .text-editor {
    height: 100%;

    .CodeMirror {
      height: 100%;
    }
  }

  .loading {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    display: flex;
    justify-content: center;
    align-items: center;
    width: 100%;
    height: 300px;
    font-weight: bold;
    font-size: 24px;
    text-transform: uppercase;
    user-select: none;

    animation: text-spacing 1s ease-in infinite alternate;
  }
}

@media screen and (max-width: 800px) {
  .text-edit-view {
    width: 100vw;
    height: 100vh;
    max-width: unset;
    margin: 0;
  }
}
</style>

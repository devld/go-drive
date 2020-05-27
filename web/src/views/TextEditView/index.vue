<template>
  <div class="text-edit-view">
    <h1 class="filename">
      <span>{{ filename }}</span>
      <span class="button-close" @click="$emit('close')">×</span>
    </h1>
    <text-editor v-if="!error" v-model="content" :filename="filename" line-numbers />
    <error-view v-else :status="error.status" :message="error.message" />
    <div v-if="!inited" class="loading">Loading...</div>
  </div>
</template>
<script>
import { filename } from '@/utils'
import { entry, getContent } from '@/api'
import TextEditor from '@/components/TextEditor'

export default {
  name: 'TextEditView',
  components: { TextEditor },
  props: {
    path: {
      type: String,
      required: true
    }
  },
  data () {
    return {
      error: null,
      inited: false,

      file: null,
      content: ''
    }
  },
  computed: {
    filename () {
      return filename(this.path)
    }
  },
  created () {
    this.loadFile()
  },
  methods: {
    async loadFile () {
      this.inited = false
      let path = this.path
      if (!path.startsWith('/')) path = '/' + path
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
      this.content = await getContent(this.path)
      return this.content
    }
  }
}
</script>
<style lang="scss">
@import url("~codemirror/lib/codemirror.css");
@import url("~codemirror-github-light/lib/codemirror-github-light-theme.css");

.text-edit-view {
  position: relative;
  height: 100%;
  padding-top: 53px;
  background-color: #fff;
  overflow: hidden;
  box-sizing: border-box;

  .filename {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    margin: 0;
    text-align: center;
    border-bottom: 1px solid #eaecef;
    padding: 10px 0;
    font-size: 28px;
    font-weight: normal;
    z-index: 10;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  }

  .button-close {
    position: absolute;
    top: 50%;
    right: 1em;
    transform: translateY(-50%);
    user-select: none;
    cursor: pointer;
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
</style>

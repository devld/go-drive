<template>
  <div class="text-edit-page">
    <h1 class="filename">{{ filename }}</h1>
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
  name: 'TextEditPage',
  components: { TextEditor },
  data () {
    return {
      path: null,
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
  mounted () {
    this.path = '/' + this.$route.params.path
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

.text-edit-page {
  position: relative;
  max-width: 880px;
  margin: 42px auto 0;
  padding-top: 16px;
  background-color: #fff;
  border-radius: 16px;
  overflow: hidden;

  .filename {
    margin: 0;
    text-align: center;
    border-bottom: 1px solid #eaecef;
    padding-bottom: 0.3em;
    font-size: 2em;
    font-weight: normal;
  }

  .editor {
    margin: 16px;

    .CodeMirror {
      height: unset;
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

@media screen and (max-width: 900px) {
  .text-edit-page {
    margin: 16px;
  }
}
</style>

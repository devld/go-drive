<template>
  <div class="download-view-page">
    <h1 class="page-title">
      <span>Download</span>
      <button class="plain-button close-button" @click="$emit('close')">
        <i-icon svg="#icon-close" />
      </button>
    </h1>
    <div class="page-content">
      <template v-if="singleEntry">
        <entry-icon :entry="singleEntry" />
        <h2 class="filename">{{ singleEntry.name }}</h2>
        <a
          class="download-button"
          target="_blank"
          :href="$.fileUrl(singleEntry.path, singleEntry.meta.access_key)"
        >
          Download
          <span class="file-size">{{ $.formatBytes(singleEntry.size) }}</span>
        </a>
      </template>
      <template v-else>
        <textarea
          ref="links"
          class="download-links"
          readonly
          :value="downloadLinks"
          v-focus
          @focus="downloadLinksFocus"
        ></textarea>
      </template>
    </div>
  </div>
</template>
<script>
export default {
  name: 'DownloadView',
  props: {
    entry: {
      type: [Array, Object],
      required: true
    },
    entries: { type: Array }
  },
  computed: {
    singleEntry () {
      if (Array.isArray(this.entry)) {
        if (this.entry.length === 1) return this.entry[0]
        return null
      } else {
        return this.entry
      }
    },
    downloadLinks () {
      if (this.singleEntry) return ''
      return this.entry.map(e => this.$.fileUrl(e.path, e.meta.access_key)).join('\n')
    }
  },
  methods: {
    downloadLinksFocus () {
      this.$refs.links.select()
      this.$refs.links.scrollTop = 0
      this.$refs.links.scrollLeft = 0
    }
  }
}
</script>
<style lang="scss">
.download-view-page {
  position: relative;
  width: 300px;
  background: #fff;
  box-shadow: 0 0 6px rgba(0, 0, 0, 0.1);
  padding: 16px 16px 20px;

  .page-title {
    font-size: 28px;
    margin: 0 0 20px;
    font-weight: normal;
    user-select: none;

    .close-button {
      float: right;
    }
  }

  .entry-icon {
    width: 150px;
    height: 150px;
  }

  .page-content {
    text-align: center;
  }

  .filename {
    font-weight: normal;
    font-size: 18px;
    margin: 10px 0;
    word-break: break-all;
  }

  .download-button {
    display: inline-block;
    color: #fff;
    background-color: #00bfa5;
    text-decoration: none;
    padding: 10px 16px;
    margin-top: 16px;
    transition: 0.3s;
    user-select: none;

    &:hover {
      box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
    }
  }

  .download-links {
    width: 100%;
    min-height: 200px;
    max-height: 40vh;
    outline: none;
    border: none;
    resize: none;
    white-space: pre;
    overflow: auto;
  }
}
</style>

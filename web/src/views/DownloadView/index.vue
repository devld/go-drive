<template>
  <div class="download-view-page">
    <h1 class="page-title">
      <span>Download</span>
      <button class="close-button" @click="$emit('close')"></button>
    </h1>
    <div class="page-content">
      <entry-icon :entry="entry" />
      <h2 class="filename">{{ entry.name }}</h2>
      <a class="download-button" target="_blank" :href="$.fileUrl(path)">Download</a>
    </div>
  </div>
</template>
<script>
import { filename } from '@/utils'

export default {
  name: 'DownloadView',
  props: {
    path: {
      type: String,
      required: true
    },
    entries: {
      type: Array,
      required: true
    }
  },
  computed: {
    entry () {
      return this.entries.find(e => e.name === filename(this.path))
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
    .icon {
      width: 150px;
      height: 150px;
    }
  }

  .page-content {
    text-align: center;
  }

  .filename {
    font-weight: normal;
    font-size: 18px;
    margin: 10px 0;
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
}
</style>

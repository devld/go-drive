<template>
  <div class="pdf-view-page">
    <h1 class="filename">
      <span :title="entry.name">{{ entry.name }}</span>
      <button
        class="close-button plain-button"
        title="Close"
        @click="$emit('close')"
      >
        <i-icon svg="#icon-close" />
      </button>
    </h1>

    <iframe
      :src="$.fileUrl(entry.path, entry.meta.access_key)"
      frameborder="0"
      class="pdf-view-page__iframe"
    ></iframe>
  </div>
</template>
<script>
export default {
  name: 'PdfView',
  props: {
    entry: {
      type: [Array, Object],
      required: true,
    },
    entries: { type: Array },
  },
}
</script>
<style lang="scss">
.pdf-view-page {
  width: 100vw;
  height: 100vh;
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  padding-top: 48px;
  box-sizing: border-box;
  @include var(background-color, secondary-bg-color);

  .filename {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    margin: 0;
    text-align: center;
    border-bottom: 1px solid;
    @include var(border-color, border-color);
    padding: 10px 2.5em;
    font-size: 20px;
    font-weight: normal;
    z-index: 10;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .close-button {
    position: absolute;
    top: 50%;
    right: 0.5em;
    transform: translateY(-50%);
  }

  iframe {
    width: 100%;
    flex: 1;
    border: none;
  }
}
</style>

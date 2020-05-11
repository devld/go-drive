<template>
  <div class="error-view">
    <span class="error-code" :title="message">{{ status || 'ERROR' }}</span>
    <span class="error-message">{{ ERROR_MESSAGES[status] || message }}</span>
  </div>
</template>
<script>

const ERROR_MESSAGES = {
  403: 'Operation Not Allowed',
  404: 'Resource Not Found',
  500: 'Server Error'
}

export default {
  name: 'ErrorView',
  props: {
    status: {
      type: [Number, String]
    },
    message: {
      type: String
    }
  },
  created () {
    this.ERROR_MESSAGES = ERROR_MESSAGES
  }
}
</script>
<style lang="scss">
.error-view {
  user-select: none;
  text-align: center;
  padding: 40px 0;

  .error-code {
    display: block;
    font-weight: bold;
    font-size: 80px;
    color: #787878;
    animation: text-flicker-in-glow 4s 2s linear infinite reverse both;
  }
}
</style>

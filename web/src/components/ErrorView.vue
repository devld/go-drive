<template>
  <div class="error-view">
    <span class="error-code" :title="message">{{ status || 'ERROR' }}</span>
    <span class="error-message">{{
      (ERROR_MESSAGES[status] && $t(ERROR_MESSAGES[status], { status })) ||
      message
    }}</span>
    <div class="back-button">
      <simple-button @click="$router.go(-1)">{{
        $t('app.go_back')
      }}</simple-button>
    </div>
  </div>
</template>
<script>
const ERROR_MESSAGES = {
  403: 'error.not_allowed',
  404: 'error.not_found',
  500: 'error.server_error',
}

export default {
  name: 'ErrorView',
  props: {
    status: {
      type: [Number, String],
    },
    message: {
      type: String,
    },
  },
  created() {
    this.ERROR_MESSAGES = ERROR_MESSAGES
  },
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

  .back-button {
    margin-top: 42px;
  }
}
</style>

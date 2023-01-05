<template>
  <div class="error-view">
    <span class="error-code" :title="message">{{ status || 'ERROR' }}</span>
    <span class="error-message">{{
      (ERROR_MESSAGES[status!] && $t(ERROR_MESSAGES[status!], { status })) ||
      message
    }}</span>
    <div class="back-button">
      <SimpleButton @click="$router.go(-1)">{{
        $t('app.go_back')
      }}</SimpleButton>
    </div>
  </div>
</template>
<script setup lang="ts">
const ERROR_MESSAGES: Record<number | string, string> = {
  403: 'error.not_allowed',
  404: 'error.not_found',
  500: 'error.server_error',
}

defineProps({
  status: {
    type: [Number, String],
  },
  message: {
    type: String,
  },
})
</script>
<style lang="scss">
.error-view {
  user-select: none;
  -webkit-user-select: none;
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

<template>
  <div class="error-view">
    <span class="error-code" :title="message">{{ status || 'ERROR' }}</span>
    <span>{{ message }}</span>
    <div class="back-button">
      <SimpleButton @click="$router.go(-1)">{{
        $t('app.go_back')
      }}</SimpleButton>
    </div>
  </div>
</template>
<script setup lang="ts">
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
  -webkit-user-select: none;
  user-select: none;
  text-align: center;
  padding: 40px 0;

  .error-code {
    display: block;
    font-weight: bold;
    font-size: 80px;
    color: var(--color-text-muted);
  }

  .back-button {
    margin-top: 42px;
  }
}
</style>

<template>
  <div class="error-page">
    <span class="error-code">{{ code }}</span>
    <span class="error-message">{{ messages[+code] }}</span>
  </div>
</template>
<script>
export default {
  name: 'ErroPage',
  props: {
    code: {
      type: String
    }
  },
  data () {
    return {
      messages: {
        403: 'Operation Not Allowed',
        404: 'Resource Not Found',
        500: 'Server Error'
      }
    }
  }
}
</script>
<style lang="scss" scoped>
.error-page {
  user-select: none;
  text-align: center;
  margin-top: 25vh;

  .error-code {
    display: block;
    font-weight: bold;
    font-size: 100px;
  }
}
</style>

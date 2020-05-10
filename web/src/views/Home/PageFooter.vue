<template>
  <div class="page-footer">
    <div class="markdown-body" v-markdown="readme" />
  </div>
</template>
<script>
export default {
  name: 'PageFooter',
  props: {
    readme: {
      type: String
    }
  }
}
</script>
<style lang="scss" scoped>
.page-footer {
  padding: 16px;
}
</style>

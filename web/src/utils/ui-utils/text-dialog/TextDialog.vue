<template functional>
  <div class="text-dialog__inner">{{ props.opts.message }}</div>
</template>
<script>
export default {
  name: 'TextDialogInner',
  props: {
    opts: {
      type: Object,
      required: true
    }
  }
}
</script>
<style lang="scss">
.text-dialog__inner {
  margin-top: 10px;
  min-width: 200px;
  max-width: 80vw;
}
</style>

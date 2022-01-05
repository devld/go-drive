<template>
  <div class="text-dialog__inner">{{ opts.message }}</div>
</template>
<script setup>
defineProps({
  opts: {
    type: Object,
    required: true,
  },
})
</script>
<style lang="scss">
.text-dialog__inner {
  margin-top: 10px;
  min-width: 200px;
  max-width: 400px;
  max-height: 40vh;
  word-break: break-all;
  overflow-x: hidden;
  overflow-y: auto;
}

@media screen and (max-width: 400px) {
  .text-dialog__inner {
    max-width: 80vw;
  }
}
</style>

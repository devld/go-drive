<template>
  <svg class="icon" aria-hidden>
    <use :xlink:href="svg" />
  </svg>
</template>
<script>
export default {
  name: 'IIcon',
  props: {
    svg: {
      type: String,
      required: true
    }
  }
}
</script>

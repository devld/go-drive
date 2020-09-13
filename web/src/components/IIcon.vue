<template functional>
  <svg
    class="icon"
    aria-hidden
    @click="listeners.click && listeners.click($event)"
    :class="[
      data.class,
      data.staticClass
    ]"
  >
    <use :xlink:href="props.svg" />
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

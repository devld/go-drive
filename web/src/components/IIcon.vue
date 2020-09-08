<template functional>
  <svg
    class="icon"
    aria-hidden
    @click="listeners.click && listeners.click()"
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

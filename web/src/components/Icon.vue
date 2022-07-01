<template>
  <svg class="icon" aria-hidden @click="emit('click', $event)">
    <use :xlink:href="svg" />
  </svg>
</template>
<script setup lang="ts">
defineProps({
  svg: {
    type: String,
    required: true,
  },
})

const emit = defineEmits<{ (e: 'click', event: MouseEvent): void }>()
</script>

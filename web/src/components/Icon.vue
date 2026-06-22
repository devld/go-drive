<template>
  <!-- eslint-disable vue/no-v-html -- content comes from the trusted static icon set -->
  <svg
    class="icon"
    aria-hidden="true"
    :viewBox="def?.viewBox"
    @click="emit('click', $event)"
    v-html="def?.content"
  />
  <!-- eslint-enable vue/no-v-html -->
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { icons, type IconName } from './icons'

const props = defineProps<{ name: IconName }>()

const def = computed(() => icons[props.name])

const emit = defineEmits<{ (e: 'click', event: MouseEvent): void }>()
</script>

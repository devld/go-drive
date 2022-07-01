<template>
  <h1 class="handler-title-bar">
    <span v-if="$slots.actions" class="handler-title-bar-actions">
      <slot name="actions" />
    </span>
    <span class="handler-title-bar-text" :title="props.title"
      ><slot>{{ props.title }}</slot></span
    >
    <button
      v-if="closeable"
      class="handler-title-bar-close plain-button"
      title="Close"
      @click="emit('close')"
    >
      <Icon svg="#icon-close" />
    </button>
  </h1>
</template>
<script setup lang="ts">
const props = defineProps({
  title: {
    type: String,
  },
  closeable: {
    type: Boolean,
    default: true,
  },
})

const emit = defineEmits<{ (e: 'close'): void }>()
</script>
<style lang="scss">
.handler-title-bar {
  display: flex;
  align-items: center;
  overflow: hidden;
  white-space: nowrap;
  user-select: none;
  padding: 0 10px;
  margin: 0;
  height: 48px;
}

.handler-title-bar-text {
  margin: 0 1em;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: center;
  font-size: 20px;
  font-weight: normal;
}

.handler-title-bar-actions {
  margin-right: auto;
  display: flex;
  align-items: center;
}

.handler-title-bar-close {
  margin-left: auto;
}
</style>

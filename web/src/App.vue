<template>
  <div v-if="inited" class="app">
    <RouterView />
  </div>
</template>
<script setup lang="ts">
import { alert } from '@/utils/ui-utils'
import { ref } from 'vue'
import { useAppStore } from './store'

const store = useAppStore()

const inited = ref(false)

store
  .init()
  .then(() => {
    inited.value = true
  })
  .catch((e) => {
    alert(e.message)
  })
</script>

<style lang="scss">
body {
  margin: 0;
  padding: 0;
  background-color: var(--body-bg-color);
  color: var(--primary-text-color);
}

.app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}
</style>

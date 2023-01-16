<template>
  <div v-if="store.inited" class="app">
    <RouterView />
  </div>
</template>
<script setup lang="ts">
import { alert } from '@/utils/ui-utils'
import { useAppStore } from './store'

const store = useAppStore()

store.init().catch((e) => {
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

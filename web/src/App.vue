<template>
  <div class="app">
    <router-view />
  </div>
</template>
<script setup>
import { alert } from '@/utils/ui-utils'
import { useStore } from 'vuex'

useStore()
  .dispatch('init')
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

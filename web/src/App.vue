<template>
  <div class="app">
    <router-view />
  </div>
</template>
<script>
export default {
  name: 'App',
  created () {
    this.$store.dispatch('init').catch(e => {
      this.$alert(e.message)
    })
  }
}
</script>

<style lang="scss">
body {
  margin: 0;
  padding: 0;
  background-color: #f2f5fa;
}

.app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}
</style>

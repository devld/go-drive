<template>
  <div class="misc-settings">
    <RootPermissions />
    <SearchIndex timer />
    <CleanInvalid />
    <CleanCache />
  </div>
</template>
<script lang="ts">
export default { name: 'MiscSettings' }
</script>
<script setup lang="ts">
import CleanCache from './CleanCache.vue'
import CleanInvalid from './CleanInvalid.vue'
import RootPermissions from './RootPermissions.vue'
import SearchIndex from './SearchIndex.vue'
</script>
<style lang="scss">
.misc-settings {
  padding: 16px;

  .section {
    padding-top: 1em;
    margin-bottom: 2em;

    &:not(:first-child) {
      border-top: solid 1px;
      border-color: var(--border-color);
    }
  }

  .section-title {
    margin: 0 0 16px;
    font-size: 20px;
    font-weight: normal;
  }
}
</style>

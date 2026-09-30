<template>
  <div class="misc-settings">
    <RootPermissions />
    <SearchIndex timer />
    <div class="section maintenance-section">
      <AdminDocHeading
        level="section"
        message-key="p.admin.docs.misc_maintenance"
      >
        {{ $t('p.admin.misc.maintenance') }}
      </AdminDocHeading>
      <CleanInvalid />
      <CleanCache />
    </div>
  </div>
</template>
<script lang="ts">
export default { name: 'MiscSettings' }
</script>
<script setup lang="ts">
import AdminDocHeading from '../AdminDocHeading.vue'
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
      border-color: var(--color-border);
    }
  }

  .maintenance-action + .maintenance-action {
    margin-top: 24px;
  }
}
</style>

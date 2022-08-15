<template>
  <div class="site-config">
    <OptionsConfigure
      :title="$t('p.admin.site.app_name')"
      :form="siteConfigForm"
    />
  </div>
</template>
<script setup lang="ts">
import { FormItem } from '@/types';
import { computed } from 'vue';
import OptionsConfigure from './OptionsConfigure.vue';

const siteConfigForm = computed<FormItem[]>(() => [
  { field: 'app.name', type: 'text' },
])
</script>
<style lang="scss">
.site-config {
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

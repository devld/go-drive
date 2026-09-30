<template>
  <div class="admin-doc-heading" :class="`admin-doc-heading--${level}`">
    <div class="admin-doc-heading__title-group">
      <component :is="headingTag" class="admin-doc-heading__title">
        <slot />
      </component>
      <AdminDocLinks v-if="messageKey" :message-key="messageKey" />
    </div>
    <div v-if="slots.actions" class="admin-doc-heading__actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, useSlots } from 'vue'
import AdminDocLinks from './AdminDocLinks.vue'

const props = withDefaults(
  defineProps<{
    messageKey?: string
    level?: 'page' | 'section' | 'subsection'
  }>(),
  { level: 'page' }
)

const headingTag = computed(() => (props.level === 'subsection' ? 'h3' : 'h2'))
const slots = useSlots()
</script>

<style scoped lang="scss">
.admin-doc-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  column-gap: 16px;
  row-gap: 4px;
}

.admin-doc-heading__title-group,
.admin-doc-heading__actions {
  display: flex;
  align-items: center;
}

.admin-doc-heading--page {
  margin: 12px 16px 8px;
}

.admin-doc-heading--section {
  margin: 0 0 12px;
}

.admin-doc-heading--subsection {
  margin: 0 0 6px;
}

.admin-doc-heading__title {
  margin: 0;
  font-weight: normal;
}

.admin-doc-heading--page .admin-doc-heading__title {
  font-size: 24px;
}

.admin-doc-heading--section .admin-doc-heading__title {
  font-size: 20px;
}

.admin-doc-heading--subsection .admin-doc-heading__title {
  font-size: 18px;
}
</style>

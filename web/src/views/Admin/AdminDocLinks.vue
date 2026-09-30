<template>
  <a
    v-if="documentationLink"
    :href="documentationLink.href"
    class="admin-doc-links"
    :title="documentationLink.label"
    :aria-label="documentationLink.label"
    target="_blank"
    rel="noreferrer noopener"
  >
    <Icon name="help" />
  </a>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '@go-drive/i18n'

const props = defineProps<{ messageKey: string }>()

const { t } = useI18n()
const documentationLink = computed(() => {
  const match = t(props.messageKey).match(/\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)/u)
  if (!match) return null

  return { label: match[1], href: match[2] }
})
</script>

<style scoped lang="scss">
.admin-doc-links {
  display: inline-flex;
  align-items: center;
  margin-left: 0.5em;
  color: inherit;
  text-decoration: none;
  cursor: help;
}
</style>

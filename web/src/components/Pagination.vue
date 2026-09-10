<template>
  <nav
    v-if="pageCount > 1"
    class="pagination"
    :aria-label="$t('app.pagination.label')"
  >
    <SimpleButton
      small
      type="info"
      :disabled="page <= 1"
      :title="$t('app.pagination.previous')"
      :aria-label="$t('app.pagination.previous')"
      @click="setPage(page - 1)"
    >
      ‹
    </SimpleButton>
    <span class="pagination__summary">
      {{ $t('app.pagination.summary', { page, pages: pageCount, total }) }}
    </span>
    <SimpleButton
      small
      type="info"
      :disabled="page >= pageCount"
      :title="$t('app.pagination.next')"
      :aria-label="$t('app.pagination.next')"
      @click="setPage(page + 1)"
    >
      ›
    </SimpleButton>
  </nav>
</template>
<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps({
  page: { type: Number, required: true },
  pageSize: { type: Number, required: true },
  total: { type: Number, required: true },
})

const emit = defineEmits<{ (e: 'update:page', page: number): void }>()

const pageCount = computed(() =>
  Math.max(1, Math.ceil(props.total / props.pageSize))
)

const setPage = (page: number) => {
  const next = Math.min(pageCount.value, Math.max(1, page))
  if (next !== props.page) emit('update:page', next)
}
</script>
<style lang="scss">
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  margin-top: 16px;

  &__summary {
    color: var(--color-text-muted);
    font-size: 14px;
    font-variant-numeric: tabular-nums;
  }
}
</style>

<template>
  <div
    v-if="state.active"
    class="entry-drag-status"
    :class="{ 'entry-drag-status--invalid': !!state.invalidReason }"
    role="status"
    aria-live="polite"
  >
    <span class="entry-drag-status__main">
      <Icon :name="state.invalidReason ? 'reject' : actionIcon" />
      <span class="entry-drag-status__action">{{ actionLabel }}</span>
      <template v-if="!state.invalidReason">
        <span
          class="entry-drag-status__name"
          :title="sourceLabel"
        >
          {{ sourceLabel }}
        </span>
        <span
          v-if="state.entries.length > 1"
          class="entry-drag-status__count"
        >
          {{ t('p.entry_drag.source_count', { n: state.entries.length }) }}
        </span>
        <span class="entry-drag-status__arrow">→</span>
        <span
          class="entry-drag-status__name"
          :title="targetLabel"
        >
          {{ targetLabel }}
        </span>
      </template>
    </span>
  </div>
</template>
<script setup lang="ts">
import type { IconName } from '@/components/icons'
import type {
  EntryDragState,
  EntryDropInvalidReason,
} from './useDrag'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  state: EntryDragState
}>()

const { t } = useI18n()

const invalidReasonKey: Record<EntryDropInvalidReason, string> = {
  'not-directory': 'p.entry_drag.not_directory',
  'target-readonly': 'p.entry_drag.target_readonly',
  'same-target': 'p.entry_drag.same_target',
  descendant: 'p.entry_drag.descendant',
}

const actionIcon = computed<IconName>(() => {
  if (props.state.action === 'copy') return 'copy'
  if (props.state.action === 'link') return 'path'
  return 'move'
})

const actionLabel = computed(() => {
  if (props.state.invalidReason) {
    return t(invalidReasonKey[props.state.invalidReason])
  }
  if (!props.state.action) return t('p.entry_drag.dragging_action')
  return t(`p.entry_drag.${props.state.action}_action`)
})

const sourceLabel = computed(() => props.state.entries[0]?.name ?? '')

const targetLabel = computed(() => {
  const targetPath = props.state.targetPath
  if (targetPath === undefined) return t('p.entry_drag.target_prompt')
  if (!targetPath) return t('app.root_path')
  return targetPath.split('/').filter(Boolean).at(-1) ?? targetPath
})
</script>
<style scoped lang="scss">
.entry-drag-status {
  position: fixed;
  z-index: 900;
  left: 50%;
  bottom: 24px;
  width: max-content;
  max-width: min(480px, calc(100vw - 24px));
  box-sizing: border-box;
  transform: translateX(-50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
  padding: 10px 14px;
  border-radius: 8px;
  background-color: var(--secondary-bg-color);
  color: var(--primary-text-color);
  box-shadow: var(--dialog-content-shadow);
  pointer-events: none;
  white-space: nowrap;
}

.entry-drag-status__main {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-width: 0;

  .icon {
    width: 20px;
    height: 20px;
    color: var(--btn-bg-color-default);
    flex: none;
  }
}

.entry-drag-status--invalid .icon {
  color: var(--btn-bg-color-danger);
}

.entry-drag-status__action,
.entry-drag-status__count,
.entry-drag-status__arrow {
  flex: none;
}

.entry-drag-status__action {
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.entry-drag-status__name {
  min-width: 0;
  max-width: 150px;
  flex: 0 1 auto;
  overflow: hidden;
  text-overflow: ellipsis;
  text-align: right;
}

.entry-drag-status__arrow {
  color: var(--secondary-text-color);
}

@media screen and (max-width: 700px) {
  .entry-drag-status {
    bottom: 16px;
  }
}
</style>

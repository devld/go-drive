<template>
  <component
    :is="variant === 'inline' ? 'span' : 'div'"
    class="loading-state"
    :class="[
      `loading-state--${variant}`,
      { 'loading-state--surface': surface },
    ]"
    role="status"
    aria-live="polite"
    aria-busy="true"
    :aria-label="accessibleLabel"
  >
    <span
      class="loading-state__content"
      :class="{
        'glass-surface': surface && ['page', 'dialog'].includes(variant),
      }"
    >
      <LoadingIndicator class="loading-state__indicator" />
      <span v-if="displayText" class="loading-state__text">{{ displayText }}</span>
      <slot />
    </span>
  </component>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { s } from '@/i18n'
import LoadingIndicator from './LoadingIndicator.vue'

const props = defineProps({
  text: {
    type: [String, Object] as PropType<I18nText>,
  },
  variant: {
    type: String as PropType<'inline' | 'panel' | 'overlay' | 'page' | 'dialog'>,
    default: 'panel',
  },
  surface: {
    type: Boolean,
    default: true,
  },
})

const { t } = useI18n()
const displayText = computed(() => s(props.text) || '')
const accessibleLabel = computed(() => displayText.value || t('app.loading'))
</script>

<style lang="scss">
.loading-state {
  box-sizing: border-box;
  color: var(--color-text-muted);
  -webkit-user-select: none;
  user-select: none;
}

.loading-state__indicator {
  flex: none;
}

.loading-state__content {
  box-sizing: border-box;
}

.loading-state--inline {
  display: inline-flex;
  align-items: center;
  line-height: 1.5;

  .loading-state__content {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }

  .loading-state__indicator {
    width: 1em;
    height: 1em;
  }
}

.loading-state--panel,
.loading-state--overlay,
.loading-state--page,
.loading-state--dialog {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;

  .loading-state__content {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
  }
}

.loading-state--panel {
  min-height: 96px;
  padding: 24px;

  &.loading-state--surface {
    background-color: var(--color-bg-elevated);
    border-radius: var(--radius-dialog);
  }

  .loading-state__indicator {
    width: 24px;
    height: 24px;
  }
}

.loading-state--overlay {
  position: absolute;
  inset: 0;
  z-index: 20;
  padding: 24px;
  background-color: var(--color-loading-overlay);
  -webkit-backdrop-filter: blur(2px);
  backdrop-filter: blur(2px);

  .loading-state__indicator {
    width: 32px;
    height: 32px;
  }
}

.loading-state--page {
  min-height: 100vh;
  padding: 24px;

  .loading-state__content {
    min-width: 176px;
    max-width: calc(100vw - 48px);
    padding: 22px 24px;
    border-radius: var(--radius-dialog);
    box-shadow: var(--shadow-elevated);
  }

  .loading-state__indicator {
    width: 40px;
    height: 40px;
  }
}

.loading-state--dialog {
  .loading-state__content {
    min-width: 176px;
    max-width: calc(100vw - 48px);
    padding: 22px 24px;
    border-radius: var(--radius-dialog);
    box-shadow: var(--shadow-elevated);
  }

  .loading-state__indicator {
    width: 40px;
    height: 40px;
  }

  .loading-state__text {
    max-width: min(520px, calc(100vw - 96px));
    line-height: 1.55;
    overflow-wrap: anywhere;
    word-break: normal;
  }
}
</style>

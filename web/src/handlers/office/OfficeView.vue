<template>
  <div
    class="office-view"
    data-ui="preview"
    data-handler="office"
  >
    <HandlerTitleBar :title="filename" @close="close">
      <template #actions>
        <span v-if="session" class="office-view-mode">
          {{ $t(`handler.office.${session.mode}`) }}
        </span>
      </template>
    </HandlerTitleBar>

    <div v-if="error" class="office-view-message">
      <strong>{{ $t('handler.office.load_failed') }}</strong>
      <span>{{ error }}</span>
    </div>
    <div v-else-if="!session" class="office-view-message">
      {{ $t('handler.office.loading') }}
    </div>
    <template v-else>
      <iframe
        :name="frameName"
        :title="$t('handler.office.frame_title')"
        class="office-frame"
        sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-top-navigation allow-popups-to-escape-sandbox"
        allow="clipboard-read 'src'; clipboard-write 'src'"
        allowfullscreen
        @load="loaded = true"
      ></iframe>
      <form
        ref="form"
        class="office-form"
        :action="session.actionUrl"
        :target="frameName"
        method="post"
      >
        <input name="access_token" :value="session.accessToken" />
        <input
          name="access_token_ttl"
          :value="`${session.accessTokenTtl}`"
        />
        <input name="user_id" :value="session.userId" />
        <input name="owner_id" :value="session.ownerId" />
      </form>
      <div v-if="!loaded" class="office-view-message office-view-loading">
        {{ $t('handler.office.loading') }}
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { createWOPISession, WOPISession } from '@/api'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import { Entry } from '@/types'
import { filename as filenameFn } from '@/utils'
import { computed, nextTick, onMounted, ref } from 'vue'
import { EntryHandlerContext } from '../types'

const props = defineProps({
  entry: {
    type: Object as PropType<Entry>,
    required: true,
  },
  entries: { type: Array as PropType<Entry[]> },
  ctx: {
    type: Object as PropType<EntryHandlerContext>,
    required: true,
  },
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'refresh'): void
}>()

const filename = computed(() => filenameFn(props.entry.path))
const frameName = `office-frame-${crypto.randomUUID()}`
const form = ref<HTMLFormElement | null>(null)
const session = ref<WOPISession | null>(null)
const error = ref('')
const loaded = ref(false)

onMounted(async () => {
  try {
    session.value = await createWOPISession(props.entry.path)
    await nextTick()
    form.value?.submit()
  } catch (e: any) {
    error.value = e?.message ?? String(e)
  }
})

const close = () => {
  emit('refresh')
  emit('close')
}
</script>

<style lang="scss">
.office-view {
  position: relative;
  width: 100vw;
  height: 100%;
  padding-top: 48px;
  overflow: hidden;
  background-color: var(--color-bg-elevated);
  box-sizing: border-box;

  .handler-title-bar {
    position: absolute;
    inset: 0 0 auto;
  }
}

.office-view-mode {
  color: var(--color-text-muted);
}

.office-frame {
  width: 100%;
  height: 100%;
  border: 0;
}

.office-form {
  display: none;
}

.office-view-message {
  position: absolute;
  inset: 48px 0 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 24px;
  text-align: center;
}

.office-view-loading {
  background-color: var(--color-bg-elevated);
}
</style>

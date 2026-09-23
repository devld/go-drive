<template>
  <div class="file-preview-host">
    <HandlerTitleBar :title="props.entry.name" @close="emit('close')" />
    <div class="file-preview-host__content">
      <component
        :is="props.previewer"
        v-if="loaded"
        v-bind="previewProps"
      />
      <ErrorView
        v-else-if="error"
        :status="error.status"
        :message="error.message"
      />
      <LoadingState v-else variant="panel" :text="$t('app.loading')" />
    </div>
  </div>
</template>
<script setup lang="ts">
import type { Component } from 'vue'
import { computed, ref, watch } from 'vue'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import { LoadingState } from '@go-drive/utils'
import ErrorView from '@/components/ErrorView.vue'
import { Entry } from '@/types'

type PreviewLoader = (entry: Entry) => Promise<unknown>
type PreviewPropsFactory = (data: unknown, entry: Entry) => Record<string, unknown>

const props = defineProps({
  entry: {
    type: Object as PropType<Entry>,
    required: true,
  },
  previewer: {
    type: Object as PropType<Component>,
    required: true,
  },
  load: {
    type: Function as PropType<PreviewLoader>,
    required: true,
  },
  makeProps: {
    type: Function as PropType<PreviewPropsFactory>,
    default: (data: unknown) => ({ data }),
  },
})

const emit = defineEmits<{ (e: 'close'): void }>()
const loaded = ref(false)
const data = ref<unknown>()
const error = ref<{ status?: number | string; message: string }>()
let loadId = 0

const previewProps = computed(() =>
  props.makeProps(data.value, props.entry)
)

const loadFile = async () => {
  const id = ++loadId
  loaded.value = false
  data.value = undefined
  error.value = undefined
  try {
    data.value = await props.load(props.entry)
    if (id !== loadId) return
    loaded.value = true
  } catch (e: any) {
    if (id !== loadId) return
    error.value = {
      status: e?.status,
      message: e?.message || 'Unable to preview this file',
    }
  }
}

watch(() => props.entry.path, loadFile, { immediate: true })
</script>
<style lang="scss">
.file-preview-host {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100vw;
  height: 100%;
  min-height: 0;
  padding-top: 48px;
  overflow: hidden;
  box-sizing: border-box;
  background-color: var(--color-bg-elevated);
  box-shadow: var(--shadow-elevated);

  > .handler-title-bar {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
  }
}

.file-preview-host__content {
  flex: 1;
  min-height: 0;
}
</style>

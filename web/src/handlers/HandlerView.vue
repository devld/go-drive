<template>
  <component
    :is="HANDLER_COMPONENTS[state!.component]"
    v-if="showing"
    :entry="state!.data.entry"
    :entries="state!.data.entries"
    :parent="state!.data.parent"
    :ctx="state!.ctx"
    @refresh="requestListRefresh"
    @close="requestClose"
    @entry-change="requestEntryChange"
    @save-state="requestSaveStateChange"
  />
</template>
<script setup lang="ts">
import { getHandler, HANDLER_COMPONENTS, isHandlerSupports } from './handlers'
import { ref, computed } from 'vue'
import { Entry } from '@/types'
import { EntryHandlerContext, EntryHandlerExecutionParams } from './types'

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'close', entry: Entry | Entry[]): void
  (e: 'entry-change', path: string): void
}>()

const state = ref<
  | {
      handler: string
      component: string

      data: EntryHandlerExecutionParams
      ctx: EntryHandlerContext

      savedState: boolean
    }
  | undefined
>()

const showing = computed(() => !!state.value)

const show = (
  handlerName: string,
  data: EntryHandlerExecutionParams,
  ctx: EntryHandlerContext
) => {
  const handler = getHandler(handlerName)
  if (!handler || !handler.view) return false
  if (!isHandlerSupports(handler, ctx, data)) return false

  state.value = {
    handler: handlerName,
    component: handler.view.name,
    data: { ...data },
    ctx,
    savedState: true,
  }
}

const hide = () => {
  if (!state.value) return
  state.value = undefined
}

const savedState = computed(() => state.value?.savedState)

const requestSaveStateChange = (s?: boolean) => {
  if (!state.value) return
  state.value!.savedState = !!s
}
const requestListRefresh = () => {
  emit('refresh')
}
const requestClose = () => {
  emit('close', state.value!.data.entry)
}
const requestEntryChange = (path: string) => {
  emit('entry-change', path)
}

defineExpose({
  show,
  hide,
  showing,
  savedState,
})
</script>

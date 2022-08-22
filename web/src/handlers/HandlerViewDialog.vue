<template>
  <DialogView class="entry-handler-dialog" eager :show="showing">
    <HandlerView ref="view" v-bind="events" />
  </DialogView>
</template>
<script setup lang="ts">
import HandlerView from './HandlerView.vue'
import { ref, computed } from 'vue'
import { Entry } from '@/types'
import {
  EntryHandlerExecutionParams,
  EntryHandlerExecutionOption,
  EntryHandlerViewHandle,
} from './types'

let opt: EntryHandlerExecutionOption
let data: EntryHandlerExecutionParams
let handlerName: string

const events = {
  onRefresh: () => opt.onRefresh?.(),
  onClose: (entry: Entry | Entry[]) => opt.onClose?.(entry),
  onEntryChange: (path: string) => opt.onEntryChange?.(path),
}

const view = ref<InstanceType<typeof HandlerView> | null>(null)
const showing = computed(() => view.value?.showing)

const handle: EntryHandlerViewHandle = {
  get handler() {
    return handlerName
  },
  get data() {
    return data
  },
  get showing() {
    return !!showing.value
  },
  get saved() {
    return view.value!.savedState ?? true
  },
  show(
    handlerName_: string,
    data_: EntryHandlerExecutionParams,
    opt_: EntryHandlerExecutionOption
  ) {
    handlerName = handlerName_
    opt = opt_
    data = data_
    return view.value!.show(handlerName, data, opt.ctx)
  },
  get hide() {
    return view.value!.hide
  },
}

defineExpose(handle)
</script>
<style lang="scss">
.entry-handler-dialog {
  .dialog-view__content {
    background-color: transparent;
  }
}
</style>

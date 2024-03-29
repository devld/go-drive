<template>
  <div class="permissions-view">
    <HandlerTitleBar :title="filename" @close="emit('close')">
      <template #actions>
        <SimpleButton
          :loading="saving"
          :disabled="!canSave"
          @click="savePermissions"
        >
          {{ $t('hv.permission.save') }}
        </SimpleButton>
      </template>
    </HandlerTitleBar>

    <PermissionsEditor
      ref="editorEl"
      :path="path"
      @save-state="setSaveState"
      @savable="canSave = $event"
    />
  </div>
</template>
<script setup lang="ts">
import { filename as filenameFn } from '@/utils'
import PermissionsEditor from '@/views/Admin/PermissionsEditor.vue'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import { alert } from '@/utils/ui-utils'
import { computed, ref } from 'vue'
import { Entry } from '@/types'

const props = defineProps({
  entry: {
    type: Object as PropType<Entry>,
    required: true,
  },
  entries: { type: Array as PropType<Entry[]> },
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save-state', v: boolean): void
}>()

const saving = ref(false)
const canSave = ref(true)

const path = computed(() => props.entry.path)

const filename = computed(() => filenameFn(path.value))

const editorEl = ref<InstanceType<typeof PermissionsEditor> | null>(null)

const savePermissions = async () => {
  saving.value = true
  try {
    await editorEl.value!.save()
  } catch (e: any) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}

const setSaveState = (saved: boolean) => {
  emit('save-state', saved)
}
</script>
<style lang="scss">
.permissions-view {
  position: relative;
  overflow-x: hidden;
  overflow-y: auto;
  width: 340px;
  padding-top: 48px;
  height: 300px;
  background-color: var(--secondary-bg-color);
  box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);

  .handler-title-bar {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
  }

  .permissions {
    .simple-table {
      width: 100%;
    }
  }
}
</style>

<template>
  <div class="permissions-view">
    <handler-title-bar :title="filename" @close="emit('close')">
      <template #actions>
        <simple-button
          :loading="saving"
          :disabled="!canSave"
          @click="savePermissions"
        >
          {{ $t('hv.permission.save') }}
        </simple-button>
      </template>
    </handler-title-bar>

    <permissions-editor
      ref="editorEl"
      v-model="permissions"
      :path="path"
      @save-state="setSaveState"
    />
  </div>
</template>
<script setup>
import { filename as filenameFn } from '@/utils'
import PermissionsEditor from '@/views/Admin/PermissionsEditor.vue'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import { alert } from '@/utils/ui-utils'
import { computed, ref, watch } from 'vue'

const props = defineProps({
  entry: {
    type: Object,
    required: true,
  },
  entries: { type: Array },
})

const emit = defineEmits(['close', 'save-state'])

const permissions = ref([])
const saving = ref(false)
const canSave = ref(true)

const path = computed(() => props.entry.path)

const filename = computed(() => filenameFn(path.value))

const editorEl = ref(null)

const savePermissions = async () => {
  saving.value = true
  try {
    await editorEl.value.save()
  } catch (e) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}

const setSaveState = (saved) => {
  emit('save-state', saved)
}

watch(
  () => permissions.value,
  () => {
    canSave.value = editorEl.value.validate()
  },
  { deep: true }
)
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

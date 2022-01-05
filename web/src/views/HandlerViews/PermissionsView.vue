<template>
  <div class="permissions-view">
    <h1 class="filename">
      <simple-button
        class="header-button save-button"
        :loading="saving"
        :disabled="!canSave"
        @click="savePermissions"
      >
        {{ $t('hv.permission.save') }}
      </simple-button>
      <span :title="filename">{{ filename }}</span>
      <button
        class="header-button close-button plain-button"
        title="Close"
        @click="emit('close')"
      >
        <i-icon svg="#icon-close" />
      </button>
    </h1>
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
  padding-top: 60px;
  height: 300px;
  background-color: var(--secondary-bg-color);
  box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);

  .filename {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    margin: 0;
    text-align: center;
    border-bottom: 1px solid #eaecef;
    border-color: var(--border-color);
    padding: 10px 2.5em;
    font-size: 20px;
    font-weight: normal;
    z-index: 10;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .header-button {
    position: absolute;
    top: 50%;
    transform: translateY(-50%);
  }

  .save-button {
    left: 1em;
  }

  .close-button {
    right: 0.5em;
  }

  .permissions {
    .simple-table {
      width: 100%;
    }
  }
}
</style>

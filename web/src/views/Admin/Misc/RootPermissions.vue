<template>
  <div class="section">
    <h1 class="section-title">
      {{ $t('p.admin.misc.permission_of_root') }}
      <simple-button
        :loading="saving"
        :disabled="!permissionsCanSave"
        @click="savePermissions"
      >
        {{ $t('p.admin.misc.save') }}
      </simple-button>
    </h1>
    <permissions-editor
      ref="permissionsEditorEl"
      v-model="permissions"
      :path="rootPath"
    />
  </div>
</template>
<script setup>
import { alert } from '@/utils/ui-utils'
import { ref, watch } from 'vue'
import PermissionsEditor from '../PermissionsEditor.vue'

const rootPath = ref('')
const saving = ref(false)
const permissionsCanSave = ref(true)

const permissionsEditorEl = ref(null)

const permissions = ref([])

const savePermissions = async () => {
  saving.value = true
  try {
    await permissionsEditorEl.value.save()
  } catch (e) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}

watch(
  () => permissions.value,
  () => {
    permissionsCanSave.value = permissionsEditorEl.value.validate()
  },
  { deep: true }
)
</script>

<template>
  <div class="section">
    <h1 class="section-title">
      {{ $t('p.admin.misc.permission_of_root') }}
      <SimpleButton
        :loading="saving"
        :disabled="!permissionsCanSave"
        @click="savePermissions"
      >
        {{ $t('p.admin.misc.save') }}
      </SimpleButton>
    </h1>
    <PermissionsEditor
      ref="permissionsEditorEl"
      :path="rootPath"
      @savable="permissionsCanSave = $event"
    />
  </div>
</template>
<script setup lang="ts">
import { alert } from '@/utils/ui-utils'
import { ref } from 'vue'
import PermissionsEditor from '../PermissionsEditor.vue'

const rootPath = ref('')
const saving = ref(false)
const permissionsCanSave = ref(true)

const permissionsEditorEl = ref<InstanceType<typeof PermissionsEditor> | null>(
  null
)

const savePermissions = async () => {
  saving.value = true
  try {
    await permissionsEditorEl.value!.save()
  } catch (e: any) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}
</script>

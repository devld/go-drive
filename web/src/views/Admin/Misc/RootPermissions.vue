<template>
  <div class="section">
    <AdminDocHeading
      level="section"
      message-key="p.admin.docs.misc_access_control"
    >
      {{ $t('p.admin.misc.permission_of_root') }}
      <template #actions>
        <SimpleButton
          :loading="saving"
          :disabled="!permissionsCanSave"
          @click="savePermissions"
        >
          {{ $t('p.admin.save') }}
        </SimpleButton>
      </template>
    </AdminDocHeading>
    <PermissionsEditor
      ref="permissionsEditorEl"
      :path="rootPath"
      @savable="permissionsCanSave = $event"
    />
  </div>
</template>
<script setup lang="ts">
import AdminDocHeading from '../AdminDocHeading.vue'
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

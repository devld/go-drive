<template>
  <div class="misc-settings">
    <div class="section">
      <h1 class="section-title">
        Permission of root
        <simple-button
          @click="savePermissions"
          :loading="saving"
          :disabled="!permissionsCanSave"
        >
          Save
        </simple-button>
      </h1>
      <permissions-editor
        ref="permissionsEditor"
        :path="rootPath"
        v-model="permissions"
      />
    </div>
    <div class="section">
      <h1 class="section-title">Clean invalid permissions and mounts</h1>
      <simple-button :loading="cleaning" @click="cleanPermissionsAndMounts">
        Clean
      </simple-button>
    </div>
  </div>
</template>
<script>
import { cleanPermissionsAndMounts } from '@/api/admin'
import PermissionsEditor from './PermissionsEditor'

export default {
  name: 'MiscSettings',
  components: { PermissionsEditor },
  data () {
    return {
      permissions: [],
      rootPath: '',
      saving: false,
      permissionsCanSave: true,

      cleaning: false
    }
  },
  watch: {
    permissions: {
      deep: true,
      handler () {
        this.permissionsCanSave = this.$refs.permissionsEditor.validate()
      }
    }
  },
  methods: {
    async savePermissions () {
      this.saving = true
      try {
        await this.$refs.permissionsEditor.save()
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.saving = false
      }
    },
    async cleanPermissionsAndMounts () {
      this.cleaning = true
      try {
        const n = await cleanPermissionsAndMounts()
        this.$alert(`${n} invalid paths cleaned`)
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.cleaning = false
      }
    }
  }
}
</script>
<style lang="scss">
.misc-settings {
  padding: 16px;

  .section {
    margin-bottom: 20px;
  }

  .section-title {
    margin: 0 0 16px;
    font-size: 20px;
    font-weight: normal;
  }
}
</style>

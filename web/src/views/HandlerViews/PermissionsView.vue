<template>
  <div class="permissions-view">
    <h1 class="filename">
      <simple-button
        class="header-button save-button"
        @click="savePermissions"
        :loading="saving"
        :disabled="!canSave"
      >
        {{ $t("hv.permission.save") }}
      </simple-button>
      <span :title="filename">{{ filename }}</span>
      <button
        class="header-button close-button plain-button"
        title="Close"
        @click="$emit('close')"
      >
        <i-icon svg="#icon-close" />
      </button>
    </h1>
    <permissions-editor
      ref="editor"
      :path="path"
      v-model="permissions"
      @save-state="setSaveState"
    />
  </div>
</template>
<script>
import { filename } from '@/utils'
import PermissionsEditor from '@/views/Admin/PermissionsEditor'

export default {
  name: 'PermissionsView',
  components: { PermissionsEditor },
  props: {
    entry: {
      type: Object,
      required: true
    },
    entries: { type: Array }
  },
  data () {
    return {
      permissions: [],

      saving: false,
      canSave: true
    }
  },
  watch: {
    permissions: {
      deep: true,
      handler () {
        this.canSave = this.$refs.editor.validate()
      }
    }
  },
  computed: {
    filename () {
      return filename(this.path)
    },
    path () {
      return this.entry.path
    }
  },
  methods: {
    async savePermissions () {
      this.saving = true
      try {
        await this.$refs.editor.save()
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.saving = false
      }
    },
    setSaveState (saved) {
      this.$emit('save-state', saved)
    }
  }
}
</script>
<style lang="scss">
.permissions-view {
  position: relative;
  overflow-x: hidden;
  overflow-y: auto;
  width: 340px;
  padding-top: 60px;
  height: 300px;
  @include var(background-color, secondary-bg-color);
  box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);

  .filename {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    margin: 0;
    text-align: center;
    border-bottom: 1px solid #eaecef;
    @include var(border-color, border-color);
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

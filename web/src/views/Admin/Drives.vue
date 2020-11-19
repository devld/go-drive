<template>
  <div class="drives-manager" :class="{ editing: !!drive }">
    <div class="drives-list">
      <div class="actions">
        <simple-button
          class="add-button"
          icon="#icon-add"
          title="Add drive"
          @click="addDrive"
        />
        <simple-button
          icon="#icon-refresh2"
          title="Reload drives to take effect"
          :loading="reloading"
          @click="reloadDrives"
        >
          Reload drives
        </simple-button>
      </div>
      <table class="simple-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Operation</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="d in drives"
            :key="d.name"
            :class="{ 'not-enabled-drive': !d.enabled }"
          >
            <td class="center">{{ d.name }}</td>
            <td class="center">{{ d.type }}</td>
            <td class="center">
              <simple-button
                title="Edit"
                small
                icon="#icon-edit"
                @click="editDrive(d)"
              />
              <simple-button
                title="Delete"
                type="danger"
                small
                icon="#icon-delete"
                @click="deleteDrive(d)"
              />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div class="drive-edit" v-if="drive">
      <div class="small-title">
        {{ edit ? `Edit drive: ${drive.name}` : "Add drive" }}
      </div>
      <div class="drive-form">
        <simple-form ref="baseForm" :form="baseForm" v-model="drive" />
        <simple-form
          v-if="drive.type && driveForms[drive.type]"
          ref="configForm"
          :form="driveForms[drive.type].configForm"
          v-model="drive.config"
        />
        <div class="form-item save-button">
          <simple-button small @click="saveDrive" :loading="saving"
            >Save</simple-button
          >
          <simple-button small type="info" @click="cancelEdit"
            >Cancel</simple-button
          >
        </div>
      </div>
      <div v-if="drive && driveInit" class="drive-init">
        <div class="small-title">
          Configure
          <span
            class="drive-init-state"
            :class="{ 'drive-configured': driveInit.configured }"
            >{{ driveInit.configured ? "Configured" : "Not configured" }}</span
          >
        </div>
        <o-auth-configure
          :key="drive.name"
          v-if="driveInit.oauth"
          :configured="driveInit.configured"
          :data="driveInit.oauth"
          :drive="drive"
          @refresh="getDriveInitConfigInfo"
        />
        <div class="drive-init-form" v-if="driveInit.form">
          <simple-form
            ref="initForm"
            :form="driveInit.form"
            v-model="driveInitForm"
          />
          <simple-button small @click="saveDriveConfig">Save</simple-button>
        </div>
      </div>
    </div>
    <div class="edit-tips" v-else>
      <simple-button icon="#icon-add" title="Add drive" @click="addDrive" small
        >Add</simple-button
      >&nbsp;or edit drive
    </div>
  </div>
</template>
<script>
import { createDrive, deleteDrive, getDrives, reloadDrives, updateDrive, getDriveInitConfig, initDrive } from '@/api/admin'
import Drives from './drives-config'

import OAuthConfigure from './drive-configure/OAuth'

export default {
  name: 'DrivesManager',
  components: { OAuthConfigure },
  data () {
    return {
      drives: [],

      drive: null,
      edit: false,
      saving: false,

      driveInit: null,
      driveInitForm: {},

      reloading: false,

      driveForms: Drives
    }
  },
  computed: {
    baseForm () {
      return [
        { field: 'name', label: 'Name', type: 'text', required: true, disabled: this.edit },
        { field: 'enabled', label: 'Enabled', type: 'checkbox' },
        {
          field: 'type', label: 'Type', type: 'select', required: true,
          options: Object.keys(Drives).map(type => ({
            name: Drives[type].name,
            value: type,
            title: Drives[type].description
          }))
        }
      ]
    }
  },
  created () {
    this.loadDrives()
  },
  methods: {
    async loadDrives () {
      try {
        this.drives = await getDrives()
      } catch (e) {
        this.$alert(e.message)
      }
    },
    addDrive () {
      this.drive = {
        name: '',
        enabled: '1',
        type: '',
        config: null
      }
      this.edit = false
    },
    editDrive (drive) {
      this.drive = {
        name: drive.name,
        enabled: drive.enabled ? '1' : '',
        type: drive.type,
        config: JSON.parse(drive.config)
      }
      this.edit = true
      this.getDriveInitConfigInfo()
    },
    async deleteDrive (drive) {
      this.$confirm({
        title: 'Delete drive',
        message: `Are you sure to delete drive ${drive.name}`,
        confirmType: 'danger',
        onOk: () => {
          return deleteDrive(drive.name)
            .then(() => {
              if (this.drive && drive.name === this.drive.name) {
                this.drive = null
              }
              this.loadDrives()
            }, e => {
              this.$alert(e.message)
              return Promise.reject(e)
            })
        }
      })
    },
    async saveDrive () {
      try {
        await Promise.all([
          this.$refs.baseForm.validate(),
          this.$refs.configForm && this.$refs.configForm.validate()
        ])
      } catch { return }

      const drive = {
        name: this.drive.name,
        enabled: !!this.drive.enabled,
        type: this.drive.type,
        config: JSON.stringify(this.drive.config)
      }
      this.saving = true
      try {
        if (this.edit) {
          await updateDrive(this.drive.name, drive)
        } else {
          await createDrive(drive)
        }
        this.edit = true
      } catch (e) {
        this.$alert(e.message)
        return
      } finally {
        this.saving = false
      }
      this.getDriveInitConfigInfo()
      this.loadDrives()
    },
    cancelEdit () {
      this.drive = null
      this.driveInit = null
      this.driveInitForm = {}
    },
    async getDriveInitConfigInfo () {
      this.$loading(true)
      try {
        this.driveInit = await getDriveInitConfig(this.drive.name)
        this.driveInitForm = (this.driveInit && this.driveInit.value) || {}
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.$loading()
      }
    },
    async saveDriveConfig () {
      try { await this.$refs.initForm.validate() } catch { return }
      this.$loading(true)
      try {
        await initDrive(this.drive.name, this.driveInitForm)
      } catch (e) {
        this.$alert(e.message)
        return
      } finally {
        this.$loading()
      }
      this.getDriveInitConfigInfo()
    },
    async reloadDrives () {
      this.reloading = true
      try {
        await reloadDrives()
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.reloading = false
      }
    }
  }
}
</script>
<style lang="scss">
.drives-manager {
  display: flex;

  .drive-edit {
    padding: 16px;
  }

  .drives-list {
    padding: 16px;
  }

  .not-enabled-drive {
    color: #999;
  }

  .drive-init {
    margin-top: 32px;
  }

  .drive-init-state {
    margin-left: 1em;
    font-size: 14px;
    color: #ffa000;
  }

  .drive-configured {
    color: #00e676;
  }

  .drive-init-form {
    margin-top: 1em;
  }

  .small-title {
    font-size: 18px;
    margin-bottom: 16px;
  }

  .edit-tips {
    flex: 1;
    display: flex;
    justify-content: center;
    align-items: center;
    white-space: pre;
  }

  .actions {
    margin-bottom: 16px;

    .add-button {
      display: none;
    }
  }

  .user-item {
    &:not(:last-child) {
      margin-right: 10px;
    }
  }

  .save-button {
    margin-top: 32px;
  }

  .drive-form {
    .simple-form:first-child {
      margin-bottom: 10px;
    }
  }

  @media screen and (max-width: 600px) {
    justify-content: center;

    .actions {
      .add-button {
        display: inline;
      }
    }

    .edit-tips {
      display: none;
    }

    &.editing {
      .edit-tips {
        display: block;
      }

      .drives-list {
        display: none;
      }
    }
  }
}
</style>

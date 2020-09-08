<template>
  <div class="drives-manager" :class="{ 'editing': !!drive }">
    <div class="drives-list">
      <div class="actions">
        <simple-button class="add-button" icon="#icon-add" title="Add drive" @click="addDrive" />
        <simple-button
          icon="#icon-refresh2"
          title="Reload drives to take effect"
          :loading="reloading"
          @click="reloadDrives"
        >Reload drives</simple-button>
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
          <tr v-for="d in drives" :key="d.name">
            <td class="center">{{ d.name }}</td>
            <td class="center">{{ d.type }}</td>
            <td class="center">
              <simple-button title="Edit" small icon="#icon-edit" @click="editDrive(d)" />
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
      <div class="small-title">{{ edit ? `Edit drive: ${drive.name}` : 'Add drive' }}</div>
      <div class="drive-form">
        <div class="form-item">
          <span class="label">Name</span>
          <input type="text" class="value" :disabled="edit" v-model="drive.name" />
        </div>
        <div class="form-item">
          <span class="label">Type</span>
          <select v-model="drive.type" class="value">
            <option
              v-for="t in driveTypes"
              :key="t.type"
              :value="t.type"
              :title="t.description"
            >{{ t.type }}</option>
          </select>
        </div>
        <div class="form-item" v-if="drive.type">
          <div class="value">
            <simple-form :form="driveConfigFormMap[drive.type]" v-model="drive.config" />
          </div>
        </div>
        <div class="form-item save-button">
          <simple-button small @click="saveDrive" :loading="saving">Save</simple-button>
          <simple-button small type="info" @click="drive = null">Cancel</simple-button>
        </div>
      </div>
    </div>
    <div class="edit-tips" v-else>
      <simple-button icon="#icon-add" title="Add drive" @click="addDrive" small>Add</simple-button>&nbsp;or edit drive
    </div>
  </div>
</template>
<script>
import { createDrive, deleteDrive, getDrives, reloadDrives, updateDrive } from '@/api/admin'
import { mapOf } from '@/utils'

export default {
  name: 'DrivesManager',
  data () {
    return {
      drives: [],

      drive: null,
      edit: false,
      saving: false,

      reloading: false,

      driveTypes: [
        {
          type: 'fs', description: 'Local file system drive',
          configForm: [
            { field: 'path', label: 'Root', type: 'text', description: 'The path of root', required: true }
          ]
        }
      ]
    }
  },
  computed: {
    driveConfigFormMap () {
      return mapOf(this.driveTypes, d => d.type, d => d.configForm)
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
        type: '',
        config: null
      }
      this.edit = false
    },
    editDrive (drive) {
      this.drive = {
        name: drive.name,
        type: drive.type,
        config: JSON.parse(drive.config)
      }
      this.edit = true
    },
    async deleteDrive (drive) {
      this.$confirm({
        title: 'Delete drive',
        message: `Are you sure to delete drive ${drive.name}`,
        confirmType: 'danger',
        onOk: () => {
          return deleteDrive(drive.name)
            .then(() => {
              this.loadDrives()
            }, e => {
              this.$alert(e.message)
              return Promise.reject(e)
            })
        }
      })
    },
    async saveDrive () {
      const drive = {
        name: this.drive.name,
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
        this.loadDrives()
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.saving = false
      }
    },
    async reloadDrives () {
      this.reloading = false
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

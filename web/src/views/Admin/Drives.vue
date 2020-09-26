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
          >Reload drives</simple-button
        >
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
          :form="driveForms[drive.type]"
          v-model="drive.config"
        />
        <div class="form-item save-button">
          <simple-button small @click="saveDrive" :loading="saving"
            >Save</simple-button
          >
          <simple-button small type="info" @click="drive = null"
            >Cancel</simple-button
          >
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
import { createDrive, deleteDrive, getDrives, reloadDrives, updateDrive } from '@/api/admin'

export default {
  name: 'DrivesManager',
  data () {
    return {
      drives: [],

      drive: null,
      edit: false,
      saving: false,

      reloading: false,

      driveForms: {
        fs: [
          { field: 'path', label: 'Root', type: 'text', description: 'The path of root', required: true }
        ],
        s3: [
          { field: 'id', label: 'AccessKey', type: 'text', required: true },
          { field: 'secret', label: 'SecretKey', type: 'password', required: true },
          { field: 'bucket', label: 'Bucket', type: 'text', required: true },
          { field: 'path_style', label: 'PathStyle', type: 'checkbox', description: 'Force use path style api' },
          { field: 'region', label: 'Region', type: 'text' },
          { field: 'endpoint', label: 'Endpoint', type: 'text', description: 'The S3 api endpoint' },
          { field: 'proxy_upload', label: 'ProxyIn', type: 'checkbox', description: 'Upload files to server proxy' },
          { field: 'proxy_download', label: 'ProxyOut', type: 'checkbox', description: 'Download files from server proxy' },
          { field: 'cache_ttl', label: 'CacheTTL', type: 'text', description: 'Cache time to live. Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".' },
          { field: 'max_cache', label: 'MaxCache', type: 'text', description: 'Maximum number of caches, if less than or equal to 0, no cache' }
        ]
      }
    }
  },
  computed: {
    baseForm () {
      return [
        { field: 'name', label: 'Name', type: 'text', required: true, disabled: this.edit },
        {
          field: 'type', label: 'Type', type: 'select', required: true,
          options: [
            { name: 'File system', value: 'fs', title: 'Local file system drive' },
            { name: 'S3', value: 's3', title: 'S3 compatible storage' }
          ]
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
      try {
        await Promise.all([
          this.$refs.baseForm.validate(),
          this.$refs.configForm && this.$refs.configForm.validate()
        ])
      } catch { return }

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
        this.drive = null
        this.loadDrives()
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.saving = false
      }
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

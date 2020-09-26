<template>
  <div class="groups-manager" :class="{ editing: !!group }">
    <div class="groups-list">
      <div class="actions">
        <simple-button icon="#icon-add" title="Add group" @click="addGroup" />
      </div>
      <table class="simple-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Operation</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="g in groups" :key="g.name">
            <td class="center">{{ g.name }}</td>
            <td class="center">
              <simple-button
                title="Edit"
                small
                icon="#icon-edit"
                @click="editGroup(g)"
              />
              <simple-button
                title="Delete"
                type="danger"
                small
                icon="#icon-delete"
                @click="deleteGroup(g)"
              />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div class="group-edit" v-if="group">
      <div class="small-title">
        {{ edit ? `Edit group: ${group.name}` : "Add group" }}
      </div>
      <div class="group-form">
        <simple-form ref="form" :form="groupForm" v-model="group" />
        <div class="form-item">
          <span class="label">Users</span>
          <div class="value">
            <span class="user-item" v-for="u in users" :key="u.username">
              <input
                type="checkbox"
                :value="u.username"
                v-model="group.users"
              />
              <span class="user-name">{{ u.username }}</span>
            </span>
          </div>
        </div>
        <div class="form-item save-button">
          <simple-button small @click="saveGroup" :loading="saving"
            >Save</simple-button
          >
          <simple-button small type="info" @click="group = null"
            >Cancel</simple-button
          >
        </div>
      </div>
    </div>
    <div class="edit-tips" v-else>
      <simple-button icon="#icon-add" title="Add group" @click="addGroup" small
        >Add</simple-button
      >&nbsp;or edit group
    </div>
  </div>
</template>
<script>
import { createGroup, deleteGroup, getGroup, getGroups, getUsers, updateGroup } from '@/api/admin'

export default {
  name: 'GroupsManager',
  data () {
    return {
      users: [],
      groups: [],

      group: null,
      edit: false,
      saving: false
    }
  },
  computed: {
    groupForm () {
      return [
        { field: 'name', label: 'Name', type: 'text', required: true, disabled: this.edit }
      ]
    }
  },
  created () {
    this.loadGroups()
    this.loadUsers()
  },
  methods: {
    async loadUsers () {
      try {
        this.users = await getUsers()
      } catch (e) {
        this.$alert(e.message)
      }
    },
    async loadGroups () {
      try {
        this.groups = await getGroups()
      } catch (e) {
        this.$alert(e.message)
      }
    },
    addGroup () {
      this.group = {
        name: '',
        users: []
      }
      this.edit = false
    },
    async editGroup (group) {
      try {
        const g = await getGroup(group.name)
        g.users = g.users.map(g => g.username)
        this.group = g
        this.edit = true
      } catch (e) {
        this.$alert(e.message)
      }
    },
    async deleteGroup (group) {
      this.$confirm({
        title: 'Delete group',
        message: `Are you sure to delete group ${group.name}`,
        confirmType: 'danger',
        onOk: () => {
          return deleteGroup(group.name)
            .then(() => {
              this.loadGroups()
            }, e => {
              this.$alert(e.message)
              return Promise.reject(e)
            })
        }
      })
    },
    async saveGroup () {
      try { await this.$refs.form.validate() } catch { return }
      const group = {
        name: this.group.name,
        users: this.group.users.map(username => ({ username }))
      }
      this.saving = true
      try {
        if (this.edit) {
          await updateGroup(this.group.name, group)
        } else {
          await createGroup(group)
        }
        this.loadGroups()
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.saving = false
      }
    }
  }
}
</script>
<style lang="scss">
.groups-manager {
  display: flex;

  .group-edit {
    padding: 16px;
  }

  .groups-list {
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
    display: none;
    margin-bottom: 16px;
  }

  .user-item {
    &:not(:last-child) {
      margin-right: 10px;
    }
  }

  .simple-form {
    margin-bottom: 10px;
  }

  .save-button {
    margin-top: 32px;
  }

  @media screen and (max-width: 600px) {
    justify-content: center;

    .actions {
      display: block;
    }

    .edit-tips {
      display: none;
    }

    &.editing {
      .edit-tips {
        display: block;
      }

      .groups-list {
        display: none;
      }
    }
  }
}
</style>

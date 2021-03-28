<template>
  <div class="users-manager" :class="{ editing: !!user }">
    <div class="users-list">
      <div class="actions">
        <simple-button
          icon="#icon-add"
          :title="$t('p.admin.user.add_user')"
          @click="addUser"
        />
      </div>
      <table class="simple-table">
        <thead>
          <tr>
            <th>{{ $t('p.admin.user.username') }}</th>
            <th>{{ $t('p.admin.user.operation') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.username">
            <td class="center">{{ u.username }}</td>
            <td class="center line">
              <simple-button
                :title="$t('p.admin.user.edit')"
                small
                icon="#icon-edit"
                @click="editUser(u)"
              />
              <simple-button
                :title="$t('p.admin.user.delete')"
                type="danger"
                small
                icon="#icon-delete"
                @click="deleteUser(u)"
              />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div class="user-edit" v-if="user">
      <div class="small-title">
        {{
          edit
            ? $t('p.admin.user.edit_user', { n: user.username })
            : $t('p.admin.user.add_user')
        }}
      </div>
      <div class="user-form">
        <simple-form ref="form" :form="userForm" v-model="user" />
        <div class="form-item">
          <span class="label">{{ $t('p.admin.user.groups') }}</span>
          <div class="value">
            <span class="group-item" v-for="g in groups" :key="g.name">
              <input type="checkbox" :value="g.name" v-model="user.groups" />
              <span class="group-name">{{ g.name }}</span>
            </span>
          </div>
        </div>
        <div class="form-item save-button">
          <simple-button small @click="saveUser" :loading="saving">
            {{ $t('p.admin.user.save') }}
          </simple-button>
          <simple-button small type="info" @click="user = null">
            {{ $t('p.admin.user.cancel') }}
          </simple-button>
        </div>
      </div>
    </div>
    <div class="edit-tips" v-else>
      <simple-button icon="#icon-add" title="Add user" @click="addUser" small>
        {{ $t('p.admin.user.add') }}
      </simple-button>
      {{ $t('p.admin.user.or_edit') }}
    </div>
  </div>
</template>
<script>
import {
  createUser,
  deleteUser,
  getGroups,
  getUser,
  getUsers,
  updateUser,
} from '@/api/admin'

export default {
  name: 'UsersManager',
  data() {
    return {
      users: [],
      groups: [],

      user: null,
      edit: false,
      saving: false,
    }
  },
  computed: {
    userForm() {
      return [
        {
          field: 'username',
          label: this.$t('p.admin.user.f_username'),
          type: 'text',
          required: true,
          disabled: this.edit,
        },
        {
          field: 'password',
          label: this.$t('p.admin.user.f_password'),
          type: 'text',
          required: !this.edit,
        },
      ]
    },
  },
  created() {
    this.loadUsers()
    this.loadGroups()
  },
  methods: {
    async loadUsers() {
      try {
        this.users = await getUsers()
      } catch (e) {
        this.$alert(e.message)
      }
    },
    async loadGroups() {
      try {
        this.groups = await getGroups()
      } catch (e) {
        this.$alert(e.message)
      }
    },
    addUser() {
      this.user = {
        username: '',
        password: '',
        groups: [],
      }
      this.edit = false
    },
    async editUser(user) {
      try {
        const u = await getUser(user.username)
        u.groups = u.groups.map((g) => g.name)
        this.user = u
        this.edit = true
      } catch (e) {
        this.$alert(e.message)
      }
    },
    async deleteUser(user) {
      this.$confirm({
        title: this.$t('p.admin.user.delete_user'),
        message: this.$t('p.admin.user.confirm_delete', { n: user.username }),
        confirmType: 'danger',
        onOk: () => {
          return deleteUser(user.username).then(
            () => {
              this.loadUsers()
            },
            (e) => {
              this.$alert(e.message)
              return Promise.reject(e)
            }
          )
        },
      })
    },
    async saveUser() {
      try {
        await this.$refs.form.validate()
      } catch {
        return
      }
      const user = {
        username: this.user.username,
        password: this.user.password,
        groups: this.user.groups.map((name) => ({ name })),
      }
      this.saving = true
      try {
        if (this.edit) {
          await updateUser(this.user.username, user)
        } else {
          await createUser(user)
        }
        this.loadUsers()
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.saving = false
      }
    },
  },
}
</script>
<style lang="scss">
.users-manager {
  display: flex;

  .user-edit {
    padding: 16px;
  }

  .users-list {
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

  .group-item {
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

      .users-list {
        display: none;
      }
    }
  }
}
</style>

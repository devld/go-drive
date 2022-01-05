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
    <div v-if="user" class="user-edit">
      <div class="small-title">
        {{
          edit
            ? $t('p.admin.user.edit_user', { n: user.username })
            : $t('p.admin.user.add_user')
        }}
      </div>
      <div class="user-form">
        <simple-form ref="formEl" v-model="user" :form="userForm" />
        <div class="form-item">
          <span class="label">{{ $t('p.admin.user.groups') }}</span>
          <div class="value">
            <span v-for="g in groups" :key="g.name" class="group-item">
              <input v-model="user.groups" type="checkbox" :value="g.name" />
              <span class="group-name">{{ g.name }}</span>
            </span>
          </div>
        </div>
        <div class="form-item save-button">
          <simple-button small :loading="saving" @click="saveUser">
            {{ $t('p.admin.user.save') }}
          </simple-button>
          <simple-button small type="info" @click="user = null">
            {{ $t('p.admin.user.cancel') }}
          </simple-button>
        </div>
      </div>
    </div>
    <div v-else class="edit-tips">
      <simple-button icon="#icon-add" title="Add user" small @click="addUser">
        {{ $t('p.admin.user.add') }}
      </simple-button>
      {{ $t('p.admin.user.or_edit') }}
    </div>
  </div>
</template>
<script setup>
import {
  createUser,
  deleteUser as deleteUserApi,
  getGroups,
  getUser,
  getUsers,
  updateUser,
} from '@/api/admin'
import { alert, confirm } from '@/utils/ui-utils'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const users = ref([])
const groups = ref([])

const user = ref(null)
const edit = ref(false)
const saving = ref(false)

const formEl = ref(null)

const userForm = computed(() => [
  {
    field: 'username',
    label: t('p.admin.user.f_username'),
    type: 'text',
    required: true,
    disabled: edit.value,
  },
  {
    field: 'password',
    label: t('p.admin.user.f_password'),
    type: 'text',
    required: !edit.value,
  },
])

const loadUsers = async () => {
  try {
    users.value = await getUsers()
  } catch (e) {
    alert(e.message)
  }
}

const loadGroups = async () => {
  try {
    groups.value = await getGroups()
  } catch (e) {
    alert(e.message)
  }
}

const addUser = () => {
  user.value = {
    username: '',
    password: '',
    groups: [],
  }
  edit.value = false
}

const editUser = async (user_) => {
  try {
    const u = await getUser(user_.username)
    u.groups = u.groups.map((g) => g.name)
    user.value = u
    edit.value = true
  } catch (e) {
    alert(e.message)
  }
}

const deleteUser = async (user_) => {
  confirm({
    title: t('p.admin.user.delete_user'),
    message: t('p.admin.user.confirm_delete', { n: user_.username }),
    confirmType: 'danger',
    onOk: () => {
      return deleteUserApi(user_.username).then(
        () => {
          if (user_.username === user.value?.username) {
            user.value = null
          }
          loadUsers()
        },
        (e) => {
          alert(e.message)
          return Promise.reject(e)
        }
      )
    },
  })
}

const saveUser = async () => {
  try {
    await formEl.value.validate()
  } catch {
    return
  }
  const data = {
    username: user.value.username,
    password: user.value.password,
    groups: user.value.groups.map((name) => ({ name })),
  }
  saving.value = true
  try {
    if (edit.value) {
      await updateUser(data.username, data)
    } else {
      await createUser(data)
      edit.value = true
    }
    loadUsers()
  } catch (e) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}

loadUsers()
loadGroups()
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

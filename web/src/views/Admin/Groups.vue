<template>
  <div class="groups-manager" :class="{ editing: !!group }">
    <div class="groups-list">
      <div class="actions">
        <simple-button
          icon="#icon-add"
          :title="$t('p.admin.group.add_group')"
          @click="addGroup"
        />
      </div>
      <table class="simple-table">
        <thead>
          <tr>
            <th>{{ $t('p.admin.group.name') }}</th>
            <th>{{ $t('p.admin.group.operation') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="g in groups" :key="g.name">
            <td class="center">{{ g.name }}</td>
            <td class="center line">
              <simple-button
                :title="$t('p.admin.group.edit')"
                small
                icon="#icon-edit"
                @click="editGroup(g)"
              />
              <simple-button
                :title="$t('p.admin.group.delete')"
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
    <div v-if="group" class="group-edit">
      <div class="small-title">
        {{
          edit
            ? $t('p.admin.group.edit_group', { n: group.name })
            : $t('p.admin.group.add_group')
        }}
      </div>
      <div class="group-form">
        <simple-form ref="formEl" v-model="group" :form="groupForm" />
        <div class="form-item">
          <span class="label">{{ $t('p.admin.group.users') }}</span>
          <div class="value">
            <span v-for="u in users" :key="u.username" class="user-item">
              <input
                v-model="group.users"
                type="checkbox"
                :value="u.username"
              />
              <span class="user-name">{{ u.username }}</span>
            </span>
          </div>
        </div>
        <div class="save-button">
          <simple-button small :loading="saving" @click="saveGroup">
            {{ $t('p.admin.group.save') }}
          </simple-button>
          <simple-button small type="info" @click="group = null">
            {{ $t('p.admin.group.cancel') }}
          </simple-button>
        </div>
      </div>
    </div>
    <div v-else class="edit-tips">
      <simple-button
        icon="#icon-add"
        :title="$t('p.admin.group.add_group')"
        small
        @click="addGroup"
      >
        {{ $t('p.admin.group.add') }}
      </simple-button>
      {{ $t('p.admin.group.or_edit') }}
    </div>
  </div>
</template>
<script setup>
import {
  createGroup,
  deleteGroup as deleteGroupApi,
  getGroup,
  getGroups,
  getUsers,
  updateGroup,
} from '@/api/admin'
import { alert, confirm } from '@/utils/ui-utils'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const users = ref([])
const groups = ref([])
const group = ref(null)
const edit = ref(false)
const saving = ref(false)

const groupForm = computed(() => [
  {
    field: 'name',
    label: t('p.admin.group.f_name'),
    type: 'text',
    required: true,
    disabled: edit.value,
  },
])

const formEl = ref(null)

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
const addGroup = () => {
  group.value = {
    name: '',
    users: [],
  }
  edit.value = false
}
const editGroup = async (group_) => {
  try {
    const g = await getGroup(group_.name)
    g.users = g.users.map((g) => g.username)
    group.value = g
    edit.value = true
  } catch (e) {
    alert(e.message)
  }
}
const deleteGroup = async (g) => {
  confirm({
    title: t('p.admin.group.delete_group'),
    message: t('p.admin.group.delete_group', { n: g.name }),
    confirmType: 'danger',
    onOk: () => {
      return deleteGroupApi(g.name).then(
        () => {
          if (g.name === group.value?.name) {
            group.value = null
          }
          loadGroups()
        },
        (e) => {
          alert(e.message)
          return Promise.reject(e)
        }
      )
    },
  })
}

const saveGroup = async () => {
  try {
    await formEl.value.validate()
  } catch {
    return
  }
  const g = {
    name: group.value.name,
    users: group.value.users.map((username) => ({ username })),
  }
  saving.value = true
  try {
    if (edit.value) {
      await updateGroup(group.value.name, g)
    } else {
      await createGroup(g)
    }
    edit.value = true
    loadGroups()
  } catch (e) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}

loadGroups()
loadUsers()
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

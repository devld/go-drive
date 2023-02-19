<template>
  <div class="groups-manager" :class="{ editing: !!group }">
    <div class="groups-list">
      <div class="actions">
        <SimpleButton
          icon="#icon-add"
          :title="$t('p.admin.group.add_group')"
          @click="addGroup"
        />
      </div>
      <table class="simple-table">
        <colgroup>
          <col style="min-width: 100px" />
          <col style="width: 80px" />
        </colgroup>
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
              <SimpleButton
                :title="$t('p.admin.group.edit')"
                small
                icon="#icon-edit"
                @click="editGroup(g)"
              />
              <SimpleButton
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
        <SimpleForm ref="formEl" v-model="group" :form="groupForm" />
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
          <SimpleButton small :loading="saving" @click="saveGroup">
            {{ $t('p.admin.group.save') }}
          </SimpleButton>
          <SimpleButton small type="info" @click="group = null">
            {{ $t('p.admin.group.cancel') }}
          </SimpleButton>
        </div>
      </div>
    </div>
    <div v-else class="edit-tips">
      <SimpleButton
        icon="#icon-add"
        :title="$t('p.admin.group.add_group')"
        small
        @click="addGroup"
      >
        {{ $t('p.admin.group.add') }}
      </SimpleButton>
      {{ $t('p.admin.group.or_edit') }}
    </div>
  </div>
</template>
<script setup lang="ts">
import {
  createGroup,
  deleteGroup as deleteGroupApi,
  getGroup,
  getGroups,
  getUsers,
  updateGroup,
} from '@/api/admin'
import { FormItem, Group, User } from '@/types'
import { alert, confirm } from '@/utils/ui-utils'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const users = ref<User[]>([])
const groups = ref<Group[]>([])
const group = ref<O | null>(null)
const edit = ref(false)
const saving = ref(false)

const groupForm = computed<FormItem[]>(() => [
  {
    field: 'name',
    label: t('p.admin.group.f_name'),
    type: 'text',
    required: true,
    disabled: edit.value,
  },
])

const formEl = ref<InstanceType<SimpleFormType> | null>(null)

const loadUsers = async () => {
  try {
    users.value = await getUsers()
  } catch (e: any) {
    alert(e.message)
  }
}
const loadGroups = async () => {
  try {
    groups.value = await getGroups()
  } catch (e: any) {
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
const editGroup = async (group_: Group) => {
  try {
    const g: O = await getGroup(group_.name)
    g.users = g.users!.map((g: User) => g.username)
    group.value = g
    edit.value = true
  } catch (e: any) {
    alert(e.message)
  }
}
const deleteGroup = async (g: Group) => {
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
    await formEl.value!.validate()
  } catch {
    return
  }
  const g = {
    name: group.value!.name,
    users: group.value!.users.map((username: string) => ({ username })),
  }
  saving.value = true
  try {
    if (edit.value) {
      await updateGroup(group.value!.name, g)
    } else {
      await createGroup(g)
    }
    edit.value = true
    loadGroups()
  } catch (e: any) {
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

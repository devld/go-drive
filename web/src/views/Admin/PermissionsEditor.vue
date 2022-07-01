<template>
  <div class="permissions">
    <table class="simple-table">
      <thead>
        <tr>
          <th>{{ $t('p.admin.p_edit.subject') }}</th>
          <th>{{ $t('p.admin.p_edit.rw') }}</th>
          <th>{{ $t('p.admin.p_edit.policy') }}</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(p, i) in permissions" :key="p.subject ?? ''">
          <td class="center">
            <select v-model="p.subject">
              <option
                v-for="s in subjects"
                :key="s.subject"
                :value="s.subject"
                :disabled="selectedSubjects[s.subject]"
              >
                {{
                  s.type === 'any'
                    ? $t('p.admin.p_edit.any')
                    : `${s.type}: ${s.name}`
                }}
              </option>
            </select>
          </td>
          <td class="center">
            <input v-model="p.permission.read" type="checkbox" />
            <input v-model="p.permission.write" type="checkbox" />
          </td>
          <td class="center">
            <SimpleButton
              :title="$t('p.admin.p_edit.reject')"
              icon="#icon-reject"
              small
              :type="
                p.policy === PathPermissionPolicy.REJECTED ? 'danger' : 'info'
              "
              @click="p.policy = PathPermissionPolicy.REJECTED"
            />
            <SimpleButton
              :title="$t('p.admin.p_edit.accept')"
              icon="#icon-accept"
              small
              :type="
                p.policy === PathPermissionPolicy.ACCEPTED ? undefined : 'info'
              "
              @click="p.policy = PathPermissionPolicy.ACCEPTED"
            />
          </td>
          <td>
            <SimpleButton
              type="danger"
              icon="#icon-delete"
              small
              @click="removePermission(i)"
            />
          </td>
        </tr>
        <tr>
          <td class="center" colspan="4">
            <SimpleButton icon="#icon-add" small @click="addPermission" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
<script setup lang="ts">
import {
  getGroups,
  getPermissions,
  getUsers,
  savePermissions,
} from '@/api/admin'
import { PathPermissionPerm, PathPermissionPolicy } from '@/types'
import { mapOf } from '@/utils'
import { alert } from '@/utils/ui-utils'
import { computed, nextTick, ref, watch, watchEffect } from 'vue'

interface PermissionSubject {
  type: 'any' | 'user' | 'group'
  name: string
  subject: string
}

interface EditPathPermission {
  subject: null | string
  permission: { read: boolean; write: boolean }
  policy: PathPermissionPolicy
}

const props = defineProps({
  path: {
    type: String,
    required: true,
  },
})

const emit = defineEmits<{
  (e: 'savable', v: boolean): void
  (e: 'save-state', v: boolean): void
}>()

const permissions = ref<EditPathPermission[]>([])
const subjects = ref<PermissionSubject[]>([])

const selectedSubjects = computed(() =>
  mapOf(
    permissions.value,
    (p) => p.subject!,
    () => true
  )
)

const validate = () => {
  for (const p of permissions.value) {
    if (p.subject === null) {
      return false
    }
  }
  return true
}

const addPermission = () => {
  permissions.value.push({
    subject: null,
    permission: { read: true, write: false },
    policy: 0,
  })
}
const removePermission = (i: number) => {
  permissions.value.splice(i, 1)
}

const loadPermissions = async () => {
  try {
    const data = await getPermissions(props.path)
    permissions.value = data.map((p) => ({
      subject: p.subject,
      permission: {
        read:
          (p.permission & PathPermissionPerm.Read) === PathPermissionPerm.Read,
        write:
          (p.permission & PathPermissionPerm.Write) ===
          PathPermissionPerm.Write,
      },
      policy: p.policy,
    }))
    nextTick(() => {
      setSaveState(true)
    })
  } catch (e: any) {
    alert(e.message)
  }
}

const save = async () => {
  await savePermissions(
    props.path,
    permissions.value.map((p) => ({
      subject: p.subject,
      permission:
        (p.permission.read
          ? PathPermissionPerm.Read
          : PathPermissionPerm.Empty) |
        (p.permission.write
          ? PathPermissionPerm.Write
          : PathPermissionPerm.Empty),
      policy: p.policy,
    }))
  )
  setSaveState(true)
}

const loadSubjects = async () => {
  try {
    const res = await Promise.all([getUsers(), getGroups()])
    subjects.value = [
      { type: 'any', name: '*', subject: 'ANY' },
      ...res[0].map(
        (u) =>
          ({
            type: 'user',
            name: u.username,
            subject: `u:${u.username}`,
          } as PermissionSubject)
      ),
      ...res[1].map(
        (g) =>
          ({
            type: 'group',
            name: g.name,
            subject: `g:${g.name}`,
          } as PermissionSubject)
      ),
    ]
  } catch (e: any) {
    alert(e.message)
  }
}
const setSaveState = (saved: boolean) => {
  emit('save-state', saved)
}

watch(
  () => permissions.value,
  () => {
    setSaveState(false)
  }
)

watchEffect(() => {
  loadPermissions()
})

loadSubjects()

watch(permissions, () => {
  emit('savable', validate())
})

defineExpose({ validate, save })
</script>

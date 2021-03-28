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
        <tr v-for="(p, i) in permissions" :key="p.subject">
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
            <input type="checkbox" v-model="p.permission.read" />
            <input type="checkbox" v-model="p.permission.write" />
          </td>
          <td class="center">
            <simple-button
              :title="$t('p.admin.p_edit.reject')"
              @click="p.policy = 0"
              icon="#icon-reject"
              small
              :type="p.policy === 0 ? 'danger' : 'info'"
            />
            <simple-button
              :title="$t('p.admin.p_edit.accept')"
              @click="p.policy = 1"
              icon="#icon-accept"
              small
              :type="p.policy === 1 ? '' : 'info'"
            />
          </td>
          <td>
            <simple-button
              type="danger"
              icon="#icon-delete"
              small
              @click="removePermission(i)"
            />
          </td>
        </tr>
        <tr>
          <td class="center" colspan="4">
            <simple-button icon="#icon-add" small @click="addPermission" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
<script>
import {
  getGroups,
  getPermissions,
  getUsers,
  savePermissions,
} from '@/api/admin'
import { mapOf } from '@/utils'

const PERMISSION_EMPTY = 0
const PERMISSION_READ = 1 << 0
const PERMISSION_WRITE = 1 << 1

export default {
  name: 'PermissionsEditor',
  props: {
    path: {
      type: String,
      required: true,
    },
    value: {
      type: Array,
    },
  },
  data() {
    return {
      permissions: [],
      subjects: [],
    }
  },
  created() {
    this.loadSubjects()
    this.loadPermissions()
  },
  computed: {
    selectedSubjects() {
      return mapOf(
        this.permissions,
        (p) => p.subject,
        () => true
      )
    },
  },
  watch: {
    value: {
      immediate: true,
      handler(val) {
        if (val === this.permissions) return
        this.permissions = [...(val || [])]
      },
    },
    path() {
      this.loadPermissions()
    },
    permissions() {
      this.setSaveState(false)
      this.$emit('input', this.permissions)
    },
  },
  methods: {
    validate() {
      for (const p of this.permissions) {
        if (p.subject === null) {
          return false
        }
      }
      return true
    },
    addPermission() {
      this.permissions.push({
        subject: null,
        permission: { read: true, write: false },
        policy: 0,
      })
    },
    removePermission(i) {
      this.permissions.splice(i, 1)
    },
    async loadPermissions() {
      try {
        const data = await getPermissions(this.path)
        this.permissions = data.map((p) => ({
          subject: p.subject,
          permission: {
            read: (p.permission & PERMISSION_READ) === PERMISSION_READ,
            write: (p.permission & PERMISSION_WRITE) === PERMISSION_WRITE,
          },
          policy: p.policy,
        }))
        this.$nextTick(() => {
          this.setSaveState(true)
        })
      } catch (e) {
        this.$alert(e.message)
      }
    },
    async save() {
      await savePermissions(
        this.path,
        this.permissions.map((p) => ({
          subject: p.subject,
          permission:
            (p.permission.read ? PERMISSION_READ : PERMISSION_EMPTY) |
            (p.permission.write ? PERMISSION_WRITE : PERMISSION_EMPTY),
          policy: p.policy,
        }))
      )
      this.setSaveState(true)
    },
    async loadSubjects() {
      try {
        const res = await Promise.all([getUsers(), getGroups()])
        this.subjects = [
          { type: 'any', name: '*', subject: 'ANY' },
          ...res[0].map((u) => ({
            type: 'user',
            name: u.username,
            subject: `u:${u.username}`,
          })),
          ...res[1].map((g) => ({
            type: 'group',
            name: g.name,
            subject: `g:${g.name}`,
          })),
        ]
      } catch (e) {
        this.$alert(e.message)
      }
    },
    setSaveState(saved) {
      this.$emit('save-state', saved)
    },
  },
}
</script>

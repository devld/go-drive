<template>
  <div
    v-if="store.isAdmin"
    class="admin-page glass-surface"
    data-ui="admin-page"
    data-surface="glass"
  >
    <ul class="menu-list">
      <li
        v-for="m in menus"
        :key="m.path"
        class="menu-item"
        :class="{ active: currentMenu === m.path }"
      >
        <RouterLink class="menu-link" :to="m.path">{{ m.name }}</RouterLink>
      </li>
    </ul>
    <div class="menu-content">
      <RouterView />
    </div>
  </div>
  <div v-else class="admin-page-permission-error-message">
    {{ $t('p.admin.admin_group_required') }}
  </div>
</template>
<script lang="ts">
export default { name: 'AdminPage' }
</script>
<script setup lang="ts">
import { useAppStore } from '@/store'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

const { t } = useI18n()

const store = useAppStore()
const router = useRouter()

const menus = [
  { name: t('p.admin.t_site'), path: '/admin/site' },
  { name: t('p.admin.t_users'), path: '/admin/users' },
  { name: t('p.admin.t_groups'), path: '/admin/groups' },
  { name: t('p.admin.t_drives'), path: '/admin/drives' },
  { name: t('p.admin.t_extra_drives'), path: '/admin/extra-drives' },
  { name: t('p.admin.t_jobs'), path: '/admin/jobs' },
  { name: t('p.admin.t_path_meta'), path: '/admin/path-meta' },
  { name: t('p.admin.t_file_buckets'), path: '/admin/file-buckets' },
  { name: t('p.admin.t_misc'), path: '/admin/misc' },
  { name: t('p.admin.t_statistics'), path: '/admin/stats' },
]

const currentMenu = computed(() => router.currentRoute.value.path)
</script>
<style lang="scss">
.admin-page-permission-error-message {
  height: 200px;
  display: flex;
  justify-content: center;
  align-items: center;
  -webkit-user-select: none;
  user-select: none;
}

.admin-page {
  box-sizing: border-box;
  max-width: 1000px;
  margin: 16px auto 0;
  background-color: var(--color-bg-glass);
  border: 1px solid var(--color-glass-border);
  border-radius: var(--radius-dialog);
  outline: none;
  display: flex;
  overflow: hidden;

  .menu-list {
    margin: 0;
    padding: 0;
  }

  .menu-item {
    margin: 0;
    padding: 0;
    list-style-type: none;
    white-space: nowrap;

    &:hover {
      background-color: var(--color-bg-hover);
    }

    &.active {
      background-color: var(--color-bg-selected);
    }
  }

  .menu-link {
    box-sizing: border-box;
    display: inline-block;
    width: 100%;
    padding: 8px 16px;
    text-decoration: none;
    color: var(--color-text);
  }

  // pc
  .menu-list {
    width: 120px;
    border-radius:
      calc(var(--radius-dialog) - 1px)
      0
      0
      calc(var(--radius-dialog) - 1px);
    overflow: hidden;
    border-right: solid 1px;
    border-color: var(--color-border);
  }

  .menu-item {
    &:not(:last-child) {
      border-bottom: solid 1px;
      border-color: var(--color-border);
    }
  }

  .menu-item:first-child .menu-link {
    border-radius: calc(var(--radius-dialog) - 1px) 0 0 0;
  }

  .menu-item:last-child .menu-link {
    border-radius: 0 0 0 calc(var(--radius-dialog) - 1px);
  }

  .menu-item:only-child .menu-link {
    border-radius:
      calc(var(--radius-dialog) - 1px)
      0
      0
      calc(var(--radius-dialog) - 1px);
  }

  .menu-content {
    flex: 1;
    overflow: hidden;
  }
}

@media screen and (max-width: 1000px) {
  .admin-page {
    margin: 16px;
    display: block;

    .menu-list {
      display: flex;
      width: 100%;
      border-radius:
        calc(var(--radius-dialog) - 1px)
        calc(var(--radius-dialog) - 1px)
        0
        0;
      overflow-x: auto;
      overflow-y: hidden;
      border-bottom: solid 1px;
      border-color: var(--color-border);
      border-right: none;
      padding: 0;
    }

    .menu-item {
      &:not(:last-child) {
        border-bottom: none;
      }
    }

    .menu-link {
      padding: 10px 16px;
    }

    .menu-item:first-child .menu-link {
      border-radius: calc(var(--radius-dialog) - 1px) 0 0 0;
    }

    .menu-item:last-child .menu-link {
      border-radius: 0 calc(var(--radius-dialog) - 1px) 0 0;
    }

    .menu-item:only-child .menu-link {
      border-radius:
        calc(var(--radius-dialog) - 1px)
        calc(var(--radius-dialog) - 1px)
        0
        0;
    }
  }
}
</style>

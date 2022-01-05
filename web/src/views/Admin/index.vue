<template>
  <div class="admin-page">
    <ul class="menu-list">
      <li
        v-for="m in menus"
        :key="m.path"
        class="menu-item"
        :class="{ active: currentMenu === m.path }"
      >
        <router-link class="menu-link" :to="m.path">{{ m.name }}</router-link>
      </li>
    </ul>
    <div class="menu-content">
      <router-view />
    </div>
  </div>
</template>
<script>
export default { name: 'AdminPage' }
</script>
<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

const { t } = useI18n()

const router = useRouter()

const menus = [
  { name: t('p.admin.t_users'), path: '/admin/users' },
  { name: t('p.admin.t_groups'), path: '/admin/groups' },
  { name: t('p.admin.t_drives'), path: '/admin/drives' },
  { name: t('p.admin.t_misc'), path: '/admin/misc' },
]

const currentMenu = computed(() => router.currentRoute.value.path)
</script>
<style lang="scss">
.admin-page {
  max-width: 900px;
  margin: 16px auto 0;
  background-color: var(--primary-bg-color);
  border-radius: 16px;
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

    &:hover {
      background-color: var(--hover-bg-color);
    }

    &.active {
      background-color: var(--select-bg-color);
    }
  }

  .menu-link {
    box-sizing: border-box;
    display: inline-block;
    width: 100%;
    padding: 8px 16px;
    text-decoration: none;
    color: var(--primary-text-color);
  }

  // pc
  .menu-list {
    width: 100px;
    padding: 16px 0 42px;
    border-right: solid 1px;
    border-color: var(--border-color);
  }

  .menu-item {
    &:not(:last-child) {
      border-bottom: solid 1px;
      border-color: var(--border-color);
    }
  }

  .menu-content {
    flex: 1;
  }
}

@media screen and (max-width: 900px) {
  .admin-page {
    margin: 16px;
    display: block;

    .menu-list {
      display: flex;
      width: 100%;
      overflow-x: auto;
      overflow-y: hidden;
      border-bottom: solid 1px;
      border-color: var(--border-color);
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
  }
}
</style>

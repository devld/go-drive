<template>
  <div class="admin-page">
    <ul class="menu-list">
      <li
        class="menu-item"
        :class="{ 'active': currentMenu === m.path }"
        v-for="m in menus"
        :key="m.path"
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
export default {
  name: 'Admin',
  data () {
    return {
      menus: [
        { name: 'Users', path: '/admin/users' },
        { name: 'Groups', path: '/admin/groups' },
        { name: 'Drives', path: '/admin/drives' }
      ]
    }
  },
  computed: {
    currentMenu () {
      return this.$route.path
    }
  }
}
</script>
<style lang="scss">
.admin-page {
  max-width: 880px;
  margin: 16px auto 0;
  background-color: #fff;
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
      background-color: #ebebeb;
    }

    &.active {
      background-color: #ebebeb;
    }
  }

  .menu-link {
    box-sizing: border-box;
    display: inline-block;
    width: 100%;
    padding: 8px 16px;
    text-decoration: none;
    color: #000;
  }

  // pc
  .menu-list {
    width: 120px;
    padding: 16px 0 42px;
    border-right: solid 1px #ebebeb;
  }

  .menu-item {
    &:not(:last-child) {
      border-bottom: solid 1px #ebebeb;
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
      border-bottom: solid 1px #ebebeb;
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

<template>
  <div class="entry-menu">
    <h2 class="entry-menu__entry">
      <entry-icon :entry="entry" />
      <span class="entry-menu__entry-name">{{ entry.name }}</span>
    </h2>
    <ul class="entry-menu__menus">
      <li
        @click="$emit('click', { entry, menu: m })"
        class="entry-menu__menu-item"
        v-for="(m, i) in menus"
        :key="i"
        :title="m.display.description"
      >{{ m.display.name }}</li>
    </ul>
  </div>
</template>
<script>
export default {
  name: 'EntryMenu',
  props: {
    menus: {
      type: Array,
      required: true
    },
    entry: {
      type: Object,
      required: true
    }
  }
}
</script>
<style lang="scss">
.entry-menu {
  display: block;
  background-color: #fff;
  padding: 20px 0;
  min-width: 280px;
  max-width: 90vw;
}

.entry-menu__entry {
  padding: 0 20px;
  margin: 16px 0;

  font-size: 26px;
  font-weight: normal;

  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.entry-menu__entry-name {
  margin-left: 0.5em;
}

.entry-menu__menus {
  margin: 0;
  padding: 0;
  user-select: none;
}

.entry-menu__menu-item {
  list-style-type: none;
  padding: 10px 20px;
  cursor: pointer;

  &:hover {
    background-color: rgba(0, 0, 0, 0.08);
  }
}
</style>

<template>
  <div class="entry-menu">
    <h2 class="entry-menu__entry" v-if="!multiple">
      <entry-icon :entry="entry" />
      <span class="entry-menu__entry-name">{{ entry.name }}</span>
    </h2>
    <ul class="entry-menu__menus">
      <li
        @click="$emit('click', { entry, menu: m })"
        class="entry-menu__menu-item"
        :class="m.display.type && `entry-menu__menu-item-${m.display.type}`"
        v-for="(m, i) in menus"
        :key="i"
        :title="m.display.description"
      >
        <span class="entry-menu__icon">
          <i-icon v-if="m.display.icon" :svg="m.display.icon" />
        </span>
        <span class="entry-menu__text">{{ m.display.name }}</span>
      </li>
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
      type: [Object, Array],
      required: true
    }
  },
  computed: {
    multiple () {
      return Array.isArray(this.entry)
    }
  }
}
</script>
<style lang="scss">
.entry-menu {
  @include var(background-color, secondary-bg-color);
  padding: 20px 0;
  width: 280px;
  overflow: hidden;
  user-select: none;
}

.entry-menu__entry {
  padding: 0 20px;
  margin: 0 0 16px;

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
  max-height: 40vh;
  margin: 0;
  padding: 0;
  user-select: none;
  overflow-x: hidden;
  overflow-y: auto;
}

.entry-menu__menu-item {
  display: flex;
  align-items: center;
  list-style-type: none;
  padding: 0 20px;
  cursor: pointer;
  transition: 0.1s;

  &:hover {
    @include var(background-color, hover-bg-color);
  }
}

.entry-menu__icon {
  margin: 6px 10px 6px 0;
  .icon {
    display: flex;
    width: 24px;
    height: 24px;
  }
}

.entry-menu__menu-item-danger {
  &:hover {
    color: #fff;
    background-color: #f56c6c;
  }
}

@media screen and (max-width: 600px) {
  .entry-menu {
    max-width: 90vw;
  }
}
</style>

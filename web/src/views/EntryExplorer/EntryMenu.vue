<template>
  <div class="entry-menu">
    <h2 v-if="!multiple" class="entry-menu__entry">
      <EntryIcon :entry="(entry as Entry)" />
      <span class="entry-menu__entry-name">{{ (entry as Entry).name }}</span>
    </h2>
    <ul class="entry-menu__menus">
      <li
        v-for="(m, i) in menus"
        :key="i"
        class="entry-menu__menu-item"
        :class="m.display.type && `entry-menu__menu-item-${m.display.type}`"
        :title="s(m.display.description)"
        @click="emit('click', { entry, menu: m })"
      >
        <span class="entry-menu__icon">
          <Icon v-if="m.display.icon" :svg="m.display.icon" />
        </span>
        <span class="entry-menu__text">{{ m.display.name }}</span>
      </li>
    </ul>
  </div>
</template>
<script setup lang="ts">
import { EntryHandlerMenuItem } from '@/handlers/types'
import { Entry } from '@/types'
import { computed } from 'vue'
import { EntryMenuClickData } from './types'

const props = defineProps({
  menus: {
    type: Array as PropType<EntryHandlerMenuItem[]>,
    required: true,
  },
  entry: {
    type: [Object, Array] as PropType<Entry | Entry[]>,
    required: true,
  },
})

const emit = defineEmits<{ (e: 'click', v: EntryMenuClickData): void }>()

const multiple = computed(() => Array.isArray(props.entry))
</script>
<style lang="scss">
.entry-menu {
  background-color: var(--secondary-bg-color);
  padding: 20px 0;
  width: 280px;
  overflow: hidden;
  user-select: none;
  -webkit-user-select: none;
}

.entry-menu__entry {
  display: flex;

  padding: 0 20px;
  margin: 0 0 16px;

  font-size: 26px;
  font-weight: normal;

  .entry-icon {
    vertical-align: middle;
  }
}

.entry-menu__entry-name {
  flex: 1;
  display: inline-block;
  margin-left: 0.5em;
  line-height: 42px;
  vertical-align: middle;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.entry-menu__menus {
  max-height: 40vh;
  margin: 0;
  padding: 0;
  user-select: none;
  -webkit-user-select: none;
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
    background-color: var(--hover-bg-color);
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

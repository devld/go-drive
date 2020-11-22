<template>
  <div
    class="entry-item"
    :class="[
      `entry-item--${entry.type}`,
      `entry-item--view-${viewMode}`,
      entry.type === 'file' ? `entry-item--ext-${ext}` : '',
    ]"
    @click="$emit('click', $event)"
    :title="
      entry.name === '..'
        ? ''
        : `${entry.name}\n${$.formatTime(entry.mod_time)}\n` +
          `${$.formatBytes(entry.size)}`
    "
  >
    <i-icon
      class="entry-item__icon"
      v-if="icon"
      :svg="icon"
      @click="$emit('icon-click', $event)"
    />
    <entry-icon
      v-else
      class="entry-item__icon"
      :entry="entry"
      @click="$emit('icon-click', $event)"
    />
    <span class="entry-item__info">
      <span class="entry-item__name">
        <i v-if="entry.meta.is_mount">@</i>{{ entry.name }}
      </span>
      <span class="entry-item__modified-time">{{
        entry.mod_time >= 0 ? $.formatTime(entry.mod_time) : ""
      }}</span>
      <span class="entry-item__size">{{
        entry.size >= 0 ? $.formatBytes(entry.size) : ""
      }}</span>
    </span>
  </div>
</template>
<script>
import { filenameExt } from '@/utils'

export default {
  name: 'EntryItem',
  props: {
    entry: {
      type: Object,
      required: true
    },
    icon: {
      type: String
    },
    viewMode: {
      type: String,
      default: 'line',
      validator: val => val === 'line' || val === 'block'
    }
  },
  computed: {
    ext () {
      return filenameExt(this.entry.name)
    }
  }
}
</script>
<style lang="scss">
.entry-item__name {
  i {
    color: #999;
  }
}

.entry-item--view-line {
  display: flex;
  cursor: pointer;
  padding: 4px 16px;

  .entry-item__icon {
    width: 42px;
    height: 42px;
    margin-right: 0.5em;
    font-size: 42px;
  }

  .entry-item__info {
    flex: 1;
    display: flex;
    align-items: center;
    overflow: hidden;
  }

  .entry-item__name {
    flex: 1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .entry-item__modified-time {
    white-space: nowrap;
  }

  .entry-item__size {
    width: 80px;
    text-align: right;
    white-space: nowrap;
  }
}

.entry-item--view-block {
  $size: 90px;
  width: $size;
  padding: 10px;

  .entry-icon__thumbnail {
    transition: 0.3s;
  }

  &:hover {
    .entry-icon__thumbnail {
      transform: scale(1.2);
    }
  }

  .entry-item__icon {
    width: 100%;
    height: $size;
    display: block;
    font-size: $size;
    margin-bottom: 10px;
  }

  .entry-item__name {
    display: block;
    white-space: nowrap;
    text-align: center;
    overflow: hidden;
    text-overflow: ellipsis;
    font-size: 12px;
  }

  .entry-item__modified-time {
    display: none;
  }

  .entry-item__size {
    display: none;
  }
}
</style>

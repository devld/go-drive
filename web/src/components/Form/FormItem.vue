<template>
  <div class="form-item">
    <span v-if="item.label" class="label">
      {{ item.label }}
      <span
        class="form-item-description"
        v-if="item.description"
        :title="item.description"
        @click="showDescription"
      >
        <i-icon svg="#icon-help" />
      </span>
    </span>
    <input
      v-if="item.type === 'text'"
      class="value"
      type="text"
      :value="value"
      @input="textInput"
      :required="item.required"
    />
  </div>
</template>
<script>
export default {
  name: 'FormItem',
  props: {
    value: {
      type: [String, Number]
    },
    item: {
      type: Object,
      required: true
    }
  },
  methods: {
    textInput (e) {
      this.$emit('input', e.target.value)
    },
    showDescription () {
      this.$alert(this.item.description)
    }
  }
}
</script>

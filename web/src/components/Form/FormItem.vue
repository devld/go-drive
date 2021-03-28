<template>
  <div class="form-item" :class="{ error: !!error, required: item.required }">
    <span v-if="item.label" class="label">
      <span>{{ item.label }}</span>
      <span class="form-item-required" v-if="item.required">*</span>
    </span>
    <span v-if="item.description" class="description">
      {{ item.description }}
    </span>
    <textarea
      v-if="item.type === 'textarea'"
      class="value"
      :name="item.field"
      :value="value"
      @input="textInput"
      :required="item.required"
      :disabled="item.disabled"
      rows="4"
    />
    <input
      v-if="item.type === 'text'"
      class="value"
      type="text"
      :name="item.field"
      :value="value"
      @input="textInput"
      :required="item.required"
      :disabled="item.disabled"
    />
    <input
      v-if="item.type === 'password'"
      class="value"
      type="password"
      :name="item.field"
      :value="value"
      @input="textInput"
      :required="item.required"
      :disabled="item.disabled"
    />
    <input
      v-if="item.type === 'checkbox'"
      class="value"
      type="checkbox"
      :name="item.field"
      :checked="!!value"
      @input="checkboxInput"
      :required="item.required"
      :disabled="item.disabled"
    />
    <select
      v-if="item.type === 'select'"
      class="value"
      :name="item.field"
      :value="value"
      @input="selectInput"
      :required="item.required"
      :disabled="item.disabled"
    >
      <option
        v-for="o in item.options"
        :key="o.value"
        :value="o.value"
        :title="o.title"
        :disabled="o.disabled"
      >
        {{ o.name }}
      </option>
    </select>
    <span v-if="error" class="form-item-error">{{ error }}</span>
  </div>
</template>
<script>
export default {
  name: 'FormItem',
  props: {
    value: {
      type: [String, Number],
    },
    item: {
      type: Object,
      required: true,
    },
  },
  data() {
    return {
      error: null,
    }
  },
  methods: {
    async validate() {
      if (this.item.required && !this.value) {
        this.error = this.$t('form.required_msg', { f: this.item.label })
        throw new Error(this.error)
      }
      return this.value
    },
    clearError() {
      this.error = null
    },
    textInput(e) {
      this.$emit('input', e.target.value)
      this.clearError()
    },
    checkboxInput(e) {
      this.$emit('input', e.target.checked ? '1' : '')
      this.clearError()
    },
    selectInput(e) {
      this.$emit('input', e.target.value)
      this.clearError()
    },
  },
}
</script>
<style lang="scss">
.form-item.error {
  position: relative;
  padding-bottom: 24px;

  .value {
    border: solid 1px red;
  }
}

.form-item-required {
  color: red;
}

.form-item-error {
  position: absolute;
  bottom: 0;
  right: 0;
  color: red;
}
</style>

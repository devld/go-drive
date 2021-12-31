<template>
  <form
    class="simple-form"
    @submit="onSubmit"
    :autocomplete="noAutoComplete ? 'off' : 'on'"
  >
    <form-item
      v-for="item in form"
      :key="item.field"
      ref="fields"
      :item="item"
      v-model="data[item.field]"
      @input="emitInput"
    />
  </form>
</template>
<script>
import FormItem from './FormItem'

export default {
  name: 'SimpleForm',
  components: { FormItem },
  props: {
    form: {
      type: Array,
      required: true,
    },
    value: {
      type: Object,
    },
    noAutoComplete: {
      type: Boolean,
    },
  },
  watch: {
    value: {
      immediate: true,
      deep: true,
      handler(val) {
        if (val === this.data) return
        this.data = val || {}
      },
    },
  },
  data() {
    return {
      data: {},
    }
  },
  created() {
    this.fillDefaultValue()
  },
  methods: {
    async validate() {
      await Promise.all(this.$refs.fields.map(f => f.validate()))
    },
    clearError() {
      this.$refs.fields.forEach(f => {
        f.clearError()
      })
    },
    onSubmit(e) {
      e.preventDefault()
    },
    fillDefaultValue() {
      if (this.value) return
      const dat = {}
      for (const f of this.form) {
        if (f.defaultValue) dat[f.field] = f.defaultValue
      }
      this.data = dat
      this.emitInput()
    },
    emitInput() {
      this.$emit('input', this.data)
    },
  },
}
</script>
<style lang="scss">
.simple-form {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;

  .form-item {
    width: 232px;
  }
}
</style>

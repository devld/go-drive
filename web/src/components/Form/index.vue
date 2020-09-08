<template>
  <div class="simple-form">
    <form-item
      v-for="item in form"
      :key="item.field"
      :item="item"
      v-model="data[item.field]"
      @input="emitInput"
    />
  </div>
</template>
<script>
import FormItem from './FormItem'

export default {
  name: 'SimpleForm',
  components: { FormItem },
  props: {
    form: {
      type: Array,
      required: true
    },
    value: {
      type: Object
    }
  },
  watch: {
    value: {
      immediate: true,
      deep: true,
      handler (val) {
        if (val === this.data) return
        this.data = val ? { ...val } : {}
      }
    }
  },
  data () {
    return {
      data: {}
    }
  },
  methods: {
    emitInput () {
      this.$emit('input', this.data)
    }
  }
}
</script>

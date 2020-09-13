<template>
  <div class="file-exists-confirm-inner">
    <p>'{{ opts.filename }}' already exists, override or skip?</p>
    <p>
      <input type="checkbox" v-model="all" />
      Apply for all
    </p>
  </div>
</template>
<script>
export default {
  name: 'FileExistsConfirmDialogInner',
  props: {
    opts: {
      type: Object,
      required: true
    }
  },
  data () {
    return {
      all: false
    }
  },
  methods: {
    beforeConfirm () {
      return { all: this.all }
    },
    beforeCancel () {
      console.log('before cancel')
      return { all: this.all }
    }
  }
}
</script>

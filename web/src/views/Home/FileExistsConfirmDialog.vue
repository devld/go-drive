<template>
  <div class="file-exists-confirm-inner">
    <p>{{ $t('p.home.file_exists', { n: opts.filename }) }}</p>
    <p>
      <input type="checkbox" v-model="all" />
      {{ $t('p.home.apply_all') }}
    </p>
  </div>
</template>
<script>
export default {
  name: 'FileExistsConfirmDialogInner',
  props: {
    opts: {
      type: Object,
      required: true,
    },
  },
  data() {
    return {
      all: false,
    }
  },
  methods: {
    beforeConfirm() {
      return { all: this.all }
    },
    beforeCancel() {
      return { all: this.all }
    },
  },
}
</script>

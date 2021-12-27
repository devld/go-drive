<template>
  <div class="oauth-configure">
    <simple-button @click="doOauth">{{ data.text }}</simple-button>
    <div class="oauth-principal" v-if="data.principal">
      {{ $t('p.admin.oauth_connected', { p: data.principal }) }}
    </div>
  </div>
</template>
<script>
import { initDrive } from '@/api/admin'
export default {
  name: 'OAuthConfigure',
  props: {
    configured: {
      type: Boolean,
      required: true,
    },
    data: {
      required: true,
    },
    drive: {
      type: Object,
      required: true,
    },
  },
  created() {
    window.addEventListener('message', this.authorized)
  },
  beforeDestroy() {
    window.removeEventListener('message', this.authorized)
    if (this._w) {
      this._w.close()
    }
  },
  methods: {
    doOauth() {
      if (this._w) this._w.close()

      const win = window.open(
        this.data.url,
        this.data.title,
        'width=400,height=600,menubar=0,toolbar=0'
      )
      this._w = win
    },
    async authorized({ data }) {
      this._w.close()
      if (!data.oauth) return

      if (data.error) {
        console.error('OAuth error: ', data)
        this.$alert(data.error + ': ' + data.data.error_description)
        return
      }

      this.$loading(true)
      try {
        await initDrive(this.drive.name, data.data)
      } catch (e) {
        this.$alert(e.message)
        return
      } finally {
        this.$loading()
      }
      this.$emit('refresh')
    },
  },
}
</script>
<style lang="scss">
.oauth-configure {
  .oauth-principal {
    margin-top: 0.5em;
    font-size: 14px;
    color: var(--secondary-text-color);
  }
}
</style>

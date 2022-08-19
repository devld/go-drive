import { Component, Plugin } from 'vue'

import Icon from './Icon.vue'
import { EntryIcon, EntryLink, EntryItem, EntryList, PathBar } from './entry'
import ErrorView from './ErrorView.vue'
import DialogView from './DialogView/index.vue'
import FloatButton from './FloatButton'
import SimpleButton from './SimpleButton'
import SimpleFormItem from './Form/FormItem.vue'
import SimpleForm from './Form/index.vue'
import SimpleDropdown from './SimpleDropdown.vue'
import ProgressBar from './ProgressBar.vue'

const components: O<Component> = {
  Icon,
  SimpleButton,
  SimpleForm,
  SimpleFormItem,
  SimpleDropdown,
  EntryIcon,
  EntryLink,
  EntryList,
  EntryItem,
  PathBar,
  ErrorView,
  DialogView,
  FloatButton,
  ProgressBar,
}

export default {
  install(app) {
    Object.keys(components).forEach((key) => {
      app.component(key, components[key])
    })
  },
} as Plugin

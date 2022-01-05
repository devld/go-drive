import IIcon from './IIcon.vue'
import EntryIcon from './EntryItem/EntryIcon.vue'
import EntryLink from './EntryLink.vue'
import EntryItem from './EntryItem/index.vue'
import EntryList from './EntryList.vue'
import PathBar from './PathBar.vue'
import ErrorView from './ErrorView.vue'
import DialogView from './DialogView/index.vue'
import FloatButton from './FloatButton.vue'
import SimpleButton from './SimpleButton.vue'
import SimpleFormItem from './Form/FormItem.vue'
import SimpleForm from './Form/index.vue'
import SimpleDropdown from './SimpleDropdown.vue'
import ProgressBar from './ProgressBar.vue'

const components = {
  IIcon,
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
  /**
   * @param {import('vue').App} app
   */
  install(app) {
    Object.keys(components).forEach((key) => {
      app.component(key, components[key])
    })
  },
}

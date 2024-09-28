import type { PropType as VuePropType } from 'vue'

// components
import Icon from '@/components/Icon.vue'
import SimpleButton from '@/components/SimpleButton'
import SimpleForm, { SimpleFormItem } from '@/components/Form'
import SimpleDropdown from '@/components/SimpleDropdown.vue'
import {
  EntryItem,
  EntryIcon,
  EntryLink,
  EntryList,
  PathBar,
} from '@/components/entry'
import ErrorView from '@/components/ErrorView.vue'
import DialogView from '@/components/DialogView'
import FloatButton from '@/components/FloatButton.vue'
import ProgressBar from '@/components/ProgressBar.vue'

import { s } from '@/i18n'

declare global {
  declare type O<T = any> = Record<string, T>

  declare type Fn<R = any> = () => R
  declare type Fn1<A = any, R = any> = (arg1: A) => R
  declare type Fnn<
    A extends any[] = any[],
    R = any,
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    F extends (...args: A) => any = (...args: A) => R
  > = (...args: A) => R

  declare type PromiseValue<T> = Promise<T> | T

  declare type I18nTextObject = { key: string; args?: O<any> }
  declare type I18nText = string | I18nTextObject

  declare type PropType<T> = VuePropType<T>

  interface Window {
    ___config___: {
      /** api base URL */
      api: string
      /** app title */
      appName: string
      /** app version: vX.X.X */
      version: string
    }
  }

  // components type
  declare type SimpleFormType = typeof SimpleForm
  declare type EntryListType = typeof EntryList
}

declare module 'vue' {
  export interface ComponentCustomProperties {
    s: typeof s
  }

  export interface GlobalComponents {
    Icon: typeof Icon
    SimpleButton: typeof SimpleButton
    SimpleForm: typeof SimpleForm
    SimpleFormItem: typeof SimpleFormItem
    SimpleDropdown: typeof SimpleDropdown
    EntryItem: typeof EntryItem
    EntryIcon: typeof EntryIcon
    EntryLink: typeof EntryLink
    EntryList: typeof EntryList
    PathBar: typeof PathBar
    ErrorView: typeof ErrorView
    DialogView: typeof DialogView
    FloatButton: typeof FloatButton
    ProgressBar: typeof ProgressBar
  }
}

declare module 'vue-router' {
  interface RouteMeta extends Record<string, any> {
    title?: I18nText
  }
}

export default global

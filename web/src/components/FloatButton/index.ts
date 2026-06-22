import type { IconName } from '@/components/icons'
import FloatButton from './FloatButton.vue'

export interface FloatButtonItem extends O {
  title?: I18nText
  slot: string
  icon?: IconName
}

export interface FloatButtonClickEventData {
  button: FloatButtonItem
  index: number
}

export default FloatButton

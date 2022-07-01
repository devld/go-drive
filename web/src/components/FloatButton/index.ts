import FloatButton from './FloatButton.vue'

export interface FloatButtonItem extends O {
  title?: I18nText
  slot: string
  icon?: string
}

export interface FloatButtonClickEventData {
  button: FloatButtonItem
  index: number
}

export default FloatButton

export * from './model'

export interface ApiError {
  message: string
}

export type FormItemType =
  | 'textarea'
  | 'text'
  | 'password'
  | 'checkbox'
  | 'select'
  | 'form'

export interface FormItemOption {
  name: I18nText
  title?: I18nText
  value: string
  disabled?: boolean
}

export interface FormItemForm {
  key: string
  name?: I18nText
  form: FormItem[]
}

export interface FormItemForms {
  addText?: I18nText
  maxItems?: number
  forms: FormItemForm[]
}

export interface BaseFormItem extends O {
  label?: I18nText
  type?: FormItemType
  field?: string
  required?: boolean
  description?: I18nText
  disabled?: boolean

  options?: FormItemOption[]

  forms?: FormItemForms

  defaultValue?: string
}

export interface FormItem extends BaseFormItem {
  class?: string
  width?: string | number
  slot?: string
  placeholder?: I18nText
  validate?: (v: any) => PromiseValue<true | I18nText | undefined>
}

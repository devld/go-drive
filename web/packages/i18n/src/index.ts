import type { Plugin } from 'vue'
import { createI18n } from 'vue-i18n'
export { useI18n } from 'vue-i18n'
import enUS from './lang/en-US.json'
import enUSPreview from './lang/preview/en-US.json'

export interface I18nTextObject {
  key: string
  args?: Record<string, unknown>
}

export type I18nText = string | I18nTextObject

const DEFAULT_LANG = 'en-US'
const loadedLanguages: string[] = [DEFAULT_LANG]

const withPreviewMessages = (
  messages: Record<string, unknown>,
  previewMessages: Record<string, unknown>
) => ({
  ...messages,
  preview: previewMessages,
})

const i18n = createI18n({
  legacy: false,
  warnHtmlMessage: false,
  globalInjection: true,
  locale: DEFAULT_LANG,
  fallbackLocale: DEFAULT_LANG,
  messages: {
    [DEFAULT_LANG]: withPreviewMessages(enUS, enUSPreview),
  } as Record<string, any>,
})

function loadLanguage(lang: string) {
  if (i18n.global.locale.value === lang) return lang
  if (loadedLanguages.includes(lang)) return _setLang(lang)
  return Promise.all([
    import(`./lang/${lang}.json`),
    import(`./lang/preview/${lang}.json`),
  ]).then(([msgs, previewMessages]) => {
    i18n.global.setLocaleMessage(
      lang,
      withPreviewMessages(msgs.default, previewMessages.default)
    )
    loadedLanguages.push(lang)
    return _setLang(lang)
  })
}

function _setLang(lang: string) {
  i18n.global.locale.value = lang
  if (typeof document !== 'undefined') {
    document.querySelector('html')?.setAttribute('lang', lang)
  }
  return lang
}

export function getLang() {
  return i18n.global.locale.value
}

export async function setLang(lang: string) {
  try {
    return await loadLanguage(lang)
  } catch {
    return getLang()
  }
}

function _tFn(this: I18nTextObject) {
  return (
    i18n.global as unknown as {
      t: (key: string, data: Record<string, unknown>) => string
    }
  ).t(this.key, this.args ?? {})
}

export function T(
  key: string,
  args?: Record<string, unknown>
): I18nTextObject {
  const o = { key, args, t: '' }
  Object.defineProperty(o, 'i18n', { enumerable: false, get: () => true })
  o.toString = _tFn
  return o
}

export function isT(o: any): o is I18nTextObject {
  return !!(o && o.i18n === true && o.toString === _tFn)
}

export const s = (t?: I18nText) => t?.toString()

export default {
  install(app) {
    i18n.install(app)
    app.config.globalProperties.s = s
  },
} as Plugin

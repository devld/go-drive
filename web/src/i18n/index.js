import Vue from 'vue'
import VueI18N from 'vue-i18n'
import enUS from './lang/en-US'

Vue.use(VueI18N)

const DEFAULT_LANG = 'en-US'
const loadedLanguages = [DEFAULT_LANG]

const i18n = new VueI18N({
  locale: DEFAULT_LANG,
  fallbackLocale: DEFAULT_LANG,
  messages: { [DEFAULT_LANG]: enUS },
})

function loadLanguage(lang) {
  if (i18n.locale === lang) return lang
  if (loadedLanguages.includes(lang)) return _setLang(lang)
  import(/* webpackChunkName: "lang-[request]" */ `./lang/${lang}`).then(
    msgs => {
      i18n.setLocaleMessage(lang, msgs.default)
      loadedLanguages.push(lang)
      return _setLang(lang)
    }
  )
}

function _setLang(lang) {
  i18n.locale = lang
  document.querySelector('html').setAttribute('lang', lang)
  return lang
}

export function getLang() {
  return i18n.locale
}

export async function setLang(lang) {
  try {
    return await loadLanguage(lang)
  } catch {
    return getLang()
  }
}

function _tFn() {
  return i18n.t(this.key, this.args)
}

/**
 * @param {string} key i18n text key
 * @param {any} [args]
 */
export function T(key, args) {
  const o = { key, args, t: '', i18n: true }
  o.toString = _tFn
  return o
}

export function isT(o) {
  return o && o.i18n === true
}

export default i18n

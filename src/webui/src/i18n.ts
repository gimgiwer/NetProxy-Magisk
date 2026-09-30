import zhCN from './locales/zh-CN.json' with { type: 'json' }
import enUS from './locales/en-US.json' with { type: 'json' }
import ruRU from './locales/ru-RU.json' with { type: 'json' }

export interface LocaleMeta {
  code: string
  name: string
  nativeName: string
}

export type LocaleDictionary = Record<string, unknown>
export type LocaleListener = (locale: string) => void

export const STORAGE_KEY = 'netproxy.language'
export const SING_BOX_STORAGE_KEY = 'sing-box-dashboard.language'

// sing-box 控制台语言代码映射
export const SING_BOX_LANG_MAP: Record<string, string> = {
  'zh-CN': 'zh-Hans',
  'en-US': 'en',
  'ru-RU': 'ru'
}

interface LocaleEntry {
  meta: LocaleMeta
  dict: LocaleDictionary
  singBoxLang?: string
}

const registry: Record<string, LocaleEntry> = {}
const listeners = new Set<LocaleListener>()

// 注册语言元数据与词典
export function registerLocale(meta: LocaleMeta, dict: LocaleDictionary, singBoxLang?: string): void {
  registry[meta.code] = {
    meta,
    dict,
    singBoxLang: singBoxLang || SING_BOX_LANG_MAP[meta.code] || 'en'
  }
}

// 初始化默认支持语言
registerLocale(
  { code: 'zh-CN', name: 'Chinese (Simplified)', nativeName: '简体中文' },
  zhCN as unknown as LocaleDictionary,
  'zh-Hans'
)
registerLocale(
  { code: 'en-US', name: 'English (US)', nativeName: 'English' },
  enUS as unknown as LocaleDictionary,
  'en'
)
registerLocale(
  { code: 'ru-RU', name: 'Russian', nativeName: 'Русский' },
  ruRU as unknown as LocaleDictionary,
  'ru'
)

export const SUPPORTED_LOCALES = ['zh-CN', 'en-US', 'ru-RU'] as const
export type SupportedLocale = typeof SUPPORTED_LOCALES[number]

export function getSupportedLocales(): LocaleMeta[] {
  return Object.values(registry).map(entry => entry.meta)
}

export function getLocaleMeta(code: string): LocaleMeta | undefined {
  return registry[code]?.meta
}

// 自动检测浏览器首选语言
export function detectLocale(): string {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored && registry[stored]) {
      return stored
    }
  } catch {
    // 忽略无法访问 localStorage 的异常
  }

  if (typeof navigator !== 'undefined') {
    const navLangs = navigator.languages && navigator.languages.length
      ? navigator.languages
      : [navigator.language]

    for (const lang of navLangs) {
      if (!lang) continue
      const lower = lang.toLowerCase()
      if (lower.startsWith('zh')) return 'zh-CN'
      if (lower.startsWith('ru')) return 'ru-RU'
      if (lower.startsWith('en')) return 'en-US'
    }
  }

  return 'en-US'
}

let currentLocale = detectLocale()

// 同步 sing-box 控制台语言配置
function syncSingBoxLanguage(locale: string): void {
  const targetLang = registry[locale]?.singBoxLang || SING_BOX_LANG_MAP[locale]
  if (!targetLang) return
  try {
    localStorage.setItem(SING_BOX_STORAGE_KEY, targetLang)
  } catch {
    // 忽略异常
  }
}

// 初始化时确保 sing-box 语言同步
syncSingBoxLanguage(currentLocale)

export function getLocale(): string {
  return currentLocale
}

export function setLocale(locale: string): boolean {
  if (!registry[locale]) return false
  if (currentLocale === locale) return true

  currentLocale = locale

  try {
    localStorage.setItem(STORAGE_KEY, locale)
  } catch {
    // 忽略异常
  }

  syncSingBoxLanguage(locale)

  for (const listener of listeners) {
    try {
      listener(currentLocale)
    } catch {
      // 保证单个监听器失败不打断其他监听器
    }
  }

  return true
}

export function onLocaleChange(listener: LocaleListener): () => void {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

// 递归查找点分路径属性
function lookupPath(dict: LocaleDictionary | undefined, path: string): unknown {
  if (!dict) return undefined
  const parts = path.split('.')
  let current: unknown = dict
  for (const part of parts) {
    if (typeof current !== 'object' || current === null) return undefined
    current = (current as Record<string, unknown>)[part]
  }
  return current
}

// 参数插值格式化：{name} -> params[name]
function interpolate(template: string, params?: Record<string, unknown>): string {
  if (!params) return template
  return template.replace(/\{(\w+)\}/g, (match, key) => {
    return key in params ? String(params[key]) : match
  })
}

// 核心翻译函数，遵循回退链：current -> en-US -> zh-CN -> path
export function t(path: string, params?: Record<string, unknown>): string {
  const localesToTry = [currentLocale]
  if (currentLocale !== 'en-US') localesToTry.push('en-US')
  if (currentLocale !== 'zh-CN' && !localesToTry.includes('zh-CN')) localesToTry.push('zh-CN')

  for (const code of localesToTry) {
    const dict = registry[code]?.dict
    const val = lookupPath(dict, path)
    if (typeof val === 'string') {
      return interpolate(val, params)
    }
  }

  return interpolate(path, params)
}

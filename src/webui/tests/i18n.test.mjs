import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  t,
  getLocale,
  setLocale,
  onLocaleChange,
  detectLocale,
  registerLocale,
  getSupportedLocales,
  STORAGE_KEY,
  SING_BOX_STORAGE_KEY
} from '../src/i18n.ts'
import { COMMANDS } from '../src/commands.ts'
import { getHelp } from '../src/help.ts'

// 模拟简易内存 localStorage 与 navigator
class MockLocalStorage {
  constructor() {
    this.store = new Map()
  }
  getItem(key) {
    return this.store.get(key) ?? null
  }
  setItem(key, value) {
    this.store.set(key, String(value))
  }
  removeItem(key) {
    this.store.delete(key)
  }
  clear() {
    this.store.clear()
  }
}

globalThis.localStorage = new MockLocalStorage()

test('i18n 元数据支持与 SUPPORTED_LOCALES 列表完整性', () => {
  const supported = getSupportedLocales()
  const codes = supported.map(m => m.code)
  assert.ok(codes.includes('zh-CN'))
  assert.ok(codes.includes('en-US'))
  assert.ok(codes.includes('ru-RU'))

  const ru = supported.find(m => m.code === 'ru-RU')
  assert.equal(ru?.nativeName, 'Русский')
  const en = supported.find(m => m.code === 'en-US')
  assert.equal(en?.nativeName, 'English')
  const zh = supported.find(m => m.code === 'zh-CN')
  assert.equal(zh?.nativeName, '简体中文')
})

test('i18n 多语言键值解析与参数插值正确性', () => {
  // zh-CN
  setLocale('zh-CN')
  assert.equal(t('common.service'), '服务')
  assert.equal(t('states.ready'), '运行中')
  assert.equal(t('common.exit_code', { code: 0 }), '退出码: 0')
  assert.equal(t('commands.service.overview'), 'service <命令>  服务控制')

  // en-US
  setLocale('en-US')
  assert.equal(t('common.service'), 'Service')
  assert.equal(t('states.ready'), 'Running')
  assert.equal(t('common.exit_code', { code: 1 }), 'Exit code: 1')
  assert.equal(t('commands.service.overview'), 'service <command>  Service control')

  // ru-RU（严格校验俄语字母 ё）
  setLocale('ru-RU')
  assert.equal(t('common.service'), 'Сервис')
  assert.equal(t('states.ready'), 'Работает')
  assert.equal(t('common.exit_code', { code: 127 }), 'Код завершения: 127')
  assert.equal(t('commands.service.overview'), 'service <команда>  Управление службой')
  assert.ok(t('commands.app.help').includes('чёрного'))
})

test('i18n 回退链条：当前语言 -> en-US -> zh-CN -> path', () => {
  // 注册仅包含部分键值的测试语言
  registerLocale(
    { code: 'test-FR', name: 'French (Test)', nativeName: 'Français' },
    {
      common: {
        service: 'Service FR'
      }
      // states.ready 缺失，应自动回退到 en-US ('Running')
    },
    'fr'
  )

  setLocale('test-FR')
  // 存在于 test-FR
  assert.equal(t('common.service'), 'Service FR')
  // 不存在于 test-FR -> 回退至 en-US
  assert.equal(t('states.ready'), 'Running')

  // 注册第二个测试语言，在 en-US 中亦缺失
  registerLocale(
    { code: 'test-XX', name: 'Unknown (Test)', nativeName: 'Test' },
    {},
    'xx'
  )
  setLocale('test-XX')
  // 不存在的键值 -> 返回带参数替换的原始路径
  assert.equal(t('totally.unknown.key'), 'totally.unknown.key')
  assert.equal(t('unknown.with.{param}', { param: 'test' }), 'unknown.with.test')
})

test('i18n 本地存储同步 (netproxy.language 与 sing-box-dashboard.language)', () => {
  setLocale('zh-CN')
  assert.equal(globalThis.localStorage.getItem(STORAGE_KEY), 'zh-CN')
  assert.equal(globalThis.localStorage.getItem(SING_BOX_STORAGE_KEY), 'zh-Hans')

  setLocale('en-US')
  assert.equal(globalThis.localStorage.getItem(STORAGE_KEY), 'en-US')
  assert.equal(globalThis.localStorage.getItem(SING_BOX_STORAGE_KEY), 'en')

  setLocale('ru-RU')
  assert.equal(globalThis.localStorage.getItem(STORAGE_KEY), 'ru-RU')
  assert.equal(globalThis.localStorage.getItem(SING_BOX_STORAGE_KEY), 'ru')
})

test('i18n 基于 navigator.languages 与持久化值的语言自动探测', () => {
  globalThis.localStorage.clear()

  // 1. 基于 localStorage 持久化值探测
  globalThis.localStorage.setItem(STORAGE_KEY, 'ru-RU')
  assert.equal(detectLocale(), 'ru-RU')

  globalThis.localStorage.clear()

  // 2. 基于 navigator.languages 探测
  const originalDescriptor = Object.getOwnPropertyDescriptor(globalThis, 'navigator')

  function mockNavigator(languages, language) {
    Object.defineProperty(globalThis, 'navigator', {
      value: { languages, language },
      configurable: true,
      writable: true
    })
  }

  try {
    // 俄语
    mockNavigator(['ru', 'en-US'], 'ru')
    assert.equal(detectLocale(), 'ru-RU')

    // 英语
    mockNavigator(['en-GB', 'en'], 'en-GB')
    assert.equal(detectLocale(), 'en-US')

    // 中文
    mockNavigator(['zh-CN', 'zh'], 'zh-CN')
    assert.equal(detectLocale(), 'zh-CN')

    // 未知语言 -> 回退至 en-US
    mockNavigator(['de-DE', 'de'], 'de-DE')
    assert.equal(detectLocale(), 'en-US')
  } finally {
    if (originalDescriptor) {
      Object.defineProperty(globalThis, 'navigator', originalDescriptor)
    }
  }
})

test('i18n 响应式 onLocaleChange 订阅与取消订阅', () => {
  const events = []
  const unsubscribe = onLocaleChange(locale => events.push(locale))

  setLocale('en-US')
  setLocale('ru-RU')
  setLocale('ru-RU') // 重复设置相同语言不应触发多余事件

  assert.deepEqual(events, ['en-US', 'ru-RU'])

  unsubscribe()
  setLocale('zh-CN')
  assert.deepEqual(events, ['en-US', 'ru-RU']) // 无额外事件派发
})

test('i18n 语言切换时 COMMANDS 与 getHelp() 动态更新', () => {
  setLocale('zh-CN')
  assert.ok(COMMANDS.service.overview.includes('服务控制'))
  assert.ok(getHelp().includes('命令:'))

  setLocale('en-US')
  assert.ok(COMMANDS.service.overview.includes('Service control'))
  assert.ok(getHelp().includes('Commands:'))

  setLocale('ru-RU')
  assert.ok(COMMANDS.service.overview.includes('Управление службой'))
  assert.ok(getHelp().includes('Команды:'))
  assert.ok(getHelp('all').includes('Управление журналами'))
})

import { ctl, ctlJson, shell, inKsu, completions as fetchCompletions } from './exec.ts'
import { complete, replaceCompletion } from './autocomplete.ts'
import { parseCommandLine } from './command.ts'
import { decodeCtlResult } from './contract.ts'
import { formatCtlOutput } from './format.ts'
import { getHelp } from './help.ts'
import { createPoller } from './polling.ts'
import { t, getLocale, setLocale, getSupportedLocales, onLocaleChange } from './i18n.ts'
import './style.css'

const STATE_MAP: Record<string, { key: string; color: string }> = {
  ready: { key: 'states.ready', color: 'var(--good)' },
  stopped: { key: 'states.stopped', color: 'var(--secondary)' },
  failed: { key: 'states.failed', color: 'var(--danger)' },
  starting: { key: 'states.starting', color: 'var(--medium)' },
  stopping: { key: 'states.stopping', color: 'var(--medium)' },
  preparing: { key: 'states.preparing', color: 'var(--medium)' },
}

function byId<T extends HTMLElement>(id: string): T {
  const element = document.getElementById(id)
  if (!element) throw new Error(t('common.missing_element', { id }))
  return element as T
}

const output = byId<HTMLElement>('output')
const welcome = byId<HTMLElement>('welcome')
const form = byId<HTMLFormElement>('command-form')
const input = byId<HTMLInputElement>('command-input')
const suggestions = byId<HTMLElement>('suggestions')
const completeButton = byId<HTMLButtonElement>('complete')
const previousButton = byId<HTMLButtonElement>('history-prev')
const nextButton = byId<HTMLButtonElement>('history-next')
const runButton = byId<HTMLButtonElement>('run')
const copyButton = byId<HTMLButtonElement>('copy')
const clearButton = byId<HTMLButtonElement>('clear')
const latestButton = byId<HTMLButtonElement>('latest')
const serviceStatus = byId<HTMLButtonElement>('service-status')
const serviceState = byId<HTMLElement>('service-state')
const announcement = byId<HTMLElement>('announcement')
const entryTemplate = byId<HTMLTemplateElement>('command-entry')
const langSelect = byId<HTMLSelectElement>('lang-select')
const environment = byId<HTMLElement>('environment')

const history: string[] = []
let historyIndex = -1
let draft = ''
let busy = false
let composing = false
let followOutput = true
let scrollFrame = 0
let lastResult = ''
let knownGroups: string[] = []
let knownSubscriptions: string[] = []
let completionRevision = 0

let lastServiceStateRaw: string | undefined
let lastServiceOk = false
let lastServiceMessage = ''

function announce(message: string) {
  announcement.textContent = message
}

function updateControls() {
  runButton.disabled = busy || !input.value.trim()
  completeButton.disabled = busy
  previousButton.disabled = busy || !history.length || historyIndex === 0
  nextButton.disabled = busy || historyIndex === -1
  clearButton.disabled = busy || !output.querySelector('.entry')
  copyButton.disabled = !lastResult
  serviceStatus.disabled = busy
}

function scrollToLatest(force = false) {
  if (force) followOutput = true
  if (scrollFrame) cancelAnimationFrame(scrollFrame)
  scrollFrame = requestAnimationFrame(() => {
    scrollFrame = 0
    if (followOutput) output.scrollTop = output.scrollHeight
    latestButton.hidden = output.scrollHeight - output.clientHeight - output.scrollTop < 24
  })
}

function closeSuggestions() {
  suggestions.hidden = true
  suggestions.replaceChildren()
}

function focusInput() {
  input.focus({ preventScroll: true })
  input.setSelectionRange(input.value.length, input.value.length)
}

function setInput(value: string) {
  input.value = value
  historyIndex = -1
  closeSuggestions()
  updateControls()
  focusInput()
}

async function refreshCompletions() {
  const revision = ++completionRevision
  try {
    const result = await fetchCompletions()
    if (revision !== completionRevision) return
    knownGroups = result.groups
    knownSubscriptions = result.subs
  } catch {
    // 补全失败不影响终端命令执行。
  }
}

function renderServiceState() {
  if (lastServiceStateRaw === undefined && !lastServiceOk && !lastServiceMessage) {
    serviceState.textContent = t('states.detecting')
    serviceStatus.style.setProperty('--state-color', 'var(--secondary)')
    serviceStatus.title = `${t('states.detecting')} · ${t('common.status_click_hint')}`
    return
  }
  const stateMeta = lastServiceStateRaw ? STATE_MAP[lastServiceStateRaw] : undefined
  const label = stateMeta ? t(stateMeta.key) : (lastServiceStateRaw ? (t(`states.${lastServiceStateRaw}`) || lastServiceStateRaw) : t('common.status_unavailable'))
  serviceState.textContent = label
  serviceStatus.style.setProperty('--state-color', stateMeta?.color || (lastServiceOk ? 'var(--good)' : 'var(--danger)'))
  serviceStatus.title = `${label} · ${t('common.status_click_hint')}${lastServiceOk ? '' : '：' + lastServiceMessage}`
}

const statusPoller = createPoller(
  () => ctlJson<{ state?: string }>(['service', 'status']),
  result => {
    lastServiceOk = result.ok
    lastServiceStateRaw = result.ok ? (result.data?.state || '') : undefined
    lastServiceMessage = result.message || ''
    renderServiceState()
  },
)

function clearOutput() {
  output.replaceChildren(welcome)
  lastResult = ''
  copyButton.textContent = t('common.copy')
  latestButton.hidden = true
  followOutput = true
  closeSuggestions()
  updateControls()
  announce(t('common.announced_cleared'))
}

function append(entry: HTMLElement, kind: 'o' | 'e' | 'help', text: string) {
  if (!text) return
  entry.append(Object.assign(document.createElement('pre'), { className: kind, textContent: text }))
}

async function run(raw: string) {
  const command = raw.trim()
  if (!command || busy || composing) return
  if (history[history.length - 1] !== command) history.push(command)
  if (history.length > 100) history.shift()
  historyIndex = -1
  input.value = ''
  draft = ''
  closeSuggestions()
  if (command === 'clear') {
    clearOutput()
    return
  }

  const entry = entryTemplate.content.firstElementChild!.cloneNode(true) as HTMLElement
  entry.querySelector('pre')!.textContent = '❯ ' + command
  const state = entry.querySelector<HTMLElement>('.entry-state')!
  output.append(entry)
  // 只在当前页面保留有限记录，不持久化可能含凭据的输入和输出。
  const entries = output.querySelectorAll('.entry')
  if (entries.length > 100) entries[0].remove()
  busy = true
  output.setAttribute('aria-busy', 'true')
  statusPoller.setActive(false)
  updateControls()
  scrollToLatest(true)
  announce(t('common.announced_running'))

  let failed = false
  let resultText = ''
  try {
    let out = ''
    let err = ''
    let code = 0
    let kind: 'o' | 'help' = 'o'
    if (command === 'exit') {
      err = t('common.exit_hint')
      code = 1
    } else if (command === 'help' || command.startsWith('help ')) {
      out = getHelp(command.slice(4).trim())
      kind = 'help'
    } else if (command.startsWith('!')) {
      const value = command.slice(1).trim()
      if (value) ({ out, err, code } = await shell(value))
      else { err = t('common.shell_empty'); code = 1 }
    } else {
      const args = parseCommandLine(command)
      const result = await ctl(args)
      ;({ out, err, code } = result)
      // 命令仍显示原有结果，但成功标记必须同时满足 JSON 契约与退出码。
      const decoded = decodeCtlResult(result)
      failed = !decoded.ok
      if (failed && !err && (decoded.code.startsWith('transport.') || !out)) err = decoded.message
      if (!args.includes('--raw')) out = formatCtlOutput(out)
      if (['service', 'sub', 'node', 'catalog'].includes(args[0])) void refreshCompletions()
    }
    failed ||= code !== 0
    append(entry, kind, out)
    append(entry, 'e', err)
    resultText = [out, err].filter(Boolean).join('\n')
  } catch (error) {
    failed = true
    resultText = t('common.exception', { message: error instanceof Error ? error.message : String(error) })
    append(entry, 'e', resultText)
  } finally {
    lastResult = resultText
    copyButton.textContent = t('common.copy')
    entry.dataset.state = failed ? 'error' : 'success'
    state.textContent = failed ? t('common.failed') : t('common.success')
    busy = false
    output.setAttribute('aria-busy', 'false')
    updateControls()
    scrollToLatest()
    announce(failed ? t('common.announced_failed') : t('common.announced_success'))
    statusPoller.setActive(!document.hidden)
  }
}

function completeInput(): boolean {
  if (busy || composing) return false
  const result = complete(input.value, knownGroups, knownSubscriptions)
  if (!result.candidates.length) { closeSuggestions(); return false }
  setInput(result.completed)
  if (result.candidates.length > 1) {
    for (const candidate of result.candidates) {
      const button = Object.assign(document.createElement('button'), { type: 'button', textContent: candidate })
      button.dataset.completion = candidate
      suggestions.append(button)
    }
    suggestions.hidden = false
    announce(t('common.announced_candidates', { count: result.candidates.length }))
  }
  return true
}

function moveHistory(direction: -1 | 1) {
  if (busy || !history.length || (direction === 1 && historyIndex === -1)) return
  if (historyIndex === -1) { draft = input.value; historyIndex = history.length }
  historyIndex = Math.max(0, historyIndex + direction)
  if (historyIndex >= history.length) { historyIndex = -1; input.value = draft }
  else input.value = history[historyIndex]
  closeSuggestions()
  updateControls()
  focusInput()
}

function renderStaticTexts() {
  environment.textContent = inKsu ? t('common.env_ksu') : t('common.env_preview')
  environment.title = inKsu ? t('common.env_ksu') : t('common.env_preview_desc')
  input.placeholder = t('common.input_placeholder')
  completeButton.title = t('common.complete_hint')
  previousButton.title = t('common.history_prev_hint')
  previousButton.setAttribute('aria-label', t('common.history_prev_hint'))
  nextButton.title = t('common.history_next_hint')
  nextButton.setAttribute('aria-label', t('common.history_next_hint'))
  runButton.title = t('common.run_hint')
  runButton.setAttribute('aria-label', t('common.run_hint'))
  copyButton.textContent = t('common.copy')
  clearButton.textContent = t('common.clear')
  latestButton.textContent = t('common.scroll_latest')

  const welcomeH2 = welcome.querySelector('h2')
  if (welcomeH2) welcomeH2.textContent = t('common.welcome_title')
  const welcomeP = welcome.querySelector('.welcome-hint')
  if (welcomeP) welcomeP.innerHTML = t('common.welcome_hint')
  const scStatus = welcome.querySelector('[data-command="service status"] span')
  if (scStatus) scStatus.textContent = t('common.shortcut_status')
  const scNode = welcome.querySelector('[data-command="node current"] span')
  if (scNode) scNode.textContent = t('common.shortcut_node')
  const scCatalog = welcome.querySelector('[data-command="catalog list"] span')
  if (scCatalog) scCatalog.textContent = t('common.shortcut_catalog')
  const scHelp = welcome.querySelector('[data-command="help"] span')
  if (scHelp) scHelp.textContent = t('common.shortcut_help')

  renderServiceState()
}

function initLangSelect() {
  langSelect.innerHTML = ''
  for (const meta of getSupportedLocales()) {
    const opt = document.createElement('option')
    opt.value = meta.code
    opt.textContent = meta.nativeName
    langSelect.append(opt)
  }
  langSelect.value = getLocale()
  langSelect.addEventListener('change', () => {
    setLocale(langSelect.value)
  })
}

onLocaleChange(newLocale => {
  langSelect.value = newLocale
  renderStaticTexts()
})

form.addEventListener('submit', event => { event.preventDefault(); void run(input.value) })
input.addEventListener('compositionstart', () => { composing = true })
input.addEventListener('compositionend', () => { composing = false; updateControls() })
input.addEventListener('input', () => {
  historyIndex = -1
  closeSuggestions()
  updateControls()
})
input.addEventListener('keydown', event => {
  if (event.isComposing || composing || event.keyCode === 229) return
  if (event.key === 'Tab' && !event.shiftKey && completeInput()) event.preventDefault()
  else if (event.key === 'ArrowDown' && !suggestions.hidden) {
    event.preventDefault()
    suggestions.querySelector<HTMLButtonElement>('button')?.focus()
  } else if (event.key === 'ArrowUp' || event.key === 'ArrowDown') {
    event.preventDefault()
    moveHistory(event.key === 'ArrowUp' ? -1 : 1)
  } else if (event.key === 'Enter') {
    event.preventDefault()
    if (!event.repeat) void run(input.value)
  } else if (event.key === 'l' && event.ctrlKey && !busy) {
    event.preventDefault()
    clearOutput()
  }
})
suggestions.addEventListener('keydown', event => {
  if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return
  event.preventDefault()
  const buttons = Array.from(suggestions.querySelectorAll('button'))
  const index = buttons.indexOf(document.activeElement as HTMLButtonElement)
  const next = index + (event.key === 'ArrowDown' ? 1 : -1)
  if (next < 0) focusInput()
  else buttons[Math.min(next, buttons.length - 1)]?.focus()
})
document.addEventListener('keydown', event => {
  if (event.key === 'Escape' && !suggestions.hidden) { closeSuggestions(); focusInput() }
})
document.addEventListener('pointerdown', event => {
  if (!form.contains(event.target as Node)) closeSuggestions()
})
document.addEventListener('click', event => {
  const button = (event.target as Element).closest<HTMLButtonElement>('[data-command], [data-completion]')
  if (!button || busy) return
  setInput(button.dataset.command ?? replaceCompletion(input.value, button.dataset.completion!))
})
completeButton.addEventListener('click', completeInput)
previousButton.addEventListener('click', () => moveHistory(-1))
nextButton.addEventListener('click', () => moveHistory(1))
serviceStatus.addEventListener('click', () => { void run('service status') })
clearButton.addEventListener('click', clearOutput)
copyButton.addEventListener('click', async () => {
  const value = lastResult
  let copied = false
  try {
    await navigator.clipboard.writeText(value)
    copied = true
  } catch {}
  if (value !== lastResult) return
  copyButton.textContent = copied ? t('common.copied') : t('common.copy_failed')
  announce(copied ? t('common.announced_copied') : t('common.announced_copy_fail'))
})
output.addEventListener('scroll', () => {
  followOutput = output.scrollHeight - output.clientHeight - output.scrollTop < 24
  latestButton.hidden = followOutput
}, { passive: true })
latestButton.addEventListener('click', () => scrollToLatest(true))
const resizeObserver = new ResizeObserver(() => scrollToLatest())
resizeObserver.observe(output)
document.addEventListener('visibilitychange', () => statusPoller.setActive(!document.hidden && !busy))
window.addEventListener('pagehide', () => {
  statusPoller.setActive(false)
  if (scrollFrame) cancelAnimationFrame(scrollFrame)
  resizeObserver.disconnect()
})
window.addEventListener('pageshow', () => {
  resizeObserver.observe(output)
  statusPoller.setActive(!document.hidden && !busy)
})

initLangSelect()
renderStaticTexts()
byId('environment').hidden = !(import.meta.env.DEV && !inKsu)
void refreshCompletions()
updateControls()
statusPoller.setActive(!document.hidden)

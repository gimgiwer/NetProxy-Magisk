import { t } from './i18n.ts'

export interface CommandSpec {
  readonly overview: string
  readonly actions: readonly string[]
  readonly help: string
  getOverview?(): string
  getHelp?(): string
}

export const COMMANDS = {
  service: {
    get overview(): string { return t('commands.service.overview') },
    actions: ['status', 'start', 'stop', 'restart', 'reload', 'check', 'toggle'] as const,
    get help(): string { return t('commands.service.help') },
    getOverview(): string { return t('commands.service.overview') },
    getHelp(): string { return t('commands.service.help') }
  },
  catalog: {
    get overview(): string { return t('commands.catalog.overview') },
    actions: ['list', 'show'] as const,
    get help(): string { return t('commands.catalog.help') },
    getOverview(): string { return t('commands.catalog.overview') },
    getHelp(): string { return t('commands.catalog.help') }
  },
  node: {
    get overview(): string { return t('commands.node.overview') },
    actions: ['list', 'snapshot', 'current', 'show', 'get', 'export', 'delay', 'add', 'import', 'edit', 'remove', 'use'] as const,
    get help(): string { return t('commands.node.help') },
    getOverview(): string { return t('commands.node.overview') },
    getHelp(): string { return t('commands.node.help') }
  },
  sub: {
    get overview(): string { return t('commands.sub.overview') },
    actions: ['list', 'show', 'add', 'edit', 'update', 'update-all', 'activate', 'remove', 'history', 'cancel'] as const,
    get help(): string { return t('commands.sub.help') },
    getOverview(): string { return t('commands.sub.overview') },
    getHelp(): string { return t('commands.sub.help') }
  },
  mode: {
    get overview(): string { return t('commands.mode.overview') },
    actions: ['rule', 'global', 'direct', 'AllowAds'] as const,
    get help(): string { return t('commands.mode.help') },
    getOverview(): string { return t('commands.mode.overview') },
    getHelp(): string { return t('commands.mode.help') }
  },
  network: {
    get overview(): string { return t('commands.network.overview') },
    actions: ['evaluate'] as const,
    get help(): string { return t('commands.network.help') },
    getOverview(): string { return t('commands.network.overview') },
    getHelp(): string { return t('commands.network.help') }
  },
  app: {
    get overview(): string { return t('commands.app.overview') },
    actions: ['list', 'mode', 'add', 'remove', 'enable', 'disable'] as const,
    get help(): string { return t('commands.app.help') },
    getOverview(): string { return t('commands.app.overview') },
    getHelp(): string { return t('commands.app.help') }
  },
  ebpf: {
    get overview(): string { return t('commands.ebpf.overview') },
    actions: ['status'] as const,
    get help(): string { return t('commands.ebpf.help') },
    getOverview(): string { return t('commands.ebpf.overview') },
    getHelp(): string { return t('commands.ebpf.help') }
  },
  config: {
    get overview(): string { return t('commands.config.overview') },
    actions: ['list', 'read', 'check', 'validate', 'apply'] as const,
    get help(): string { return t('commands.config.help') },
    getOverview(): string { return t('commands.config.overview') },
    getHelp(): string { return t('commands.config.help') }
  },
  logs: {
    get overview(): string { return t('commands.logs.overview') },
    actions: ['show', 'clear', 'export'] as const,
    get help(): string { return t('commands.logs.help') },
    getOverview(): string { return t('commands.logs.overview') },
    getHelp(): string { return t('commands.logs.help') }
  },
} satisfies Record<string, CommandSpec>

export type CommandName = keyof typeof COMMANDS
export const COMMAND_NAMES = Object.keys(COMMANDS) as CommandName[]
export const ROOT_COMPLETIONS = [...COMMAND_NAMES, 'help', 'clear', 'exit']
export const HELP_TOPICS = [...COMMAND_NAMES, 'shell']

export const VALUE_COMPLETIONS: Record<string, string[]> = {
  'app mode': ['blacklist', 'whitelist'],
  'ebpf status': ['configured', 'all', 'local', 'shared', '--raw'],
  'logs show': ['service', 'core'],
  'logs clear': ['service', 'core'],
  'network evaluate': ['--type', '--ssid'],
  'network evaluate --type': ['wifi', 'not_wifi'],
}

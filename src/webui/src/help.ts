import { COMMAND_NAMES, COMMANDS, HELP_TOPICS, type CommandName } from './commands.ts'
import { t } from './i18n.ts'

export function getShellHelp(): string {
  return `${t('help.shell_title')}

${t('help.cmd_shell')}

${t('help.shell_desc')}
`
}

export function getMainHelp(): string {
  const commandsBlock = COMMAND_NAMES.map(name => `  ${COMMANDS[name].overview}`).join('\n')
  return `
${t('help.commands_label')}
${commandsBlock}

${t('help.global_options_label')}
${t('help.opt_json')}
${t('help.opt_timeout')}

${t('help.local_commands_label')}
${t('help.cmd_help')}
${t('help.cmd_clear')}
${t('help.cmd_exit')}
${t('help.cmd_shell')}

${t('help.examples_label')}
  service start
  node use auto default
  mode rule
  sub update-all
  node delay auto default

${t('help.help_all_hint')}
`
}

function topicHelp(topic: string): string | undefined {
  if (topic === 'shell') return getShellHelp()
  return COMMANDS[topic as CommandName]?.help
}

export function getHelp(topic?: string): string {
  const banner = t('help.banner')
  if (!topic) return banner + getMainHelp()
  const normalized = topic.toLowerCase()
  if (normalized === 'all') {
    return banner + getMainHelp() + '\n' + [...COMMAND_NAMES.map(name => COMMANDS[name].help), getShellHelp()].join('\n')
  }
  return topicHelp(normalized) || t('common.unknown_topic', { topic, topics: HELP_TOPICS.join(', ') })
}

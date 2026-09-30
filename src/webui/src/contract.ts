import { t } from './i18n.ts'

export const CONTRACT_SCHEMA = 1

export function decodeCtlResult<T>(r: ExecResult): CtlResult<T> {
  const payload = r.out.trim()

  if (payload) {
    try {
      const result = JSON.parse(payload) as Partial<CtlResult<T>>
      if (result.schema === CONTRACT_SCHEMA && typeof result.ok === 'boolean' &&
        typeof result.code === 'string' && typeof result.message === 'string') {
        if (!result.ok || r.code === 0) return result as CtlResult<T>
        return {
          ...result as CtlResult<T>,
          ok: false,
          code: 'transport.failed',
          message: r.err.trim() || t('transport.failed_with_code', { code: r.code })
        }
      }
    } catch {
      // 下面统一返回结构化的传输错误。
    }
  }

  if (r.code !== 0) {
    return {
      schema: CONTRACT_SCHEMA,
      ok: false,
      code: 'transport.failed',
      message: r.err.trim() || t('transport.failed_with_code', { code: r.code })
    }
  }
  return {
    schema: CONTRACT_SCHEMA,
    ok: false,
    code: payload ? 'transport.invalid_json' : 'transport.empty',
    message: payload ? t('transport.invalid_json') : (r.err.trim() || t('transport.empty'))
  }
}

export interface CtlResult<T = unknown> {
  schema: number
  ok: boolean
  code: string
  message: string
  data?: T
}

export interface ExecResult {
  out: string
  err: string
  code: number
}

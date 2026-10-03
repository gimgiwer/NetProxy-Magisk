import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { checkAwgSchema } from './check-awg-schema.mjs'

const source = JSON.parse(readFileSync(new URL('../../src/android/app/src/main/assets/sing-box.schema.json', import.meta.url)))

test('校验真正的 AmneziaWG (AWG) 分支，缺失和错误结构均失败', () => {
  checkAwgSchema(source)
  for (const mutate of [
    schema => { delete schema.$defs.Endpoint },
    schema => { schema.$defs.Endpoint.oneOf = [] },
    schema => {
      const wg = schema.$defs.Endpoint.oneOf.find(branch => branch.properties?.type?.const === 'wireguard')
      delete wg.properties.jc
    },
    schema => {
      const wg = schema.$defs.Endpoint.oneOf.find(branch => branch.properties?.type?.const === 'wireguard')
      delete wg.properties.h1
    },
    schema => {
      const wg = schema.$defs.Endpoint.oneOf.find(branch => branch.properties?.type?.const === 'wireguard')
      delete wg.properties.i1
    },
    schema => {
      const wg = schema.$defs.Endpoint.oneOf.find(branch => branch.properties?.type?.const === 'wireguard')
      wg.properties.jc.type = 'string'
    },
  ]) {
    const schema = structuredClone(source)
    mutate(schema)
    assert.throws(() => checkAwgSchema(schema))
  }
})

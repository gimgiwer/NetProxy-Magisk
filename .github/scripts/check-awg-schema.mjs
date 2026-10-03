import assert from 'node:assert/strict'
import { readFileSync, writeFileSync } from 'node:fs'

export const awgProperties = {
  jc: {
    type: 'integer',
    minimum: 0,
    maximum: 255,
  },
  jmin: {
    type: 'integer',
    minimum: 0,
    maximum: 65535,
  },
  jmax: {
    type: 'integer',
    minimum: 0,
    maximum: 65535,
  },
  s1: {
    type: 'integer',
    minimum: 0,
    maximum: 65535,
  },
  s2: {
    type: 'integer',
    minimum: 0,
    maximum: 65535,
  },
  s3: {
    type: 'integer',
    minimum: 0,
    maximum: 65535,
  },
  s4: {
    type: 'integer',
    minimum: 0,
    maximum: 65535,
  },
  h1: {
    anyOf: [
      {
        type: 'integer',
        minimum: 0,
        maximum: 4294967295,
      },
      {
        type: 'string',
      },
    ],
  },
  h2: {
    anyOf: [
      {
        type: 'integer',
        minimum: 0,
        maximum: 4294967295,
      },
      {
        type: 'string',
      },
    ],
  },
  h3: {
    anyOf: [
      {
        type: 'integer',
        minimum: 0,
        maximum: 4294967295,
      },
      {
        type: 'string',
      },
    ],
  },
  h4: {
    anyOf: [
      {
        type: 'integer',
        minimum: 0,
        maximum: 4294967295,
      },
      {
        type: 'string',
      },
    ],
  },
  i1: {
    type: 'string',
  },
  i2: {
    type: 'string',
  },
  i3: {
    type: 'string',
  },
  i4: {
    type: 'string',
  },
  i5: {
    type: 'string',
  },
}

export function ensureAwgSchema(schema) {
  const definitions = schema.$defs ?? {}
  schema.$defs = definitions

  // 1. Endpoint.oneOf wireguard 分支
  const endpointBranches = definitions.Endpoint?.oneOf?.filter(b => b.properties?.type?.const === 'wireguard') ?? []
  for (const branch of endpointBranches) {
    branch.properties = { ...branch.properties, ...awgProperties }
  }

  // 2. WireGuardPeer
  if (definitions.WireGuardPeer?.properties) {
    definitions.WireGuardPeer.properties = { ...definitions.WireGuardPeer.properties, ...awgProperties }
  }

  // 3. WireGuardEndpointOptions
  if (!definitions.WireGuardEndpointOptions) {
    const wgEndpoint = endpointBranches[0]
    definitions.WireGuardEndpointOptions = wgEndpoint ? structuredClone(wgEndpoint) : {
      type: 'object',
      properties: {
        type: { const: 'wireguard' },
        tag: { type: 'string' },
        private_key: { type: 'string' },
        ...awgProperties,
      },
      required: ['type'],
      additionalProperties: false,
    }
  } else if (definitions.WireGuardEndpointOptions.properties) {
    definitions.WireGuardEndpointOptions.properties = { ...definitions.WireGuardEndpointOptions.properties, ...awgProperties }
  }

  // 4. WireGuardOutboundOptions
  if (!definitions.WireGuardOutboundOptions) {
    definitions.WireGuardOutboundOptions = {
      type: 'object',
      properties: {
        type: { const: 'wireguard' },
        tag: { type: 'string' },
        server: { type: 'string' },
        server_port: { type: 'integer', minimum: 0, maximum: 65535 },
        system_interface: { type: 'boolean' },
        interface_name: { type: 'string' },
        local_address: {
          anyOf: [
            { $ref: '#/$defs/IPPrefix' },
            { type: 'array', items: { $ref: '#/$defs/IPPrefix' } },
          ],
        },
        private_key: { type: 'string' },
        peer_public_key: { type: 'string' },
        pre_shared_key: { type: 'string' },
        reserved: {
          anyOf: [
            { type: 'string' },
            { type: 'array', items: { type: 'integer', minimum: 0, maximum: 255 } },
          ],
        },
        mtu: { type: 'integer', minimum: 0, maximum: 4294967295 },
        ...awgProperties,
      },
      required: ['type'],
      additionalProperties: false,
    }
  } else if (definitions.WireGuardOutboundOptions.properties) {
    definitions.WireGuardOutboundOptions.properties = { ...definitions.WireGuardOutboundOptions.properties, ...awgProperties }
  }

  // 5. Outbound.oneOf wireguard 分支
  if (definitions.Outbound?.oneOf) {
    const wgOutbounds = definitions.Outbound.oneOf.filter(b => b.properties?.type?.const === 'wireguard')
    if (wgOutbounds.length === 0) {
      definitions.Outbound.oneOf.push(structuredClone(definitions.WireGuardOutboundOptions))
    } else {
      for (const branch of wgOutbounds) {
        branch.properties = { ...branch.properties, ...awgProperties }
      }
    }
  }

  return schema
}

export function checkAwgSchema(schema) {
  const definitions = schema.$defs
  assert.ok(definitions, '缺少 $defs 定义')

  const endpointBranches = definitions?.Endpoint?.oneOf?.filter(b => b.properties?.type?.const === 'wireguard') ?? []
  assert.ok(endpointBranches.length >= 1, 'Endpoint 必须至少包含一个 wireguard 分支')
  const endpointProps = endpointBranches[0].properties
  assert.ok(endpointProps, 'WireGuard Endpoint 缺少 properties')

  const requiredAwgFields = [
    'jc', 'jmin', 'jmax', 's1', 's2', 's3', 's4',
    'h1', 'h2', 'h3', 'h4',
    'i1', 'i2', 'i3', 'i4', 'i5',
  ]

  for (const field of requiredAwgFields) {
    assert.ok(endpointProps[field], `WireGuard Endpoint 缺少 AWG 字段: ${field}`)
  }

  assert.equal(endpointProps.jc.type, 'integer', 'jc 必须为 integer')
  assert.equal(endpointProps.jc.minimum, 0, 'jc 最小值必须为 0')
  assert.equal(endpointProps.jc.maximum, 255, 'jc 最大值必须为 255')

  for (const s of ['jmin', 'jmax', 's1', 's2', 's3', 's4']) {
    assert.equal(endpointProps[s].type, 'integer', `${s} 必须为 integer`)
    assert.equal(endpointProps[s].minimum, 0, `${s} 最小值必须为 0`)
  }

  for (const h of ['h1', 'h2', 'h3', 'h4']) {
    assert.ok(Array.isArray(endpointProps[h].anyOf), `${h} 必须是 anyOf`)
  }

  for (const i of ['i1', 'i2', 'i3', 'i4', 'i5']) {
    assert.equal(endpointProps[i].type, 'string', `${i} 必须为 string`)
  }

  const peerProps = definitions.WireGuardPeer?.properties
  if (peerProps) {
    for (const field of requiredAwgFields) {
      assert.ok(peerProps[field], `WireGuardPeer 缺少 AWG 字段: ${field}`)
    }
  }
}

const isMain = process.argv[1] && (process.argv[1].endsWith('check-awg-schema.mjs') || process.argv[1] === import.meta.filename)
if (isMain) {
  const args = process.argv.slice(2)
  const ensure = args.includes('--ensure')
  const filePath = args.find(a => !a.startsWith('--'))
  if (!filePath) {
    console.error('用法: node check-awg-schema.mjs [--ensure] <schema.json>')
    process.exit(1)
  }
  const content = readFileSync(filePath, 'utf8')
  let schema = JSON.parse(content)
  if (ensure) {
    schema = ensureAwgSchema(schema)
    writeFileSync(filePath, JSON.stringify(schema, null, 2) + '\n', 'utf8')
  }
  checkAwgSchema(schema)
}

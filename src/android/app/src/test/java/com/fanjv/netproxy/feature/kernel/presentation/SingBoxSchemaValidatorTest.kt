package com.fanjv.netproxy.feature.kernel.presentation

import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

class SingBoxSchemaValidatorTest {
    private val validator = SingBoxSchemaValidator(TEST_SCHEMA)

    @Test
    fun validDocumentPassesDeclaredSchema() = runBlocking {
        val result = validator.validate(
            """{"${'$'}schema":"$SCHEMA_URI","name":"NetProxy"}""",
        )

        assertEquals(SingBoxSchemaValidationResult.Valid, result)
    }

    @Test
    fun invalidDocumentReturnsLocatedIssue() = runBlocking {
        val result = validator.validate(
            """{"${'$'}schema":"$SCHEMA_URI","name":7}""",
        )

        assertTrue(result is SingBoxSchemaValidationResult.Invalid)
        val issue = (result as SingBoxSchemaValidationResult.Invalid).issues.single()
        assertEquals("/name", issue.instancePath)
        assertEquals(1, issue.line)
        assertTrue((issue.column ?: 0) > 0)
    }

    @Test
    fun documentWithoutSchemaIsStillValidated() = runBlocking {
        val result = validator.validate("""{"name":"NetProxy"}""")

        assertEquals(SingBoxSchemaValidationResult.Valid, result)
    }

    @Test
    fun branchValidationReturnsTheSelectedProtocolFieldError() = runBlocking {
        val branchValidator = SingBoxSchemaValidator(BRANCH_SCHEMA)

        val result = branchValidator.validate(
            """
                {
                  "items": [
                    { "type": "one", "port": 0 }
                  ]
                }
            """.trimIndent(),
        )

        assertTrue(result is SingBoxSchemaValidationResult.Invalid)
        val issue = (result as SingBoxSchemaValidationResult.Invalid).issues.single()
        assertEquals("/items/0/port", issue.instancePath)
        assertTrue(issue.message.contains("不能小于"))
        assertEquals(3, issue.line)
    }

    @Test
    fun durationPatternRequiresUnitsAndAllowsCompoundValues() = runBlocking {
        val durationValidator = SingBoxSchemaValidator(DURATION_SCHEMA)

        assertEquals(
            SingBoxSchemaValidationResult.Valid,
            durationValidator.validate("""{"timeout":"300ms"}"""),
        )
        assertTrue(durationValidator.validate("""{"timeout":"300"}""") is SingBoxSchemaValidationResult.Invalid)
        assertEquals(
            SingBoxSchemaValidationResult.Valid,
            durationValidator.validate("""{"timeout":"1h30m"}"""),
        )
    }

    @Test
    fun bundledSchemaSelectsLogicalDnsRuleByDiscriminator() = runBlocking {
        val schemaFile = sequenceOf(
            File("src/main/assets/sing-box.schema.json"),
            File("app/src/main/assets/sing-box.schema.json"),
        ).first(File::isFile)
        val bundledValidator = SingBoxSchemaValidator(schemaFile.readText())

        assertEquals(
            SingBoxSchemaValidationResult.Valid,
            bundledValidator.validate(
                """
                    {
                      "dns": {
                        "rules": [
                          {
                            "type": "logical",
                            "mode": "and",
                            "rules": [],
                            "action": "route"
                          }
                        ]
                      }
                    }
                """.trimIndent(),
            ),
        )
        assertTrue(
            bundledValidator.validate(
                """
                    {
                      "dns": {
                        "rules": [
                          {
                            "type": "unsupported",
                            "mode": "and",
                            "rules": [],
                            "action": "route"
                          }
                        ]
                      }
                    }
                """.trimIndent(),
            ) is SingBoxSchemaValidationResult.Invalid,
        )
    }

    @Test
    fun allOfTracksEvaluatedPropertiesBeforeRejectingUnknownField() = runBlocking {
        val branchValidator = SingBoxSchemaValidator(BRANCH_SCHEMA)

        val result = branchValidator.validate(
            """
                {
                  "logical": {
                    "type": "logical",
                    "mode": "and",
                    "unexpected": true
                  }
                }
            """.trimIndent(),
        )

        assertTrue(result is SingBoxSchemaValidationResult.Invalid)
        val issues = (result as SingBoxSchemaValidationResult.Invalid).issues
        assertEquals(listOf("/logical/unexpected"), issues.map { it.instancePath })
        assertTrue(issues.single().message.contains("不允许字段"))
    }

    @Test
    fun malformedJsonReturnsTheParserLocation() = runBlocking {
        val result = validator.validate(
            """
                {
                  "name":
                }
            """.trimIndent(),
        )

        assertTrue(result is SingBoxSchemaValidationResult.Invalid)
        val issue = (result as SingBoxSchemaValidationResult.Invalid).issues.single()
        assertEquals("", issue.instancePath)
        assertTrue((issue.line ?: 0) > 0)
        assertTrue((issue.column ?: 0) > 0)
        assertTrue(issue.message.startsWith("JSON 语法错误："))
    }

    @Test
    fun additionalPropertiesRejectsUnknownFields() = runBlocking {
        val result = validator.validate(
            """{"${'$'}schema":"$SCHEMA_URI","name":"NetProxy","unknown":true}""",
        )

        assertTrue(result is SingBoxSchemaValidationResult.Invalid)
        val issue = (result as SingBoxSchemaValidationResult.Invalid).issues.single()
        assertEquals("/unknown", issue.instancePath)
        assertTrue(issue.message.contains("不允许字段"))
    }

    @Test
    fun anyOfAndPropertyNamesValidateTheirSelectedBranch() = runBlocking {
        val branchValidator = SingBoxSchemaValidator(BRANCH_SCHEMA)

        val result = branchValidator.validate(
            """
                {
                  "labels": [1],
                  "interfaces": { "rmnet0": "192.0.2.1" }
                }
            """.trimIndent(),
        )

        assertTrue(result is SingBoxSchemaValidationResult.Invalid)
        val issues = (result as SingBoxSchemaValidationResult.Invalid).issues
        assertEquals(
            setOf("/labels/0", "/interfaces/rmnet0"),
            issues.map(SingBoxSchemaIssue::instancePath).toSet(),
        )
    }

    @Test
    fun bundledSingBoxSchemaValidatesConfigFragment() = runBlocking {
        val schemaFile = sequenceOf(
            File("src/main/assets/sing-box.schema.json"),
            File("app/src/main/assets/sing-box.schema.json"),
        ).first(File::isFile)
        val bundledValidator = SingBoxSchemaValidator(schemaFile.readText())
        val result = bundledValidator.validate(
            """
                {
                  "${'$'}schema": "https://sing-box.sagernet.org/schema.json",
                  "log": { "level": "info" },
                  "inbounds": [
                    {
                      "type": "mixed",
                      "tag": "mixed-in",
                      "listen": "::",
                      "listen_port": 7080
                    }
                  ]
                }
            """.trimIndent(),
        )

        assertEquals(SingBoxSchemaValidationResult.Valid, result)
    }

    @Test
    fun bundledSchemaScopesEbpfPrivateAddressToDataPlanes() = runBlocking {
        val schemaFile = sequenceOf(
            File("src/main/assets/sing-box.schema.json"),
            File("app/src/main/assets/sing-box.schema.json"),
        ).first(File::isFile)
        val bundledValidator = SingBoxSchemaValidator(schemaFile.readText())

        assertEquals(
            SingBoxSchemaValidationResult.Valid,
            bundledValidator.validate(
                """
                    {
                      "inbounds": [
                        {
                          "type": "ebpf",
                          "tc_priority": 1,
                          "local": {
                            "enabled": true,
                            "data_plane": "cgroup",
                            "dns_mode": "respect_policy",
                            "ipv6": true,
                            "bypass_private_address": true,
                            "bypass_port": [53, 853],
                            "bypass_port_range": ["8000:8080"]
                          },
                          "shared": {
                            "enabled": true,
                            "data_plane": "packet_rewrite",
                            "dns_mode": "off",
                            "interface": ["wlan2"],
                            "ipv6": false,
                            "bypass_private_address": false,
                            "bypass_port": [67, 68],
                            "bypass_port_range": ["10000:10100"]
                          }
                        }
                      ]
                    }
                """.trimIndent(),
            ),
        )
        assertTrue(
            bundledValidator.validate(
                """
                    {
                      "inbounds": [
                        {
                          "type": "ebpf",
                          "bypass_private_address": true
                        }
                      ]
                    }
                """.trimIndent(),
            ) is SingBoxSchemaValidationResult.Invalid,
        )
        assertTrue(
            bundledValidator.validate(
                """
                    {
                      "inbounds": [
                        {
                          "type": "ebpf",
                          "mode": "local",
                          "local": { "enabled": true, "data_plane": "cgroup" }
                        }
                      ]
                    }
                """.trimIndent(),
            ) is SingBoxSchemaValidationResult.Invalid,
        )
    }

    @Test
    fun bundledSchemaValidatesAmneziaWGEndpointAndRejectsInvalidFields() = runBlocking {
        val schemaFile = sequenceOf(
            File("src/main/assets/sing-box.schema.json"),
            File("app/src/main/assets/sing-box.schema.json"),
        ).first(File::isFile)
        val bundledValidator = SingBoxSchemaValidator(schemaFile.readText())

        // 验证合法的 AmneziaWG (AWG) 端点配置通过校验
        val validAwgConfig = """
            {
              "endpoints": [
                {
                  "type": "wireguard",
                  "tag": "awg-ep",
                  "private_key": "aW5pdGlhbF9wcml2YXRlX2tleV9leGFtcGxlCg==",
                  "address": ["10.0.0.2/32"],
                  "jc": 4,
                  "jmin": 40,
                  "jmax": 70,
                  "s1": 15,
                  "s2": 30,
                  "s3": 5,
                  "s4": 10,
                  "h1": 12345678,
                  "h2": "0x12345678",
                  "h3": 87654321,
                  "h4": "0x87654321",
                  "i1": "01020304",
                  "i2": "05060708",
                  "i3": "090a0b0c",
                  "i4": "0d0e0f10",
                  "i5": "11121314",
                  "peers": [
                    {
                      "address": "198.51.100.1",
                      "port": 51820,
                      "public_key": "cGVlcl9wdWJsaWNfa2V5X2V4YW1wbGVfaGVyZQ==",
                      "allowed_ips": ["0.0.0.0/0"],
                      "jc": 4,
                      "jmin": 40,
                      "jmax": 70,
                      "h1": 12345678
                    }
                  ]
                }
              ]
            }
        """.trimIndent()
        assertEquals(SingBoxSchemaValidationResult.Valid, bundledValidator.validate(validAwgConfig))

        // 验证 jc 超出 0-255 范围时被拦截拒绝
        val invalidJcConfig = """
            {
              "endpoints": [
                {
                  "type": "wireguard",
                  "tag": "awg-invalid",
                  "private_key": "aW5pdGlhbF9wcml2YXRlX2tleV9leGFtcGxlCg==",
                  "jc": 256
                }
              ]
            }
        """.trimIndent()
        assertTrue(bundledValidator.validate(invalidJcConfig) is SingBoxSchemaValidationResult.Invalid)

        // 验证 Outbound WireGuard 支持 AWG 混淆字段
        val validAwgOutbound = """
            {
              "outbounds": [
                {
                  "type": "wireguard",
                  "tag": "awg-out",
                  "server": "198.51.100.1",
                  "server_port": 51820,
                  "local_address": ["10.0.0.2/32"],
                  "private_key": "aW5pdGlhbF9wcml2YXRlX2tleV9leGFtcGxlCg==",
                  "peer_public_key": "cGVlcl9wdWJsaWNfa2V5X2V4YW1wbGVfaGVyZQ==",
                  "jc": 3,
                  "jmin": 20,
                  "jmax": 50,
                  "s1": 10,
                  "s2": 20,
                  "h1": 1
                }
              ]
            }
        """.trimIndent()
        assertEquals(SingBoxSchemaValidationResult.Valid, bundledValidator.validate(validAwgOutbound))
    }

    @Test
    fun bundledSchemaSupportsRef1ndExtensions() = runBlocking {
        val schemaFile = sequenceOf(
            File("src/main/assets/sing-box.schema.json"),
            File("app/src/main/assets/sing-box.schema.json"),
        ).first(File::isFile)
        val bundledValidator = SingBoxSchemaValidator(schemaFile.readText())
        val result = bundledValidator.validate(
            """
                {
                  "${'$'}schema": "$REF1ND_SCHEMA_URI",
                  "dns": {
                    "servers": [
                      {
                        "type": "group",
                        "tag": "dns-proxy",
                        "servers": ["cloudflare", "google"]
                      }
                    ]
                  },
                  "route": {
                    "rules": [
                      { "action": "sniff-override-destination" }
                    ]
                  },
                  "providers": []
                }
            """.trimIndent(),
        )

        assertEquals(SingBoxSchemaValidationResult.Valid, result)
    }

    @Test
    fun bundledSchemaOnlyUsesSupportedValidationKeywords() {
        val schemaFile = sequenceOf(
            File("src/main/assets/sing-box.schema.json"),
            File("app/src/main/assets/sing-box.schema.json"),
        ).first(File::isFile)
        val schema = singBoxSchemaJson.parseToJsonElement(schemaFile.readText()).jsonObject

        assertTrue((schema.validationKeywords() - SUPPORTED_SCHEMA_KEYWORDS).isEmpty())
    }

    @Test
    fun bundledSchemaValidatesAllStaticModuleDocuments() = runBlocking {
        val schemaFile = sequenceOf(
            File("src/main/assets/sing-box.schema.json"),
            File("app/src/main/assets/sing-box.schema.json"),
        ).first(File::isFile)
        val configFile = sequenceOf(
            File("../module/config/singbox/config.json"),
            File("../../module/config/singbox/config.json"),
            File("src/module/config/singbox/config.json"),
        ).first(File::isFile)
        val validator = SingBoxSchemaValidator(schemaFile.readText())

        val config = singBoxSchemaJson.parseToJsonElement(configFile.readText()).jsonObject
        val documents = listOf(config) + config.map { (key, value) -> JsonObject(mapOf(key to value)) }
        val failures = documents.mapNotNull { document ->
            (validator.validate(document.toString()) as? SingBoxSchemaValidationResult.Invalid)
                ?.issues?.joinToString { it.message }
        }

        assertTrue("静态配置校验失败：$failures", failures.isEmpty())
    }

    private companion object {
        const val SCHEMA_URI = "https://schemas.example.com/netproxy.json"
        const val REF1ND_SCHEMA_URI =
            "https://raw.githubusercontent.com/reF1nd/sing-box/reF1nd-testing/docs/schema.json"
        val TEST_SCHEMA = """
            {
              "${'$'}schema": "https://json-schema.org/draft/2020-12/schema",
              "type": "object",
              "properties": {
                "${'$'}schema": { "type": "string" },
                "name": { "type": "string" }
              },
              "required": ["name"],
              "additionalProperties": false
            }
        """.trimIndent()

        val DURATION_SCHEMA = """
            {
              "type": "object",
              "properties": {
                "timeout": {
                  "${'$'}ref": "#/${'$'}defs/Duration"
                }
              },
              "required": ["timeout"],
              "additionalProperties": false,
              "${'$'}defs": {
                "Duration": {
                  "type": "string",
                  "pattern": "^[-+]?(((\\d+(\\.\\d*)?|\\.\\d+)(ns|us|µs|μs|ms|s|m|h|d))+|0)$"
                }
              }
            }
        """.trimIndent()

        val BRANCH_SCHEMA = """
            {
              "type": "object",
              "properties": {
                "items": {
                  "type": "array",
                  "items": { "${'$'}ref": "#/${'$'}defs/Item" }
                },
                "logical": { "${'$'}ref": "#/${'$'}defs/Logical" },
                "labels": {
                  "anyOf": [
                    { "type": "string" },
                    { "type": "array", "items": { "type": "string" } }
                  ]
                },
                "interfaces": {
                  "type": "object",
                  "propertyNames": { "enum": ["wlan0"] },
                  "additionalProperties": { "type": "string" }
                }
              },
              "additionalProperties": false,
              "${'$'}defs": {
                "Item": {
                  "oneOf": [
                    {
                      "type": "object",
                      "properties": {
                        "type": { "const": "one" },
                        "port": { "type": "integer", "minimum": 1, "maximum": 65535 }
                      },
                      "required": ["type", "port"],
                      "additionalProperties": false
                    },
                    {
                      "type": "object",
                      "properties": {
                        "type": { "const": "two" },
                        "name": { "type": "string" }
                      },
                      "required": ["type"],
                      "additionalProperties": false
                    }
                  ]
                },
                "Logical": {
                  "type": "object",
                  "unevaluatedProperties": false,
                  "allOf": [
                    {
                      "properties": {
                        "type": { "const": "logical" }
                      },
                      "required": ["type"]
                    },
                    {
                      "properties": {
                        "mode": { "enum": ["and", "or"] }
                      },
                      "required": ["mode"]
                    }
                  ]
                }
              }
            }
        """.trimIndent()

    }
}

private val SUPPORTED_SCHEMA_KEYWORDS = setOf(
    "\$ref",
    "type",
    "properties",
    "required",
    "additionalProperties",
    "unevaluatedProperties",
    "items",
    "enum",
    "const",
    "oneOf",
    "anyOf",
    "allOf",
    "minimum",
    "maximum",
    "pattern",
    "propertyNames",
)

private fun JsonObject.validationKeywords(): Set<String> {
    val keywords = linkedSetOf<String>()

    fun JsonElement.collect() {
        val schema = this as? JsonObject ?: return
        keywords += schema.keys.intersect(SUPPORTED_SCHEMA_KEYWORDS + UNSUPPORTED_SCHEMA_KEYWORDS)
        schema["properties"].asSchemaObject()?.values?.forEach { it.collect() }
        schema["\$defs"].asSchemaObject()?.values?.forEach { it.collect() }
        schema["items"]?.collect()
        schema["propertyNames"]?.collect()
        schema["additionalProperties"]?.collect()
        schema["unevaluatedProperties"]?.collect()
        listOf("oneOf", "anyOf", "allOf").forEach { key ->
            (schema[key] as? JsonArray).orEmpty().forEach { it.collect() }
        }
    }

    collect()
    return keywords
}

private fun JsonElement?.asSchemaObject(): JsonObject? = this as? JsonObject

private val UNSUPPORTED_SCHEMA_KEYWORDS = setOf(
    "not",
    "if",
    "then",
    "else",
    "dependentRequired",
    "dependentSchemas",
    "patternProperties",
    "minProperties",
    "maxProperties",
    "prefixItems",
    "contains",
    "minContains",
    "maxContains",
    "minItems",
    "maxItems",
    "uniqueItems",
    "minLength",
    "maxLength",
    "multipleOf",
    "exclusiveMinimum",
    "exclusiveMaximum",
)

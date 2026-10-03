package com.fanjv.netproxy.feature.kernel.presentation

import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import top.yukonga.scripta.editor.completion.CompletionRequest
import top.yukonga.scripta.editor.text.TextPosition
import java.io.File

class SingBoxSchemaCompletionProviderTest {
    private val provider = SingBoxSchemaCompletionProvider(TEST_SCHEMA)

    @Test
    fun `property completion follows type discriminator`() = runBlocking {
        val document = """
            {
              "inbounds": [
                {
                  "type": "ebpf",
                  ""
                }
              ]
            }
        """.trimIndent()
        val caretOffset = document.indexOf("\"\"", document.indexOf("ebpf")) + 1

        val result = provider.complete(document.requestAt(caretOffset))
        val labels = requireNotNull(result).items.map { it.label }

        assertTrue("dns_mode" in labels)
        assertTrue("cgroup_enabled" in labels)
        assertTrue("shared_network" in labels)
        assertFalse("listen_port" in labels)
        assertFalse("type" in labels)
    }

    @Test
    fun `value completion returns enum values inside string`() = runBlocking {
        val document = """
            {
              "inbounds": [
                {
                  "type": "ebpf",
                  "dns_mode": ""
                }
              ]
            }
        """.trimIndent()
        val caretOffset = document.indexOf("\"\"", document.indexOf("dns_mode")) + 1

        val result = provider.complete(document.requestAt(caretOffset))

        val items = requireNotNull(result).items
        assertEquals(listOf("hijack", "off"), items.map { it.label })
        assertEquals("hijack", items.first().insertText)
    }

    @Test
    fun `explicit property completion inserts complete key prefix`() = runBlocking {
        val document = "{\n  \n}"
        val caretOffset = document.indexOf("  ") + 2

        val result = provider.complete(document.requestAt(caretOffset, explicit = true))
        val dns = requireNotNull(result).items.first { it.label == "dns" }

        assertEquals("\"dns\": ", dns.insertText)
    }

    @Test
    fun `array value completion offers discriminator snippets`() = runBlocking {
        val markedDocument = """
            {
              "inbounds": [
                <caret>
              ]
            }
        """.trimIndent()
        val caretOffset = markedDocument.indexOf("<caret>")
        val document = markedDocument.replace("<caret>", "")

        val result = provider.complete(document.requestAt(caretOffset, explicit = true))
        val ebpf = requireNotNull(result).items.first { it.label == "ebpf type" }

        assertEquals("{\"type\":\"ebpf\"}", ebpf.insertText)
        assertEquals("配置片段", ebpf.detail)
    }

    @Test
    fun `context help exposes json path and chinese field meaning`() {
        val document = """
            {
              "inbounds": [
                {
                  "type": "ebpf"
                }
              ]
            }
        """.trimIndent()
        val caretOffset = document.indexOf("ebpf") + 2

        val help = provider.contextHelp(document, document.positionAt(caretOffset))

        assertEquals("$.inbounds[0].type", requireNotNull(help).path)
        assertEquals("type", help.field)
        assertTrue(help.documentation.orEmpty().contains("配置对象的类型"))
    }

    @Test
    fun `wireguard endpoint completion offers amneziawg properties from bundled schema`() = runBlocking {
        val schemaFile = sequenceOf(
            File("src/main/assets/sing-box.schema.json"),
            File("app/src/main/assets/sing-box.schema.json"),
        ).first(File::isFile)
        val bundledProvider = SingBoxSchemaCompletionProvider(schemaFile.readText())

        val document = """
            {
              "endpoints": [
                {
                  "type": "wireguard",
                  ""
                }
              ]
            }
        """.trimIndent()
        val caretOffset = document.indexOf("\"\"", document.indexOf("wireguard")) + 1

        val result = bundledProvider.complete(document.requestAt(caretOffset))
        val labels = requireNotNull(result).items.map { it.label }

        // 验证 AmneziaWG 混淆参数均进入自动补全列表
        assertTrue("jc" in labels)
        assertTrue("jmin" in labels)
        assertTrue("jmax" in labels)
        assertTrue("s1" in labels)
        assertTrue("s2" in labels)
        assertTrue("s3" in labels)
        assertTrue("s4" in labels)
        assertTrue("h1" in labels)
        assertTrue("h2" in labels)
        assertTrue("h3" in labels)
        assertTrue("h4" in labels)
        assertTrue("i1" in labels)
        assertTrue("i2" in labels)
        assertTrue("i3" in labels)
        assertTrue("i4" in labels)
        assertTrue("i5" in labels)
    }

    @Test
    fun `context help exposes amneziawg field descriptions across locales`() {
        val zh = SingBoxSchemaStrings.ZH
        val en = SingBoxSchemaStrings.EN
        val ru = SingBoxSchemaStrings.RU

        // 验证中文描述正确
        assertTrue(requireNotNull(zh.fieldDocumentation("jc")).contains("AmneziaWG"))
        assertTrue(requireNotNull(zh.fieldDocumentation("jmin")).contains("垃圾数据包"))
        assertTrue(requireNotNull(zh.fieldDocumentation("h1")).contains("握手"))

        // 验证英文描述正确
        assertTrue(requireNotNull(en.fieldDocumentation("jc")).contains("AmneziaWG"))
        assertTrue(requireNotNull(en.fieldDocumentation("jmin")).contains("junk packet"))
        assertTrue(requireNotNull(en.fieldDocumentation("h1")).contains("handshake"))

        // 验证俄文描述正确，并严格检查字母 «ё» 与专业用语
        val ruJc = requireNotNull(ru.fieldDocumentation("jc"))
        val ruJmin = requireNotNull(ru.fieldDocumentation("jmin"))
        val ruH1 = requireNotNull(ru.fieldDocumentation("h1"))

        assertTrue(ruJc.contains("мусорных пакетов"))
        assertTrue(ruJmin.contains("мусорного пакета"))
        assertTrue(ruH1.contains("рукопожатия"))
        assertTrue(ru.localizedType("integer").contains("целое число"))
    }

    private fun String.requestAt(offset: Int, explicit: Boolean = false): CompletionRequest =
        CompletionRequest(
            text = this,
            caret = positionAt(offset),
            explicit = explicit,
        )

    private fun String.positionAt(offset: Int): TextPosition {
        val prefix = substring(0, offset)
        val line = prefix.count { it == '\n' }
        val lineStart = prefix.lastIndexOf('\n').let { if (it < 0) 0 else it + 1 }
        return TextPosition(line, offset - lineStart)
    }

    private companion object {
        val TEST_SCHEMA = """
            {
              "type": "object",
              "properties": {
                "dns": { "type": "object" },
                "inbounds": {
                  "type": "array",
                  "items": { "${'$'}ref": "#/${'$'}defs/Inbound" }
                }
              },
              "${'$'}defs": {
                "Inbound": {
                  "oneOf": [
                    {
                      "type": "object",
                      "properties": {
                        "type": { "const": "ebpf" },
                        "tag": { "type": "string" },
                        "cgroup_enabled": { "type": "boolean" },
                        "dns_mode": {
                          "type": "string",
                          "enum": ["hijack", "off"],
                          "default": "hijack"
                        },
                        "shared_network": { "type": "object" }
                      },
                      "required": ["type"]
                    },
                    {
                      "type": "object",
                      "properties": {
                        "type": { "const": "socks" },
                        "tag": { "type": "string" },
                        "listen_port": { "type": "integer" }
                      },
                      "required": ["type"]
                    }
                  ]
                }
              }
            }
        """.trimIndent()
    }
}



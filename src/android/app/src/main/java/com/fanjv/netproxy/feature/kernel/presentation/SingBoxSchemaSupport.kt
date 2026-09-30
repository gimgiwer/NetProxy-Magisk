package com.fanjv.netproxy.feature.kernel.presentation

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonPrimitive

/** sing-box Schema 与编辑器共用的 JSON 解析器。 */
internal val singBoxSchemaJson = Json { ignoreUnknownKeys = true }

/** JSON 文本中的一基行列位置，供补全和校验结果统一使用。 */
internal data class JsonSourcePosition(
    val line: Int,
    val column: Int,
)

/** 仅解析内置 Schema 的本地 JSON Pointer 引用，不允许访问网络。 */
internal class SingBoxSchemaReferenceResolver(
    private val root: JsonObject,
) {
    fun referencedSchema(
        schema: JsonObject,
        visitedRefs: Set<String>,
    ): Pair<String, JsonObject>? {
        val ref = schema["\$ref"]?.jsonPrimitive?.contentOrNull ?: return null
        if (ref in visitedRefs || !ref.startsWith("#/")) return null

        var target: JsonElement = root
        ref.removePrefix("#/").split('/').forEach { rawSegment ->
            val segment = rawSegment.replace("~1", "/").replace("~0", "~")
            target = (target as? JsonObject)?.get(segment) ?: return null
        }
        return ref to (target as? JsonObject ?: return null)
    }
}

/** 将字符偏移转换为一基行列位置。 */
internal fun sourcePositionAt(text: String, rawOffset: Int): JsonSourcePosition {
    val offset = rawOffset.coerceIn(0, text.length)
    var line = 1
    var lineStart = 0
    for (index in 0 until offset) {
        if (text[index] == '\n') {
            line++
            lineStart = index + 1
        }
    }
    return JsonSourcePosition(line = line, column = offset - lineStart + 1)
}

/**
 * 为有效 JSON 构建 JSON Pointer 到值起始位置的索引。
 *
 * `kotlinx.serialization` 负责语法解析；此扫描器只保留位置，避免为 Android
 * 引入完整 JSON Schema 引擎及其 Jackson 依赖。
 */
internal fun buildJsonSourceIndex(text: String): Map<String, JsonSourcePosition> =
    runCatching { JsonSourceIndexer(text).build() }.getOrDefault(emptyMap())

private class JsonSourceIndexer(
    private val text: String,
) {
    private val positions = linkedMapOf<String, JsonSourcePosition>()
    private var index = 0

    fun build(): Map<String, JsonSourcePosition> {
        skipWhitespace()
        parseValue("")
        return positions
    }

    private fun parseValue(path: String) {
        skipWhitespace()
        positions.putIfAbsent(path, sourcePositionAt(text, index))
        when (peek()) {
            '{' -> parseObject(path)
            '[' -> parseArray(path)
            '"' -> consumeString()
            else -> consumePrimitive()
        }
    }

    private fun parseObject(path: String) {
        index++
        skipWhitespace()
        if (consumeIf('}')) return

        while (index < text.length) {
            skipWhitespace()
            val key = consumeString()
            skipWhitespace()
            require(consumeIf(':')) { jsonSyntaxError("missing_colon") }
            parseValue("$path/${escapePointerSegment(key)}")
            skipWhitespace()
            if (consumeIf('}')) return
            require(consumeIf(',')) { jsonSyntaxError("missing_comma_prop") }
        }
        error(jsonSyntaxError("object_not_closed"))
    }

    private fun parseArray(path: String) {
        index++
        skipWhitespace()
        if (consumeIf(']')) return

        var itemIndex = 0
        while (index < text.length) {
            parseValue("$path/$itemIndex")
            itemIndex++
            skipWhitespace()
            if (consumeIf(']')) return
            require(consumeIf(',')) { jsonSyntaxError("missing_comma_elem") }
        }
        error(jsonSyntaxError("array_not_closed"))
    }

    private fun consumeString(): String {
        val start = index
        require(consumeIf('"')) { jsonSyntaxError("string_quote_start") }
        var escaped = false
        while (index < text.length) {
            val current = text[index++]
            if (!escaped && current == '"') {
                val literal = text.substring(start, index)
                return singBoxSchemaJson.parseToJsonElement(literal).jsonPrimitive.content
            }
            escaped = !escaped && current == '\\'
            if (current != '\\') escaped = false
        }
        error(jsonSyntaxError("string_not_closed"))
    }

    private fun consumePrimitive() {
        val start = index
        while (index < text.length && text[index] !in ",]}" && !text[index].isWhitespace()) {
            index++
        }
        require(index > start) { jsonSyntaxError("missing_json_val") }
    }

    private fun skipWhitespace() {
        while (index < text.length && text[index].isWhitespace()) index++
    }

    private fun peek(): Char? = text.getOrNull(index)

    private fun consumeIf(expected: Char): Boolean {
        if (peek() != expected) return false
        index++
        return true
    }
}

private fun escapePointerSegment(value: String): String =
    value.replace("~", "~0").replace("/", "~1")

private fun jsonSyntaxError(key: String): String {
    val lang = java.util.Locale.getDefault().language.lowercase()
    return when (key) {
        "missing_colon" -> when (lang) {
            "zh" -> "对象字段缺少冒号"
            "ru" -> "В поле объекта отсутствует двоеточие"
            else -> "Object field missing colon"
        }
        "missing_comma_prop" -> when (lang) {
            "zh" -> "对象字段缺少逗号"
            "ru" -> "В поле объекта отсутствует запятая"
            else -> "Object field missing comma"
        }
        "object_not_closed" -> when (lang) {
            "zh" -> "对象未闭合"
            "ru" -> "Объект не закрыт"
            else -> "Object not closed"
        }
        "missing_comma_elem" -> when (lang) {
            "zh" -> "数组元素缺少逗号"
            "ru" -> "В элементе массива отсутствует запятая"
            else -> "Array element missing comma"
        }
        "array_not_closed" -> when (lang) {
            "zh" -> "数组未闭合"
            "ru" -> "Массив не закрыт"
            else -> "Array not closed"
        }
        "string_quote_start" -> when (lang) {
            "zh" -> "字符串应以引号开始"
            "ru" -> "Строка должна начинаться с кавычки"
            else -> "String should start with quote"
        }
        "string_not_closed" -> when (lang) {
            "zh" -> "字符串未闭合"
            "ru" -> "Строка не закрыта"
            else -> "String not closed"
        }
        "missing_json_val" -> when (lang) {
            "zh" -> "缺少 JSON 值"
            "ru" -> "Отсутствует значение JSON"
            else -> "Missing JSON value"
        }
        else -> key
    }
}

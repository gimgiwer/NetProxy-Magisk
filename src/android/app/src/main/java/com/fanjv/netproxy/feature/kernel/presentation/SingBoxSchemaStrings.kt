package com.fanjv.netproxy.feature.kernel.presentation

import android.content.Context
import java.util.Locale

/**
 * sing-box schema 补全与校验多语言字典。
 * 在无 Context 的单元测试中默认使用 ZH。
 */
interface SingBoxSchemaStrings {
    val configSnippet: String
    val configHierarchyTooDeep: String
    val stringFormatMismatch: String
    val noMatchingType: String
    val multipleMatchingTypes: String
    val notMatchingAllowedFormat: String
    val cannotParseConfig: String
    val typeOr: String

    fun typeMismatch(allowedTypes: String, actualType: String): String
    fun valueMustBe(expected: String): String
    fun valueNotInAllowed(allowed: String): String
    fun numberCannotBeLessThan(minimum: Any): String
    fun numberCannotBeGreaterThan(maximum: Any): String
    fun missingRequiredField(name: String): String
    fun fieldNotAllowed(name: String): String
    fun jsonSyntaxError(message: String): String
    fun isTypeIssue(message: String): Boolean
    fun localizedType(type: String): String
    fun fieldDocumentation(field: String): String?

    object ZH : SingBoxSchemaStrings {
        override val configSnippet: String = "配置片段"
        override val configHierarchyTooDeep: String = "配置层级过深，无法继续校验"
        override val stringFormatMismatch: String = "字符串格式不符合要求"
        override val noMatchingType: String = "不匹配任何可用配置类型"
        override val multipleMatchingTypes: String = "同时匹配多个配置类型"
        override val notMatchingAllowedFormat: String = "不符合任何允许的配置格式"
        override val cannotParseConfig: String = "无法解析配置"
        override val typeOr: String = "或"

        override fun typeMismatch(allowedTypes: String, actualType: String): String =
            "应为$allowedTypes，实际为$actualType"

        override fun valueMustBe(expected: String): String = "值必须为 $expected"
        override fun valueNotInAllowed(allowed: String): String = "值不在允许范围内：$allowed"
        override fun numberCannotBeLessThan(minimum: Any): String = "数值不能小于 $minimum"
        override fun numberCannotBeGreaterThan(maximum: Any): String = "数值不能大于 $maximum"
        override fun missingRequiredField(name: String): String = "缺少必填字段 \"$name\""
        override fun fieldNotAllowed(name: String): String = "不允许字段 \"$name\""
        override fun jsonSyntaxError(message: String): String = "JSON 语法错误：$message"

        override fun isTypeIssue(message: String): Boolean =
            "配置类型" in message || "允许的配置格式" in message

        override fun localizedType(type: String): String = when (type) {
            "string" -> "字符串"
            "integer" -> "整数"
            "number" -> "数字"
            "boolean" -> "布尔值"
            "object" -> "对象"
            "array" -> "数组"
            "null" -> "空值"
            else -> type
        }

        override fun fieldDocumentation(field: String): String? = when (field) {
            "type" -> "配置对象的类型；选择后，补全列表会只显示该类型支持的字段。"
            "tag" -> "该对象的唯一名称，供其他配置通过标签引用。"
            "enabled" -> "控制当前功能是否启用。"
            "server" -> "远程服务器地址，可以是域名或 IP 地址。"
            "server_port" -> "远程服务器端口。"
            "listen" -> "本地监听地址。"
            "listen_port" -> "本地监听端口。"
            "outbound" -> "命中后使用的出站标签。"
            "default_domain_resolver" -> "解析服务器域名时使用的 DNS 服务器标签。"
            "rule_set" -> "匹配一个或多个规则集标签。"
            "rules" -> "按顺序匹配的规则列表。"
            "action" -> "规则命中后执行的动作。"
            "servers" -> "当前组包含的服务器或成员标签。"
            "url" -> "下载、健康检查或延迟测试使用的 URL。"
            "interval" -> "自动更新或测试的时间间隔，例如 3m、1h。"
            "path" -> "本地文件或资源路径。"
            "initial_path" -> "首次启动时使用的本地资源路径。"
            "http_client" -> "执行远程请求时使用的 HTTP Client 标签。"
            "secret" -> "访问控制接口时使用的鉴权密钥。"
            // AmneziaWG 混淆与 WireGuard 协议参数
            "private_key" -> "WireGuard / AmneziaWG 节点私钥。"
            "peers" -> "WireGuard / AmneziaWG 对端节点列表。"
            "allowed_ips" -> "允许通过此对端路由的 IP 地址段列表。"
            "pre_shared_key" -> "可选的预共享密钥（PSK），用于提供后量子安全性。"
            "persistent_keepalive_interval" -> "保持连接活跃的保活心跳发送间隔（秒）。"
            "reserved" -> "WireGuard 保留字节（例如用于 Warp 客户端识别）。"
            "jc" -> "AmneziaWG 握手前发送的垃圾数据包数量（0-255）。"
            "jmin" -> "AmneziaWG 垃圾数据包的最小载荷字节长度。"
            "jmax" -> "AmneziaWG 垃圾数据包的最大载荷字节长度。"
            "s1" -> "AmneziaWG 握手请求包（Init）填充字节长度。"
            "s2" -> "AmneziaWG 握手响应包（Response）填充字节长度。"
            "s3" -> "AmneziaWG Cookie 消息包填充字节长度。"
            "s4" -> "AmneziaWG 数据传输包（Data）填充字节长度。"
            "h1" -> "AmneziaWG 握手请求包（Init）自定义消息头魔数。"
            "h2" -> "AmneziaWG 握手响应包（Response）自定义消息头魔数。"
            "h3" -> "AmneziaWG Cookie 消息包自定义消息头魔数。"
            "h4" -> "AmneziaWG 数据传输包（Data）自定义消息头魔数。"
            "i1" -> "AmneziaWG 自定义垃圾数据包 1 载荷内容（十六进制或 Base64）。"
            "i2" -> "AmneziaWG 自定义垃圾数据包 2 载荷内容（十六进制或 Base64）。"
            "i3" -> "AmneziaWG 自定义垃圾数据包 3 载荷内容（十六进制或 Base64）。"
            "i4" -> "AmneziaWG 自定义垃圾数据包 4 载荷内容（十六进制或 Base64）。"
            "i5" -> "AmneziaWG 自定义垃圾数据包 5 载荷内容（十六进制或 Base64）。"
            else -> null
        }
    }

    object EN : SingBoxSchemaStrings {
        override val configSnippet: String = "Configuration snippet"
        override val configHierarchyTooDeep: String = "Configuration hierarchy is too deep to continue validation"
        override val stringFormatMismatch: String = "String format does not meet requirements"
        override val noMatchingType: String = "Does not match any available configuration type"
        override val multipleMatchingTypes: String = "Matches multiple configuration types simultaneously"
        override val notMatchingAllowedFormat: String = "Does not match any allowed configuration format"
        override val cannotParseConfig: String = "Cannot parse configuration"
        override val typeOr: String = " or "

        override fun typeMismatch(allowedTypes: String, actualType: String): String =
            "Expected $allowedTypes, but got $actualType"

        override fun valueMustBe(expected: String): String = "Value must be $expected"
        override fun valueNotInAllowed(allowed: String): String = "Value is not in allowed range: $allowed"
        override fun numberCannotBeLessThan(minimum: Any): String = "Value cannot be less than $minimum"
        override fun numberCannotBeGreaterThan(maximum: Any): String = "Value cannot be greater than $maximum"
        override fun missingRequiredField(name: String): String = "Missing required field \"$name\""
        override fun fieldNotAllowed(name: String): String = "Field not allowed: \"$name\""
        override fun jsonSyntaxError(message: String): String = "JSON syntax error: $message"

        override fun isTypeIssue(message: String): Boolean =
            "configuration type" in message || "allowed configuration format" in message

        override fun localizedType(type: String): String = when (type) {
            "string" -> "string"
            "integer" -> "integer"
            "number" -> "number"
            "boolean" -> "boolean"
            "object" -> "object"
            "array" -> "array"
            "null" -> "null"
            else -> type
        }

        override fun fieldDocumentation(field: String): String? = when (field) {
            "type" -> "Type of configuration object; completion filters fields accordingly."
            "tag" -> "Unique name of this object, referenced by other configs via tag."
            "enabled" -> "Controls whether current feature is enabled."
            "server" -> "Remote server address, domain name or IP address."
            "server_port" -> "Remote server port."
            "listen" -> "Local listen address."
            "listen_port" -> "Local listen port."
            "outbound" -> "Outbound tag used when matched."
            "default_domain_resolver" -> "DNS server tag used to resolve server domains."
            "rule_set" -> "Matches one or more rule set tags."
            "rules" -> "List of rules matched in order."
            "action" -> "Action executed when rule matches."
            "servers" -> "Servers or member tags included in this group."
            "url" -> "URL used for download, health check or latency test."
            "interval" -> "Interval for auto update or test, e.g. 3m, 1h."
            "path" -> "Local file or resource path."
            "initial_path" -> "Local resource path used on first startup."
            "http_client" -> "HTTP Client tag used for remote requests."
            "secret" -> "Authentication secret for control interface."
            // AmneziaWG 混淆与 WireGuard 协议参数
            "private_key" -> "Private key for WireGuard / AmneziaWG node."
            "peers" -> "List of WireGuard / AmneziaWG peer configurations."
            "allowed_ips" -> "List of IP address prefixes routed through this peer."
            "pre_shared_key" -> "Optional pre-shared key (PSK) providing post-quantum security."
            "persistent_keepalive_interval" -> "Interval in seconds between keepalive heartbeat packets."
            "reserved" -> "Reserved bytes for WireGuard (e.g. Warp client recognition)."
            "jc" -> "Number of junk/dummy packets sent before AmneziaWG handshake (0-255)."
            "jmin" -> "Minimum junk packet payload size in bytes for AmneziaWG."
            "jmax" -> "Maximum junk packet payload size in bytes for AmneziaWG."
            "s1" -> "Padding size in bytes for AmneziaWG handshake initiation packet."
            "s2" -> "Padding size in bytes for AmneziaWG handshake response packet."
            "s3" -> "Padding size in bytes for AmneziaWG cookie packet."
            "s4" -> "Padding size in bytes for AmneziaWG transport data packet."
            "h1" -> "Custom message header for AmneziaWG handshake initiation packet."
            "h2" -> "Custom message header for AmneziaWG handshake response packet."
            "h3" -> "Custom message header for AmneziaWG cookie packet."
            "h4" -> "Custom message header for AmneziaWG transport data packet."
            "i1" -> "Custom junk packet payload 1 for AmneziaWG (hex or base64)."
            "i2" -> "Custom junk packet payload 2 for AmneziaWG (hex or base64)."
            "i3" -> "Custom junk packet payload 3 for AmneziaWG (hex or base64)."
            "i4" -> "Custom junk packet payload 4 for AmneziaWG (hex or base64)."
            "i5" -> "Custom junk packet payload 5 for AmneziaWG (hex or base64)."
            else -> null
        }
    }

    object RU : SingBoxSchemaStrings {
        override val configSnippet: String = "Фрагмент конфигурации"
        override val configHierarchyTooDeep: String = "Иерархия конфигурации слишком глубокая для продолжения проверки"
        override val stringFormatMismatch: String = "Формат строки не соответствует требованиям"
        override val noMatchingType: String = "Не соответствует ни одному доступному типу конфигурации"
        override val multipleMatchingTypes: String = "Одновременно соответствует нескольким типам конфигурации"
        override val notMatchingAllowedFormat: String = "Не соответствует ни одному допустимому формату конфигурации"
        override val cannotParseConfig: String = "Не удалось разобрать конфигурацию"
        override val typeOr: String = " или "

        override fun typeMismatch(allowedTypes: String, actualType: String): String =
            "Ожидалось $allowedTypes, получено $actualType"

        override fun valueMustBe(expected: String): String = "Значение должно быть $expected"
        override fun valueNotInAllowed(allowed: String): String = "Значение вне допустимого диапазона: $allowed"
        override fun numberCannotBeLessThan(minimum: Any): String = "Значение не может быть меньше $minimum"
        override fun numberCannotBeGreaterThan(maximum: Any): String = "Значение не может быть больше $maximum"
        override fun missingRequiredField(name: String): String = "Отсутствует обязательное поле \"$name\""
        override fun fieldNotAllowed(name: String): String = "Недопустимое поле \"$name\""
        override fun jsonSyntaxError(message: String): String = "Синтаксическая ошибка JSON: $message"

        override fun isTypeIssue(message: String): Boolean =
            "типу конфигурации" in message || "формату конфигурации" in message

        override fun localizedType(type: String): String = when (type) {
            "string" -> "строка"
            "integer" -> "целое число"
            "number" -> "число"
            "boolean" -> "булево значение"
            "object" -> "объект"
            "array" -> "массив"
            "null" -> "null"
            else -> type
        }

        override fun fieldDocumentation(field: String): String? = when (field) {
            "type" -> "Тип объекта конфигурации; автодополнение покажет поддерживаемые поля."
            "tag" -> "Уникальное имя объекта для ссылки из других конфигураций через тег."
            "enabled" -> "Определяет, включён ли текущий компонент."
            "server" -> "Адрес удалённого сервера (домен или IP-адрес)."
            "server_port" -> "Порт удалённого сервера."
            "listen" -> "Локальный адрес прослушивания."
            "listen_port" -> "Локальный порт прослушивания."
            "outbound" -> "Тег исходящего узла при совпадении правила."
            "default_domain_resolver" -> "Тег DNS-сервера для разрешения доменных имён сервера."
            "rule_set" -> "Соответствие одному или нескольким наборам правил."
            "rules" -> "Список правил, проверяемых по порядку."
            "action" -> "Действие при срабатывании правила."
            "servers" -> "Список серверов или тегов участников в этой группе."
            "url" -> "URL-адрес для загрузки, проверки работоспособности или теста задержки."
            "interval" -> "Интервал автоматического обновления или проверки (например: 3m, 1h)."
            "path" -> "Путь к локальному файлу или ресурсу."
            "initial_path" -> "Путь к локальному ресурсу при первом запуске."
            "http_client" -> "Тег HTTP-клиента для удалённых запросов."
            "secret" -> "Секретный ключ для авторизации в интерфейсе управления."
            // AmneziaWG 混淆与 WireGuard 协议参数
            "private_key" -> "Закрытый ключ узла WireGuard / AmneziaWG."
            "peers" -> "Список конфигураций пиров WireGuard / AmneziaWG."
            "allowed_ips" -> "Список диапазонов IP-адресов, маршрутизируемых через этот пир."
            "pre_shared_key" -> "Необязательный предварительно согласованный ключ (PSK) для постквантовой защиты."
            "persistent_keepalive_interval" -> "Интервал отправки пакетов поддержания соединения в секундах."
            "reserved" -> "Зарезервированные байты WireGuard (например, для идентификации клиента Warp)."
            "jc" -> "Число мусорных пакетов, отправляемых перед рукопожатием AmneziaWG (0–255)."
            "jmin" -> "Минимальный размер полезной нагрузки мусорного пакета AmneziaWG в байтах."
            "jmax" -> "Максимальный размер полезной нагрузки мусорного пакета AmneziaWG в байтах."
            "s1" -> "Размер дополнения пакета инициализации рукопожатия AmneziaWG в байтах."
            "s2" -> "Размер дополнения пакета ответа на рукопожатие AmneziaWG в байтах."
            "s3" -> "Размер дополнения пакета cookie AmneziaWG в байтах."
            "s4" -> "Размер дополнения транспортного пакета данных AmneziaWG в байтах."
            "h1" -> "Заголовок пакета инициализации рукопожатия AmneziaWG (магическое число)."
            "h2" -> "Заголовок пакета ответа на рукопожатие AmneziaWG (магическое число)."
            "h3" -> "Заголовок пакета cookie AmneziaWG (магическое число)."
            "h4" -> "Заголовок транспортного пакета данных AmneziaWG (магическое число)."
            "i1" -> "Полезная нагрузка пользовательского мусорного пакета 1 AmneziaWG."
            "i2" -> "Полезная нагрузка пользовательского мусорного пакета 2 AmneziaWG."
            "i3" -> "Полезная нагрузка пользовательского мусорного пакета 3 AmneziaWG."
            "i4" -> "Полезная нагрузка пользовательского мусорного пакета 4 AmneziaWG."
            "i5" -> "Полезная нагрузка пользовательского мусорного пакета 5 AmneziaWG."
            else -> null
        }
    }

    companion object {
        fun forLocale(locale: Locale): SingBoxSchemaStrings = when (locale.language.lowercase()) {
            "zh" -> ZH
            "ru" -> RU
            else -> EN
        }

        fun forContext(context: Context): SingBoxSchemaStrings {
            val config = context.resources.configuration
            val locale = if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.N) {
                config.locales.get(0) ?: Locale.getDefault()
            } else {
                @Suppress("DEPRECATION")
                config.locale ?: Locale.getDefault()
            }
            return forLocale(locale)
        }
    }
}

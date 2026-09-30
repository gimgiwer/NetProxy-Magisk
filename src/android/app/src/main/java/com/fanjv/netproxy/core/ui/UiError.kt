package com.fanjv.netproxy.core.ui

import kotlinx.coroutines.CancellationException
import java.util.Locale

/** 将业务异常转为界面文案，同时保持协程取消语义与多语言本地化。 */
internal fun Throwable.userMessage(): String {
    if (this is CancellationException) throw this
    val raw = message?.trim()
    val lang = Locale.getDefault().language.lowercase()
    if (raw.isNullOrBlank()) {
        return when (lang) {
            "ru" -> "Сбой операции, повторите попытку позже"
            "zh" -> "操作失败，请稍后重试"
            else -> "Operation failed, please try again later"
        }
    }
    return translateErrorMessage(raw, lang)
}

private fun translateErrorMessage(raw: String, lang: String): String {
    if (lang.startsWith("zh")) return raw
    val isRu = lang == "ru"
    return if (isRu) translateErrorRu(raw) else translateErrorEn(raw)
}

private fun translateErrorRu(raw: String): String {
    val exact = mapOf(
        "目标分组没有可用节点" to "В целевой группе нет доступных узлов",
        "订阅中没有可用节点" to "В подписке нет доступных узлов",
        "Catalog 中没有可用节点，请先导入单节点、文件或订阅" to "В каталоге нет доступных узлов. Сначала импортируйте узел, файл или подписку",
        "订阅目录或分组无效" to "Каталог или группа подписки недействительны",
        "订阅不存在" to "Подписка не найдена",
        "已有服务操作正在执行" to "Другая операция со службой уже выполняется",
        "已存在服务操作正在执行" to "Другая операция со службой уже выполняется",
        "sing-box 路径为空" to "Путь к sing-box не задан",
        "sing-box 二进制路径为空" to "Путь к исполняемому файлу sing-box не задан",
        "后台 Worker 已在运行" to "Фоновый воркер уже запущен",
        "该节点类型暂不支持导出为分享链接" to "Этот тип узла пока не поддерживает экспорт в ссылку",
        "缺少 sub 操作" to "Не указано действие sub",
        "缺少 config 操作" to "Не указано действие config",
        "分组名称不能为空" to "Имя группы не может быть пустым",
        "节点标签不能为空" to "Метка узла не может быть пустой",
        "输入内容不能为空" to "Входные данные не могут быть пустыми",
        "订阅 URL 不能为空" to "URL подписки не может быть пустым",
        "服务未运行" to "Служба не запущена",
        "服务正在运行" to "Служба уже запущена",
        "配置验证失败" to "Ошибка проверки конфигурации",
        "读取状态失败" to "Не удалось прочитать статус",
        "配置已被修改，请重新加载后再保存" to "Конфигурация была изменена, перезагрузите перед сохранением",
        "配置不是有效 JSON" to "Конфигурация не является корректным JSON",
        "配置必须是 JSON 对象，不能为 null" to "Конфигурация должна быть объектом JSON и не может быть null",
        "Android 网络尚未就绪" to "Сеть Android ещё не готова",
        "Wi-Fi 已连接但无法确认 SSID" to "Wi-Fi подключён, но не удалось определить SSID",
        "网络类型必须是 wifi 或 not_wifi" to "Тип сети должен быть wifi или not_wifi",
        "应用包名不能为空" to "Имя пакета приложения не может быть пустым",
        "应用模式应为 blacklist 或 whitelist" to "Режим приложений должен быть blacklist или whitelist",
        "更新周期不能为空" to "Интервал обновления не может быть пустым",
        "更新周期无效" to "Недопустимый интервал обновления",
        "磁盘恢复失败" to "Сбой восстановления с диска",
        "跨进程锁正忙" to "Межпроцессная блокировка занята",
        "运行时配置只读" to "Конфигурация времени выполнения доступна только для чтения",
        "命令不存在" to "Команда не найдена",
        "配置无效" to "Конфигурация недействительна",
        "更新失败" to "Сбой обновления",
        "更新成功" to "Успешно обновлено",
        "不能包含换行或制表符" to "Не может содержать переводы строк или символы табуляции",
        "双引号未闭合" to "Двойные кавычки не закрыты",
        "操作失败，请稍后重试" to "Сбой операции, повторите попытку позже",
        "Provider 不是普通文件" to "Провайдер не является обычным файлом",
        "不支持的运行时文件" to "Неподдерживаемый файл времени выполнения",
        "不支持的配置目标" to "Неподдерживаемая цель конфигурации",
        "日志路径不能为空" to "Путь к логу не может быть пустым",
        "诊断包路径不能为空" to "Путь к диагностическому пакету не может быть пустым",
        "sing-box 进程已退出" to "Процесс sing-box завершился",
        "sing-box 未运行，无法重新加载" to "sing-box не запущен, перезагрузка невозможна",
        "核心或控制接口未在限定时间内就绪" to "Ядро или интерфейс управления не запустились вовремя"
    )
    exact[raw]?.let { return it }

    val patterns = listOf(
        "未找到节点:\\s*(.*)".toRegex() to "Узел не найден: $1",
        "分组不存在:\\s*(.*)".toRegex() to "Группа не существует: $1",
        "未知出站模式:\\s*(.*)".toRegex() to "Неизвестный режим маршрутизации: $1",
        "分应用代理跳过未安装应用:\\s*(.*)".toRegex() to "Пропуск неустановленного приложения: $1",
        "Catalog 分组已存在:\\s*(.*)".toRegex() to "Группа каталога уже существует: $1",
        "非法 Catalog 分组 ID:\\s*(.*)".toRegex() to "Недопустимый идентификатор группы каталога: $1",
        "非法分组 ID:\\s*(.*)".toRegex() to "Недопустимый идентификатор группы: $1",
        "非法配置键:\\s*(.*)".toRegex() to "Недопустимый ключ конфигурации: $1",
        "不支持的 module.conf 配置键:\\s*(.*)".toRegex() to "Неподдерживаемый ключ конфигурации module.conf: $1",
        "OUTBOUND_MODE 无效:\\s*(.*)".toRegex() to "Недопустимый OUTBOUND_MODE: $1",
        "SELECTOR_MODE 无效:\\s*(.*)".toRegex() to "Недопустимый SELECTOR_MODE: $1",
        "WIFI_SSID_MODE 无效:\\s*(.*)".toRegex() to "Недопустимый WIFI_SSID_MODE: $1",
        "活动分组不存在:\\s*(.*)".toRegex() to "Активная группа не существует: $1",
        "读取分组\\s+(.*?)\\s+元数据失败:\\s*(.*)".toRegex() to "Не удалось прочитать метаданные группы $1: $2",
        "读取活动分组\\s+(.*?)\\s+Provider 失败:\\s*(.*)".toRegex() to "Не удалось прочитать провайдер активной группы $1: $2",
        "协议\\s+(.*?)\\s+暂不支持导出为分享链接".toRegex() to "Протокол $1 пока не поддерживается для экспорта в ссылку",
        "VMess 传输类型\\s+(.*?)\\s+暂不支持导出".toRegex() to "Тип транспорта VMess $1 пока не поддерживается для экспорта",
        "sing-box 配置检查失败:\\s*(.*)".toRegex() to "Сбой проверки конфигурации sing-box: $1",
        "sing-box 原位重新加载失败:\\s*(.*)".toRegex() to "Сбой перезагрузки sing-box на месте: $1",
        "配置 reload 失败:\\s*(.*)".toRegex() to "Сбой перезагрузки конфигурации: $1",
        "配置 reload 失败，已恢复旧配置:\\s*(.*)".toRegex() to "Сбой перезагрузки конфигурации, восстановлена предыдущая: $1",
        "恢复旧配置失败:\\s*(.*)".toRegex() to "Сбой восстановления предыдущей конфигурации: $1",
        "加载模块配置失败:\\s*(.*)".toRegex() to "Сбой загрузки конфигурации модуля: $1",
        "提交配置事务失败:\\s*(.*)".toRegex() to "Сбой фиксации транзакции конфигурации: $1"
    )

    var result = raw
    for ((regex, replacement) in patterns) {
        if (regex.containsMatchIn(result)) {
            result = regex.replace(result, replacement)
        }
    }

    if (result.any { it in '\u4e00'..'\u9fa5' }) {
        result = result
            .replace("本地配置", "Локальная конфигурация")
            .replace("失败", "Ошибка")
            .replace("成功", "Успешно")
            .replace("已完成", "Завершено")
            .replace("正在执行", "Выполняется")
            .replace("未运行", "Не запущен")
            .replace("已停止", "Остановлен")
    }

    return result
}

private fun translateErrorEn(raw: String): String {
    val exact = mapOf(
        "目标分组没有可用节点" to "Target group has no available nodes",
        "订阅中没有可用节点" to "No available nodes in subscription",
        "Catalog 中没有可用节点，请先导入单节点、文件或订阅" to "No available nodes in Catalog; import a single node, file, or subscription first",
        "订阅目录或分组无效" to "Invalid subscription directory or group",
        "订阅不存在" to "Subscription does not exist",
        "已有服务操作正在执行" to "Another service operation is already in progress",
        "已存在服务操作正在执行" to "Another service operation is already in progress",
        "sing-box 路径为空" to "sing-box path is empty",
        "sing-box 二进制路径为空" to "sing-box binary path is empty",
        "后台 Worker 已在运行" to "Background worker is already running",
        "该节点类型暂不支持导出为分享链接" to "This node type cannot be exported as a share link",
        "缺少 sub 操作" to "Missing sub action",
        "缺少 config 操作" to "Missing config action",
        "分组名称不能为空" to "Group name cannot be empty",
        "节点标签不能为空" to "Node tag cannot be empty",
        "输入内容不能为空" to "Input content cannot be empty",
        "订阅 URL 不能为空" to "Subscription URL cannot be empty",
        "服务未运行" to "Service is not running",
        "服务正在运行" to "Service is running",
        "配置验证失败" to "Configuration validation failed",
        "读取状态失败" to "Failed to read status",
        "配置已被修改，请重新加载后再保存" to "Configuration was modified; reload before saving",
        "配置不是有效 JSON" to "Configuration is not valid JSON",
        "配置必须是 JSON 对象，不能为 null" to "Configuration must be a JSON object, cannot be null",
        "Android 网络尚未就绪" to "Android network is not ready yet",
        "Wi-Fi 已连接但无法确认 SSID" to "Wi-Fi connected but cannot determine SSID",
        "网络类型必须是 wifi 或 not_wifi" to "Network type must be wifi or not_wifi",
        "应用包名不能为空" to "Application package name cannot be empty",
        "应用模式应为 blacklist 或 whitelist" to "App mode must be blacklist or whitelist",
        "更新周期不能为空" to "Update interval cannot be empty",
        "更新周期无效" to "Invalid update interval",
        "磁盘恢复失败" to "Disk recovery failed",
        "跨进程锁正忙" to "Cross-process lock is busy",
        "运行时配置只读" to "Runtime configuration is read-only",
        "命令不存在" to "Command does not exist",
        "配置无效" to "Invalid configuration",
        "更新失败" to "Update failed",
        "更新成功" to "Updated successfully",
        "不能包含换行或制表符" to "Cannot contain newlines or tabs",
        "双引号未闭合" to "Double quotes are not closed",
        "操作失败，请稍后重试" to "Operation failed, please try again later",
        "Provider 不是普通文件" to "Provider is not a regular file",
        "不支持的运行时文件" to "Unsupported runtime file",
        "不支持的配置目标" to "Unsupported configuration target",
        "日志路径不能为空" to "Log path cannot be empty",
        "诊断包路径不能为空" to "Diagnostics package path cannot be empty",
        "sing-box 进程已退出" to "sing-box process has exited",
        "sing-box 未运行，无法重新加载" to "sing-box is not running, cannot reload",
        "核心或控制接口未在限定时间内就绪" to "Core or control interface did not get ready in time"
    )
    exact[raw]?.let { return it }

    val patterns = listOf(
        "未找到节点:\\s*(.*)".toRegex() to "Node not found: $1",
        "分组不存在:\\s*(.*)".toRegex() to "Group does not exist: $1",
        "未知出站模式:\\s*(.*)".toRegex() to "Unknown outbound mode: $1",
        "分应用代理跳过未安装应用:\\s*(.*)".toRegex() to "Skipped uninstalled app: $1",
        "Catalog 分组已存在:\\s*(.*)".toRegex() to "Catalog group already exists: $1",
        "非法 Catalog 分组 ID:\\s*(.*)".toRegex() to "Invalid catalog group ID: $1",
        "非法分组 ID:\\s*(.*)".toRegex() to "Invalid group ID: $1",
        "非法配置键:\\s*(.*)".toRegex() to "Invalid configuration key: $1",
        "不支持的 module.conf 配置键:\\s*(.*)".toRegex() to "Unsupported module.conf configuration key: $1",
        "OUTBOUND_MODE 无效:\\s*(.*)".toRegex() to "Invalid OUTBOUND_MODE: $1",
        "SELECTOR_MODE 无效:\\s*(.*)".toRegex() to "Invalid SELECTOR_MODE: $1",
        "WIFI_SSID_MODE 无效:\\s*(.*)".toRegex() to "Invalid WIFI_SSID_MODE: $1",
        "活动分组不存在:\\s*(.*)".toRegex() to "Active group does not exist: $1",
        "读取分组\\s+(.*?)\\s+元数据失败:\\s*(.*)".toRegex() to "Failed to read group $1 metadata: $2",
        "读取活动分组\\s+(.*?)\\s+Provider 失败:\\s*(.*)".toRegex() to "Failed to read provider for active group $1: $2",
        "协议\\s+(.*?)\\s+暂不支持导出为分享链接".toRegex() to "Protocol $1 is not yet supported for sharing link export",
        "VMess 传输类型\\s+(.*?)\\s+暂不支持导出".toRegex() to "VMess transport type $1 is not yet supported for export",
        "sing-box 配置检查失败:\\s*(.*)".toRegex() to "sing-box configuration check failed: $1",
        "sing-box 原位重新加载失败:\\s*(.*)".toRegex() to "sing-box in-place reload failed: $1",
        "配置 reload 失败:\\s*(.*)".toRegex() to "Configuration reload failed: $1",
        "配置 reload 失败，已恢复旧配置:\\s*(.*)".toRegex() to "Configuration reload failed, restored previous: $1",
        "恢复旧配置失败:\\s*(.*)".toRegex() to "Failed to restore previous configuration: $1",
        "加载模块配置失败:\\s*(.*)".toRegex() to "Failed to load module configuration: $1",
        "提交配置事务失败:\\s*(.*)".toRegex() to "Failed to commit configuration transaction: $1"
    )

    var result = raw
    for ((regex, replacement) in patterns) {
        if (regex.containsMatchIn(result)) {
            result = regex.replace(result, replacement)
        }
    }

    if (result.any { it in '\u4e00'..'\u9fa5' }) {
        result = result
            .replace("本地配置", "Local Configuration")
            .replace("失败", "Failed")
            .replace("成功", "Success")
            .replace("已完成", "Completed")
            .replace("正在执行", "In progress")
            .replace("未运行", "Not running")
            .replace("已停止", "Stopped")
    }

    return result
}

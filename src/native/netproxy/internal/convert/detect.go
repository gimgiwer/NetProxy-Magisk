package convert

import (
	"strings"
)

// Format 表示订阅或配置内容的格式分类。
type Format string

const (
	// FormatUnknown 表示无法识别的格式。
	FormatUnknown Format = "unknown"
	// FormatXrayJSON 表示 Xray / Happ / Incy JSON 订阅或配置格式。
	FormatXrayJSON Format = "xray"
	// FormatSingBoxJSON 表示 Sing-box 原生 JSON 或 SIP008 JSON 订阅格式。
	FormatSingBoxJSON Format = "sing-box"
	// FormatClashYAML 表示 Clash / Mihomo YAML 订阅或配置文件格式。
	FormatClashYAML Format = "clash"
	// FormatWireGuard 表示 WireGuard 或 AmneziaWG INI/CONF 配置文件格式。
	FormatWireGuard Format = "wireguard"
	// FormatURIList 表示多行或单行节点分享链接（URI）列表格式。
	FormatURIList Format = "uri"
)

// DetectFormat 精确检测订阅文本内容的格式类型。
func DetectFormat(content string) Format {
	content = strings.TrimPrefix(content, "\xef\xbb\xbf")
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return FormatUnknown
	}

	// 1. 判断是否为 Xray / Happ / Incy JSON 格式（先于通用 Sing-box JSON 判断）
	if isXrayJSON(trimmed) {
		return FormatXrayJSON
	}

	// 2. 判断是否为 Sing-box JSON 或 SIP008 JSON
	if isJSONFormat(trimmed) {
		return FormatSingBoxJSON
	}

	// 2. 判断是否为 WireGuard / AmneziaWG INI/CONF 格式
	if isWireGuardConf(trimmed) {
		return FormatWireGuard
	}

	// 3. 判断是否为 Clash / Mihomo YAML 格式
	if isClashYAML(trimmed) {
		return FormatClashYAML
	}

	// 4. 判断是否为 URI 链接列表格式
	if isURIList(trimmed) {
		return FormatURIList
	}

	return FormatUnknown
}

// isXrayJSON 检查内容是否符合 Xray / Happ / Incy JSON 订阅或配置格式。
func isXrayJSON(trimmed string) bool {
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return false
	}
	lower := strings.ToLower(trimmed)
	// Xray / Happ 特有标志字段：remarks、streamSettings、vnext、realitySettings、burstObservatory、balancers
	if strings.Contains(lower, `"remarks"`) ||
		strings.Contains(lower, `"streamsettings"`) ||
		strings.Contains(lower, `"vnext"`) ||
		strings.Contains(lower, `"realitysettings"`) ||
		strings.Contains(lower, `"burstobservatory"`) ||
		strings.Contains(lower, `"balancers"`) {
		return true
	}
	// protocol 字段配合 Xray 典型协议或内置出站
	if strings.Contains(lower, `"protocol"`) && (strings.Contains(lower, `"vless"`) ||
		strings.Contains(lower, `"vmess"`) ||
		strings.Contains(lower, `"hysteria"`) ||
		strings.Contains(lower, `"trojan"`) ||
		strings.Contains(lower, `"freedom"`) ||
		strings.Contains(lower, `"blackhole"`)) {
		return true
	}
	return false
}

// isJSONFormat 检查内容是否符合 Sing-box 或 SIP008 JSON 格式规范。
func isJSONFormat(trimmed string) bool {
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return false
	}
	// 包含关键字段或是标准 JSON 节点数组
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, `"outbounds"`) ||
		strings.Contains(lower, `"endpoints"`) ||
		strings.Contains(lower, `"servers"`) ||
		strings.Contains(lower, `"inbounds"`) ||
		strings.Contains(lower, `"route"`) {
		return true
	}
	if strings.HasPrefix(trimmed, "[") && strings.Contains(lower, `"type"`) {
		return true
	}
	return false
}

// isWireGuardConf 检查内容是否符合 WireGuard 或 AmneziaWG INI/CONF 配置结构。
func isWireGuardConf(trimmed string) bool {
	lower := strings.ToLower(trimmed)
	if !strings.Contains(lower, "[interface]") {
		return false
	}
	// 包含典型 WireGuard 字段或 Peer 小节
	return strings.Contains(lower, "privatekey") ||
		strings.Contains(lower, "address") ||
		strings.Contains(lower, "[peer]")
}

// isClashYAML 检查内容是否符合 Clash 或 Mihomo YAML 代理配置规范。
func isClashYAML(trimmed string) bool {
	// 常见 Clash 关键字匹配
	if strings.HasPrefix(trimmed, "proxies:") || strings.Contains(trimmed, "\nproxies:") {
		return true
	}
	if strings.HasPrefix(trimmed, "payload:") || strings.Contains(trimmed, "\npayload:") {
		return true
	}
	if strings.HasPrefix(trimmed, "proxy-providers:") || strings.Contains(trimmed, "\nproxy-providers:") {
		return true
	}
	// 含有 YAML 键值且定义了代理类型
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "port:") && strings.Contains(lower, "server:") && strings.Contains(lower, "type:") {
		return true
	}
	return false
}

// isURIList 检查多行文本中是否包含合法的节点分享 URI 协议头。
func isURIList(trimmed string) bool {
	knownSchemes := []string{
		"vless://",
		"vmess://",
		"ss://",
		"trojan://",
		"hysteria2://",
		"hy2://",
		"hysteria://",
		"tuic://",
		"awg://",
		"wireguard://",
		"socks://",
		"socks5://",
		"http://",
		"https://",
		"anytls://",
	}

	lines := strings.Split(trimmed, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lower := strings.ToLower(line)
		for _, scheme := range knownSchemes {
			if strings.HasPrefix(lower, scheme) {
				return true
			}
		}
	}
	return false
}

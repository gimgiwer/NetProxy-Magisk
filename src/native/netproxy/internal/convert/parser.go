package convert

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	providerparser "github.com/sagernet/sing-box/provider/parser"
	"github.com/sagernet/sing/common/json/badoption"
	"gopkg.in/yaml.v3"

	"github.com/Fanju6/NetProxy-Magisk/src/native/netproxy/internal/provider"
)

// NormalizeRawPayload 规范化原始订阅负载，移除 UTF-8 BOM，统一换行符并执行最多 3 轮递归 Base64 解码。
func NormalizeRawPayload(content string) string {
	// 1. 移除 UTF-8 字节顺序标记 (BOM)
	content = strings.TrimPrefix(content, "\xef\xbb\xbf")

	// 2. 统一换行符 CRLF / CR -> LF
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// 3. 最多执行 3 轮递归 Base64 解码
	current := content
	for i := 0; i < 3; i++ {
		decoded, ok := tryDecodeBase64(current)
		if !ok {
			break
		}
		current = strings.TrimPrefix(decoded, "\xef\xbb\xbf")
		current = strings.ReplaceAll(current, "\r\n", "\n")
		current = strings.ReplaceAll(current, "\r", "\n")

		// 检查解码后的文本是否已呈现明确的配置结构
		trimmed := strings.TrimSpace(current)
		if strings.Contains(trimmed, "://") ||
			strings.HasPrefix(trimmed, "proxies:") ||
			strings.Contains(trimmed, "\nproxies:") ||
			strings.HasPrefix(trimmed, "{") ||
			strings.HasPrefix(trimmed, "[") ||
			strings.Contains(strings.ToLower(trimmed), "[interface]") {
			break
		}
	}
	return current
}

// tryDecodeBase64 尝试使用多种 Base64 编码方式解码文本块。
func tryDecodeBase64(value string) (string, bool) {
	compact := strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, value)

	if len(compact) < 4 {
		return "", false
	}

	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}

	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(compact)
		if err != nil {
			continue
		}
		if !utf8.Valid(decoded) {
			continue
		}
		text := string(decoded)
		if isPlausibleText(text) {
			return text, true
		}
	}
	return "", false
}

// isPlausibleText 检查解码后文本是否为可打印文本而非纯二进制数据。
func isPlausibleText(s string) bool {
	if len(s) == 0 {
		return false
	}
	runes := []rune(s)
	printableCount := 0
	for _, r := range runes {
		if r == 0 {
			return false
		}
		if r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r) {
			printableCount++
		}
	}
	return float64(printableCount)/float64(len(runes)) >= 0.9
}

// ParseClashYAML 解析 Clash / Mihomo YAML 格式内容，提取 proxies 列表并转换为 sing-box 节点。
func ParseClashYAML(ctx context.Context, content string) (provider.Document, []provider.Diagnostic, error) {
	// 兼容处理 hy2 协议简写为 hysteria2
	normalizedYAML := strings.ReplaceAll(content, "type: hy2", "type: hysteria2")
	normalizedYAML = strings.ReplaceAll(normalizedYAML, "type: \"hy2\"", "type: \"hysteria2\"")
	normalizedYAML = strings.ReplaceAll(normalizedYAML, "type: 'hy2'", "type: 'hysteria2'")

	type clashPayloadWrapper struct {
		Proxies []providerparser.ClashProxy `yaml:"proxies"`
		Payload []providerparser.ClashProxy `yaml:"payload"`
	}

	var wrapper clashPayloadWrapper
	if err := yaml.Unmarshal([]byte(normalizedYAML), &wrapper); err != nil {
		diagnostic := provider.Diagnostic{Code: "clash.invalid", Message: err.Error()}
		return provider.Document{}, []provider.Diagnostic{diagnostic}, &DiagnosticsError{Diagnostics: []provider.Diagnostic{diagnostic}}
	}

	proxies := wrapper.Proxies
	if len(proxies) == 0 && len(wrapper.Payload) > 0 {
		proxies = wrapper.Payload
	}

	var document provider.Document
	var diagnostics []provider.Diagnostic

	for index, proxy := range proxies {
		if proxy.Name == "" {
			proxy.Name = fmt.Sprintf("node_%d", index+1)
		}
		if proxy.SingType == "" {
			diagnostics = append(diagnostics, provider.Diagnostic{
				Index:   index + 1,
				Source:  proxy.Name,
				Code:    "clash.protocol_unsupported",
				Message: fmt.Sprintf("unsupported Clash protocol %q", proxy.Type),
			})
			continue
		}

		outbound := proxy.Build()
		if outbound.Type == "" || outbound.Options == nil {
			diagnostics = append(diagnostics, provider.Diagnostic{
				Index:   index + 1,
				Source:  proxy.Name,
				Code:    "clash.proxy_invalid",
				Message: fmt.Sprintf("failed to build outbound for %q", proxy.Name),
			})
			continue
		}
		document.Outbounds = append(document.Outbounds, outbound)

		// 若为 WireGuard 或 Tailscale，同时作为 Endpoint 加入文档
		if proxy.SingType == C.TypeWireGuard || proxy.SingType == C.TypeTailscale {
			endpoint := proxy.BuildEndpoint()
			if endpoint.Type != "" && endpoint.Options != nil {
				document.Endpoints = append(document.Endpoints, endpoint)
			}
		}
	}

	return document, diagnostics, nil
}

// ParseWireGuardConf 解析 WireGuard / AmneziaWG INI/CONF 配置文件内容。
func ParseWireGuardConf(content string) (provider.Document, []provider.Diagnostic, error) {
	lines := strings.Split(content, "\n")
	currentSection := ""

	var privateKey string
	var addresses badoption.Listable[netip.Prefix]
	var mtu uint32 = 1420
	var tag string

	var publicKey string
	var presharedKey string
	var endpointStr string
	var allowedIPs badoption.Listable[netip.Prefix]
	var keepalive uint16
	var reserved []uint8

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			comment := strings.TrimSpace(line[1:])
			lowerComment := strings.ToLower(comment)
			if before, _, found := strings.Cut(lowerComment, "="); found && strings.TrimSpace(before) == "name" {
				_, val, _ := strings.Cut(comment, "=")
				tag = strings.TrimSpace(val)
			} else if before, _, found := strings.Cut(lowerComment, ":"); found && strings.TrimSpace(before) == "name" {
				_, val, _ := strings.Cut(comment, ":")
				tag = strings.TrimSpace(val)
			} else if tag == "" && len(comment) > 0 && !strings.Contains(comment, "\n") {
				// 当注释不含配置描述字眼时，可作为候选标签
				if !strings.Contains(lowerComment, "amnezia") && !strings.Contains(lowerComment, "wireguard") && !strings.Contains(lowerComment, "config") {
					tag = comment
				}
			}
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}

		key, val, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)

		switch currentSection {
		case "interface":
			switch key {
			case "privatekey":
				privateKey = val
			case "address":
				for _, part := range strings.Split(val, ",") {
					part = strings.TrimSpace(part)
					if part == "" {
						continue
					}
					if !strings.Contains(part, "/") {
						if strings.Contains(part, ":") {
							part += "/128"
						} else {
							part += "/32"
						}
					}
					if prefix, err := netip.ParsePrefix(part); err == nil {
						addresses = append(addresses, prefix)
					}
				}
			case "mtu":
				if m, err := strconv.ParseUint(val, 10, 32); err == nil && m > 0 {
					mtu = uint32(m)
				}
			}
		case "peer":
			switch key {
			case "publickey":
				publicKey = val
			case "presharedkey":
				presharedKey = val
			case "endpoint":
				endpointStr = val
			case "allowedips":
				for _, part := range strings.Split(val, ",") {
					part = strings.TrimSpace(part)
					if part == "" {
						continue
					}
					if !strings.Contains(part, "/") {
						if strings.Contains(part, ":") {
							part += "/128"
						} else {
							part += "/32"
						}
					}
					if prefix, err := netip.ParsePrefix(part); err == nil {
						allowedIPs = append(allowedIPs, prefix)
					}
				}
			case "persistentkeepalive":
				if k, err := strconv.ParseUint(val, 10, 16); err == nil {
					keepalive = uint16(k)
				}
			case "reserved":
				reserved = parseReservedField(val)
			}
		}
	}

	if endpointStr == "" {
		diagnostic := provider.Diagnostic{Code: "wireguard.missing_endpoint", Message: "missing peer endpoint"}
		return provider.Document{}, []provider.Diagnostic{diagnostic}, &DiagnosticsError{Diagnostics: []provider.Diagnostic{diagnostic}}
	}

	host, portStr, err := net.SplitHostPort(endpointStr)
	if err != nil {
		diagnostic := provider.Diagnostic{Code: "wireguard.invalid_endpoint", Message: err.Error()}
		return provider.Document{}, []provider.Diagnostic{diagnostic}, &DiagnosticsError{Diagnostics: []provider.Diagnostic{diagnostic}}
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil || port == 0 {
		diagnostic := provider.Diagnostic{Code: "wireguard.invalid_port", Message: fmt.Sprintf("invalid port %q", portStr)}
		return provider.Document{}, []provider.Diagnostic{diagnostic}, &DiagnosticsError{Diagnostics: []provider.Diagnostic{diagnostic}}
	}

	if len(allowedIPs) == 0 {
		if v4, err := netip.ParsePrefix("0.0.0.0/0"); err == nil {
			allowedIPs = append(allowedIPs, v4)
		}
		if v6, err := netip.ParsePrefix("::/0"); err == nil {
			allowedIPs = append(allowedIPs, v6)
		}
	}

	if tag == "" {
		tag = endpointStr
	}

	peer := option.WireGuardPeer{
		Address:                     host,
		Port:                        uint16(port),
		PublicKey:                   publicKey,
		PreSharedKey:                presharedKey,
		AllowedIPs:                  allowedIPs,
		PersistentKeepaliveInterval: keepalive,
		Reserved:                    reserved,
	}

	endpointOptions := &option.WireGuardEndpointOptions{
		Address:    addresses,
		PrivateKey: privateKey,
		MTU:        mtu,
		Peers:      []option.WireGuardPeer{peer},
	}

	outbound := option.Outbound{
		Type:    C.TypeWireGuard,
		Tag:     tag,
		Options: endpointOptions,
	}
	endpoint := option.Endpoint{
		Type:    C.TypeWireGuard,
		Tag:     tag,
		Options: endpointOptions,
	}

	document := provider.Document{
		Outbounds: []option.Outbound{outbound},
		Endpoints: []option.Endpoint{endpoint},
	}
	return document, nil, nil
}

// ParseLink 解析单个节点分享链接，支持各类主流协议以及 #tag URL 编码提取。
func ParseLink(link string) (option.Outbound, error) {
	link = strings.TrimSpace(link)
	if link == "" {
		return option.Outbound{}, errors.New("empty link")
	}

	// 提取并 URL 解码 #tag
	var explicitTag string
	if before, after, found := strings.Cut(link, "#"); found {
		link = before
		if decoded, err := url.QueryUnescape(after); err == nil && decoded != "" {
			explicitTag = strings.TrimSpace(decoded)
		} else if unescaped, err := url.PathUnescape(after); err == nil && unescaped != "" {
			explicitTag = strings.TrimSpace(unescaped)
		} else {
			explicitTag = strings.TrimSpace(after)
		}
	}

	schemeEnd := strings.Index(link, "://")
	if schemeEnd <= 0 {
		return option.Outbound{}, errors.New("missing URI scheme")
	}
	scheme := strings.ToLower(link[:schemeEnd])

	var outbound option.Outbound
	var err error

	switch scheme {
	case "ss":
		fullLink := link
		if explicitTag != "" {
			fullLink += "#" + url.QueryEscape(explicitTag)
		}
		outbound, err = parseShadowsocks(fullLink)
	case "socks", "socks5":
		fullLink := link
		if explicitTag != "" {
			fullLink += "#" + url.QueryEscape(explicitTag)
		}
		outbound, err = parseSOCKS(fullLink)
	case "http", "https":
		fullLink := link
		if explicitTag != "" {
			fullLink += "#" + url.QueryEscape(explicitTag)
		}
		outbound, err = parseHTTP(fullLink)
	case "wireguard", "awg":
		outbound, err = parseWireGuardURI(link, explicitTag)
	default:
		fullLink := link
		if explicitTag != "" {
			fullLink += "#" + url.QueryEscape(explicitTag)
		}
		outbound, err = providerparser.ParseSubscriptionLink(fullLink)
	}

	if err != nil {
		return option.Outbound{}, err
	}

	if explicitTag != "" {
		outbound.Tag = explicitTag
	}
	if outbound.Tag == "" {
		outbound.Tag = defaultTagForOutbound(outbound)
	}
	return outbound, nil
}

// ParseURIList 解析多行 URI 链接列表文本。
func ParseURIList(content string) (provider.Document, []provider.Diagnostic, error) {
	var document provider.Document
	var diagnostics []provider.Diagnostic

	for index, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		outbound, err := ParseLink(line)
		if err != nil {
			diagnostics = append(diagnostics, provider.Diagnostic{
				Index:   index + 1,
				Source:  sourceLabel(line),
				Code:    "link.invalid",
				Message: err.Error(),
			})
			continue
		}
		document.Outbounds = append(document.Outbounds, outbound)
		if outbound.Type == C.TypeWireGuard {
			if wgOptions, ok := outbound.Options.(*option.WireGuardEndpointOptions); ok {
				document.Endpoints = append(document.Endpoints, option.Endpoint{
					Type:    C.TypeWireGuard,
					Tag:     outbound.Tag,
					Options: wgOptions,
				})
			}
		}
	}

	return document, diagnostics, nil
}

// parseWireGuardURI 解析 wireguard:// 或 awg:// 格式分享链接。
func parseWireGuardURI(rawURL, tag string) (option.Outbound, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return option.Outbound{}, fmt.Errorf("invalid wireguard URI: %w", err)
	}

	server := u.Hostname()
	if server == "" {
		return option.Outbound{}, errors.New("wireguard link requires a server hostname or IP")
	}

	var port uint16 = 51820
	if u.Port() != "" {
		p, err := strconv.ParseUint(u.Port(), 10, 16)
		if err != nil || p == 0 {
			return option.Outbound{}, fmt.Errorf("invalid wireguard port %q", u.Port())
		}
		port = uint16(p)
	}

	q := u.Query()
	privateKey := ""
	if u.User != nil {
		privateKey = u.User.Username()
	}
	if privateKey == "" {
		privateKey = firstNonEmpty(q.Get("private_key"), q.Get("privatekey"), q.Get("privkey"))
	}

	publicKey := firstNonEmpty(q.Get("public_key"), q.Get("publickey"), q.Get("pubkey"))
	psk := firstNonEmpty(q.Get("preshared_key"), q.Get("presharedkey"), q.Get("psk"))

	var addresses badoption.Listable[netip.Prefix]
	ipParam := firstNonEmpty(q.Get("address"), q.Get("ip"), q.Get("addresses"))
	if ipParam != "" {
		for _, part := range strings.Split(ipParam, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if !strings.Contains(part, "/") {
				if strings.Contains(part, ":") {
					part += "/128"
				} else {
					part += "/32"
				}
			}
			if prefix, err := netip.ParsePrefix(part); err == nil {
				addresses = append(addresses, prefix)
			}
		}
	}

	var allowedIPs badoption.Listable[netip.Prefix]
	allowedParam := firstNonEmpty(q.Get("allowed_ips"), q.Get("allowedips"))
	if allowedParam != "" {
		for _, part := range strings.Split(allowedParam, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if !strings.Contains(part, "/") {
				if strings.Contains(part, ":") {
					part += "/128"
				} else {
					part += "/32"
				}
			}
			if prefix, err := netip.ParsePrefix(part); err == nil {
				allowedIPs = append(allowedIPs, prefix)
			}
		}
	} else {
		if v4, err := netip.ParsePrefix("0.0.0.0/0"); err == nil {
			allowedIPs = append(allowedIPs, v4)
		}
		if v6, err := netip.ParsePrefix("::/0"); err == nil {
			allowedIPs = append(allowedIPs, v6)
		}
	}

	var mtu uint32 = 1420
	if mtuStr := q.Get("mtu"); mtuStr != "" {
		if m, err := strconv.ParseUint(mtuStr, 10, 32); err == nil && m > 0 {
			mtu = uint32(m)
		}
	}

	var keepalive uint16
	keepaliveStr := firstNonEmpty(q.Get("keepalive"), q.Get("persistent_keepalive"), q.Get("persistent-keepalive"))
	if keepaliveStr != "" {
		if k, err := strconv.ParseUint(keepaliveStr, 10, 16); err == nil {
			keepalive = uint16(k)
		}
	}

	reserved := parseReservedField(q.Get("reserved"))

	if tag == "" {
		tag = fmt.Sprintf("wireguard-%s:%d", server, port)
	}

	peer := option.WireGuardPeer{
		Address:                     server,
		Port:                        port,
		PublicKey:                   publicKey,
		PreSharedKey:                psk,
		AllowedIPs:                  allowedIPs,
		PersistentKeepaliveInterval: keepalive,
		Reserved:                    reserved,
	}

	endpointOptions := &option.WireGuardEndpointOptions{
		Address:    addresses,
		PrivateKey: privateKey,
		MTU:        mtu,
		Peers:      []option.WireGuardPeer{peer},
	}

	return option.Outbound{
		Type:    C.TypeWireGuard,
		Tag:     tag,
		Options: endpointOptions,
	}, nil
}

// parseReservedField 解析 WireGuard 的保留字节字段（支持逗号分隔数字或 Base64）。
func parseReservedField(val string) []uint8 {
	val = strings.TrimSpace(val)
	if val == "" {
		return nil
	}
	if strings.Contains(val, ",") {
		var res []uint8
		for _, part := range strings.Split(val, ",") {
			part = strings.TrimSpace(part)
			if num, err := strconv.ParseUint(part, 10, 8); err == nil {
				res = append(res, uint8(num))
			}
		}
		return res
	}
	if decoded, err := base64.StdEncoding.DecodeString(val); err == nil && len(decoded) > 0 {
		return []uint8(decoded)
	}
	return nil
}

// firstNonEmpty 返回第一个非空的字符串值。
func firstNonEmpty(values ...string) string {
	for _, val := range values {
		val = strings.TrimSpace(val)
		if val != "" {
			return val
		}
	}
	return ""
}

// defaultTagForOutbound 根据 outbound 类型和选项生成默认标签。
func defaultTagForOutbound(outbound option.Outbound) string {
	if serverOptions, ok := outbound.Options.(option.ServerOptionsWrapper); ok {
		server := serverOptions.TakeServerOptions()
		if server.Server != "" && server.ServerPort > 0 {
			return fmt.Sprintf("%s-%s:%d", outbound.Type, server.Server, server.ServerPort)
		}
	}
	if outbound.Type != "" {
		return outbound.Type
	}
	return "node"
}

package convert

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	providerparser "github.com/sagernet/sing-box/provider/parser"

	"github.com/Fanju6/NetProxy-Magisk/src/native/netproxy/internal/provider"
)

func init() {
	// 向 provider 包注册全局默认文档解析器，以解除包之间的循环依赖
	provider.SetDefaultParser(Content)
}

// DiagnosticsError 携带解析诊断信息列表的错误类型。
type DiagnosticsError struct {
	Diagnostics []provider.Diagnostic
}

func (e *DiagnosticsError) Error() string {
	if len(e.Diagnostics) == 0 {
		return "no nodes found"
	}
	return fmt.Sprintf("no nodes found: %s", e.Diagnostics[0].Message)
}

// Link 解析单个分享链接并校验有效性。
func Link(ctx context.Context, link string, allowInsecure bool) (provider.ParseResult, error) {
	outbound, err := ParseLink(strings.TrimSpace(link))
	if err != nil {
		diagnostic := provider.Diagnostic{Source: sourceLabel(link), Code: "link.invalid", Message: err.Error()}
		return provider.ParseResult{Diagnostics: []provider.Diagnostic{diagnostic}}, &DiagnosticsError{Diagnostics: []provider.Diagnostic{diagnostic}}
	}
	document := provider.Document{Outbounds: []option.Outbound{outbound}}
	if outbound.Type == C.TypeWireGuard {
		if wgOptions, ok := outbound.Options.(*option.WireGuardEndpointOptions); ok {
			document.Endpoints = append(document.Endpoints, option.Endpoint{
				Type:    C.TypeWireGuard,
				Tag:     outbound.Tag,
				Options: wgOptions,
			})
		}
	}
	provider.NormalizeTags(&document)
	if allowInsecure {
		applyAllowInsecure(&document)
	}
	if err := provider.Validate(document); err != nil {
		diagnostic := provider.Diagnostic{Source: sourceLabel(link), Code: "link.invalid", Message: err.Error()}
		return provider.ParseResult{Diagnostics: []provider.Diagnostic{diagnostic}}, &DiagnosticsError{Diagnostics: []provider.Diagnostic{diagnostic}}
	}
	return provider.ParseResult{Document: document}, nil
}

// Input 按文件路径、单链接或订阅文本自动解析输入。
func Input(ctx context.Context, input string, allowInsecure bool) (provider.ParseResult, error) {
	if info, err := os.Stat(input); err == nil && !info.IsDir() {
		content, err := os.ReadFile(input)
		if err != nil {
			return provider.ParseResult{}, err
		}
		return Content(ctx, string(content), allowInsecure)
	}
	if strings.Contains(input, "://") && !strings.Contains(input, "\n") {
		return Link(ctx, input, allowInsecure)
	}
	return Content(ctx, input, allowInsecure)
}

// Content 全功能全格式订阅内容解析入口。
func Content(ctx context.Context, content string, allowInsecure bool) (provider.ParseResult, error) {
	normalized := NormalizeRawPayload(content)
	trimmed := strings.TrimSpace(normalized)
	if trimmed == "" {
		diagnostic := provider.Diagnostic{Code: "input.empty", Message: "input is empty"}
		return provider.ParseResult{Diagnostics: []provider.Diagnostic{diagnostic}}, &DiagnosticsError{Diagnostics: []provider.Diagnostic{diagnostic}}
	}
	ctx = provider.Context(ctx)

	format := DetectFormat(trimmed)
	switch format {
	case FormatXrayJSON:
		doc, diags, err := ParseXrayJSON(ctx, trimmed)
		if err != nil {
			return provider.ParseResult{Diagnostics: diags}, err
		}
		return finish(doc, diags, allowInsecure)

	case FormatSingBoxJSON:
		if outbounds, endpoints, err := providerparser.ParseBoxSubscription(ctx, trimmed); err == nil && len(outbounds)+len(endpoints) > 0 {
			return finish(provider.Document{Outbounds: outbounds, Endpoints: endpoints}, nil, allowInsecure)
		}
		if outbounds, endpoints, err := providerparser.ParseSIP008Subscription(ctx, trimmed); err == nil && len(outbounds)+len(endpoints) > 0 {
			return finish(provider.Document{Outbounds: outbounds, Endpoints: endpoints}, nil, allowInsecure)
		}
		diagnostic := provider.Diagnostic{Code: "json.invalid", Message: "failed to parse Sing-box / SIP008 JSON subscription"}
		return provider.ParseResult{Diagnostics: []provider.Diagnostic{diagnostic}}, &DiagnosticsError{Diagnostics: []provider.Diagnostic{diagnostic}}

	case FormatClashYAML:
		doc, diags, err := ParseClashYAML(ctx, trimmed)
		if err != nil {
			return provider.ParseResult{Diagnostics: diags}, err
		}
		return finish(doc, diags, allowInsecure)

	case FormatWireGuard:
		doc, diags, err := ParseWireGuardConf(trimmed)
		if err != nil {
			return provider.ParseResult{Diagnostics: diags}, err
		}
		return finish(doc, diags, allowInsecure)

	case FormatURIList:
		doc, diags, err := ParseURIList(trimmed)
		if err != nil {
			return provider.ParseResult{Diagnostics: diags}, err
		}
		return finish(doc, diags, allowInsecure)

	default:
		// 当格式无法准确匹配时，进行多重智能回退尝试
		if doc, diags, err := ParseXrayJSON(ctx, trimmed); err == nil && len(doc.Outbounds)+len(doc.Endpoints) > 0 {
			return finish(doc, diags, allowInsecure)
		}
		if outbounds, endpoints, err := providerparser.ParseBoxSubscription(ctx, trimmed); err == nil && len(outbounds)+len(endpoints) > 0 {
			return finish(provider.Document{Outbounds: outbounds, Endpoints: endpoints}, nil, allowInsecure)
		}
		if doc, diags, err := ParseClashYAML(ctx, trimmed); err == nil && len(doc.Outbounds)+len(doc.Endpoints) > 0 {
			return finish(doc, diags, allowInsecure)
		}
		if doc, diags, err := ParseWireGuardConf(trimmed); err == nil && len(doc.Outbounds)+len(doc.Endpoints) > 0 {
			return finish(doc, diags, allowInsecure)
		}
		if doc, diags, err := ParseURIList(trimmed); err == nil && len(doc.Outbounds)+len(doc.Endpoints) > 0 {
			return finish(doc, diags, allowInsecure)
		}
		diagnostic := provider.Diagnostic{Code: "input.unknown_format", Message: "unsupported or unrecognizable subscription format"}
		return provider.ParseResult{Diagnostics: []provider.Diagnostic{diagnostic}}, &DiagnosticsError{Diagnostics: []provider.Diagnostic{diagnostic}}
	}
}

// finish 执行标签归一化、可选安全覆盖及完整性校验。
func finish(document provider.Document, diagnostics []provider.Diagnostic, allowInsecure bool) (provider.ParseResult, error) {
	provider.NormalizeTags(&document)
	if allowInsecure {
		applyAllowInsecure(&document)
	}
	if len(document.Outbounds)+len(document.Endpoints) == 0 {
		if len(diagnostics) == 0 {
			diagnostics = append(diagnostics, provider.Diagnostic{Code: "input.no_nodes", Message: "no supported nodes found"})
		}
		return provider.ParseResult{Diagnostics: diagnostics}, &DiagnosticsError{Diagnostics: diagnostics}
	}
	if err := provider.Validate(document); err != nil {
		diagnostics = append(diagnostics, provider.Diagnostic{Code: "provider.invalid", Message: err.Error()})
		return provider.ParseResult{Diagnostics: diagnostics}, &DiagnosticsError{Diagnostics: diagnostics}
	}
	return provider.ParseResult{Document: document, Diagnostics: diagnostics}, nil
}

// parseLink 保持内部方法签名兼容。
func parseLink(link string) (option.Outbound, error) {
	return ParseLink(link)
}

func parseShadowsocks(link string) (option.Outbound, error) {
	u, err := url.Parse(link)
	if err != nil {
		return option.Outbound{}, fmt.Errorf("invalid Shadowsocks link: %w", err)
	}
	if u.Hostname() == "" || u.User == nil {
		return option.Outbound{}, errors.New("Shadowsocks link requires credentials and a server")
	}
	method := u.User.Username()
	password, hasPassword := u.User.Password()
	if !hasPassword {
		decoded, ok := decodeBase64(method)
		if !ok {
			return option.Outbound{}, errors.New("invalid Shadowsocks credentials")
		}
		method, password, hasPassword = strings.Cut(decoded, ":")
	}
	if method == "" || !hasPassword || password == "" {
		return option.Outbound{}, errors.New("invalid Shadowsocks credentials")
	}
	port, err := parsePort(u.Port())
	if err != nil {
		return option.Outbound{}, err
	}
	options := &option.ShadowsocksOutboundOptions{
		Server:     u.Hostname(),
		ServerPort: port,
		Method:     method,
		Password:   password,
	}
	if plugin := u.Query().Get("plugin"); plugin != "" {
		options.Plugin, options.PluginOptions, _ = strings.Cut(plugin, ";")
	}
	return option.Outbound{Type: C.TypeShadowsocks, Tag: u.Fragment, Options: options}, nil
}

func parseSOCKS(link string) (option.Outbound, error) {
	_, after, _ := strings.Cut(link, "://")
	body := after
	fragment := ""
	if before, after, found := strings.Cut(body, "#"); found {
		body = before
		decoded, err := url.QueryUnescape(after)
		if err == nil {
			fragment = decoded
		}
	}
	u, err := parseSOCKSURL(body, true)
	if err != nil {
		return option.Outbound{}, err
	}
	port, err := parsePort(u.Port())
	if err != nil {
		return option.Outbound{}, err
	}
	options := &option.SOCKSOutboundOptions{
		Server:     u.Hostname(),
		ServerPort: port,
		Version:    "5",
	}
	if u.User != nil {
		options.Username = u.User.Username()
		options.Password, _ = u.User.Password()
		if options.Password == "" {
			if decoded, ok := decodeUserInfo(options.Username); ok {
				options.Username, options.Password, _ = strings.Cut(decoded, ":")
			}
		}
	}
	return option.Outbound{Type: C.TypeSOCKS, Tag: fragment, Options: options}, nil
}

func parseSOCKSURL(body string, allowLegacyBase64 bool) (*url.URL, error) {
	u, err := url.Parse("socks://" + body)
	if err == nil && u.Hostname() != "" && u.Port() != "" {
		return u, nil
	}
	if !allowLegacyBase64 {
		if err != nil {
			return nil, fmt.Errorf("invalid SOCKS link: %w", err)
		}
		return nil, errors.New("SOCKS link requires a server and port")
	}
	decoded, ok := decodeBase64(body)
	if !ok || decoded == body {
		if err != nil {
			return nil, fmt.Errorf("invalid SOCKS link: %w", err)
		}
		return nil, errors.New("SOCKS link requires a server and port")
	}
	return parseSOCKSURL(decoded, false)
}

func parseHTTP(link string) (option.Outbound, error) {
	u, err := url.Parse(link)
	if err != nil {
		return option.Outbound{}, fmt.Errorf("invalid HTTP proxy link: %w", err)
	}
	if u.Hostname() == "" {
		return option.Outbound{}, errors.New("HTTP proxy link requires a server")
	}
	port, err := parsePort(u.Port())
	if err != nil {
		return option.Outbound{}, err
	}
	options := &option.HTTPOutboundOptions{
		Server:     u.Hostname(),
		ServerPort: port,
	}
	if u.User != nil {
		options.Username = u.User.Username()
		options.Password, _ = u.User.Password()
	}
	if strings.EqualFold(u.Scheme, "https") {
		options.TLS = &option.OutboundTLSOptions{Enabled: true, ServerName: u.Hostname()}
	}
	return option.Outbound{Type: C.TypeHTTP, Tag: u.Fragment, Options: options}, nil
}

func parsePort(value string) (uint16, error) {
	port, err := strconv.ParseUint(value, 10, 16)
	if err != nil || port == 0 {
		return 0, fmt.Errorf("invalid server port %q", value)
	}
	return uint16(port), nil
}

func decodeBase64(value string) (string, bool) {
	encodings := []*base64.Encoding{
		base64.RawURLEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.StdEncoding,
	}
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(value)
		if err == nil {
			return string(decoded), true
		}
	}
	return "", false
}

func decodeUserInfo(value string) (string, bool) {
	decoded, ok := decodeBase64(value)
	return decoded, ok && strings.Contains(decoded, ":")
}

func sourceLabel(input string) string {
	input = strings.TrimSpace(input)
	if end := strings.Index(input, "://"); end > 0 {
		return strings.ToLower(input[:end]) + "://..."
	}
	return "input"
}

func applyAllowInsecure(document *provider.Document) {
	for index := range document.Outbounds {
		wrapper, loaded := document.Outbounds[index].Options.(option.OutboundTLSOptionsWrapper)
		if !loaded {
			continue
		}
		tlsOptions := wrapper.TakeOutboundTLSOptions()
		if tlsOptions != nil && tlsOptions.Enabled {
			tlsOptions.Insecure = true
			wrapper.ReplaceOutboundTLSOptions(tlsOptions)
		}
	}
}

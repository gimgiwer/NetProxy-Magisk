package convert

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"

	"github.com/Fanju6/NetProxy-Magisk/src/native/netproxy/internal/provider"
)

// xrayConfigItem 表示 Xray / Happ / Incy 订阅数组中单个配置项的完整结构。
type xrayConfigItem struct {
	Remarks          string                `json:"remarks,omitempty"`
	Outbounds        []xrayOutbound        `json:"outbounds,omitempty"`
	Routing          *xrayRouting          `json:"routing,omitempty"`
	BurstObservatory *xrayBurstObservatory `json:"burstObservatory,omitempty"`
	Observatory      *xrayBurstObservatory `json:"observatory,omitempty"`
}

// xrayRouting 表示 Xray 路由小节及负载均衡器列表。
type xrayRouting struct {
	DomainStrategy string         `json:"domainStrategy,omitempty"`
	Balancers      []xrayBalancer `json:"balancers,omitempty"`
}

// xrayBalancer 表示单个负载均衡器定义。
type xrayBalancer struct {
	Tag      string        `json:"tag,omitempty"`
	Selector []string      `json:"selector,omitempty"`
	Strategy *xrayStrategy `json:"strategy,omitempty"`
}

// xrayStrategy 表示负载均衡调度策略。
type xrayStrategy struct {
	Type string `json:"type,omitempty"`
}

// xrayBurstObservatory 表示突发延迟观测器设置。
type xrayBurstObservatory struct {
	SubjectSelector []string        `json:"subjectSelector,omitempty"`
	PingConfig      *xrayPingConfig `json:"pingConfig,omitempty"`
}

// xrayPingConfig 表示观测器 Ping 测速参数。
type xrayPingConfig struct {
	Destination  string         `json:"destination,omitempty"`
	Connectivity string         `json:"connectivity,omitempty"`
	Interval     jsontext.Value `json:"interval,omitempty"`
	Timeout      jsontext.Value `json:"timeout,omitempty"`
}

// xrayOutbound 表示 Xray 出站代理或内置服务节点。
type xrayOutbound struct {
	Tag            string              `json:"tag,omitempty"`
	Protocol       string              `json:"protocol,omitempty"`
	Settings       jsontext.Value      `json:"settings,omitempty"`
	StreamSettings *xrayStreamSettings `json:"streamSettings,omitempty"`
}

// xrayStreamSettings 表示底层传输层与 TLS / Reality 安全设置。
type xrayStreamSettings struct {
	Network             string                    `json:"network,omitempty"`
	Security            string                    `json:"security,omitempty"`
	RealitySettings     *xrayRealitySettings     `json:"realitySettings,omitempty"`
	TLSSettings         *xrayTLSSettings         `json:"tlsSettings,omitempty"`
	WSSettings          *xrayWSSettings          `json:"wsSettings,omitempty"`
	GRPCSettings        *xrayGRPCSettings        `json:"grpcSettings,omitempty"`
	HTTPSettings        *xrayHTTPSettings        `json:"httpSettings,omitempty"`
	HTTPUpgradeSettings *xrayHTTPUpgradeSettings `json:"httpupgradeSettings,omitempty"`
	XHTTPSettings       *xrayXHTTPSettings       `json:"xhttpSettings,omitempty"`
}

// xrayRealitySettings 表示 VLESS Reality 扩展配置。
type xrayRealitySettings struct {
	Show        bool   `json:"show,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	ServerName  string `json:"serverName,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
	ShortId     string `json:"shortId,omitempty"`
	SpiderX     string `json:"spiderX,omitempty"`
}

// xrayTLSSettings 表示标准 TLS 传输配置。
type xrayTLSSettings struct {
	ServerName    string   `json:"serverName,omitempty"`
	AllowInsecure bool     `json:"allowInsecure,omitempty"`
	ALPN          []string `json:"alpn,omitempty"`
	Fingerprint   string   `json:"fingerprint,omitempty"`
}

// xrayWSSettings 表示 WebSocket 传输层配置。
type xrayWSSettings struct {
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// xrayGRPCSettings 表示 gRPC 传输层配置。
type xrayGRPCSettings struct {
	ServiceName string `json:"serviceName,omitempty"`
	MultiMode   bool   `json:"multiMode,omitempty"`
}

// xrayHTTPSettings 表示 HTTP/2 传输层配置。
type xrayHTTPSettings struct {
	Path string   `json:"path,omitempty"`
	Host []string `json:"host,omitempty"`
}

// xrayHTTPUpgradeSettings 表示 HTTPUpgrade 传输层配置。
type xrayHTTPUpgradeSettings struct {
	Path string `json:"path,omitempty"`
	Host string `json:"host,omitempty"`
}

// xrayXHTTPSettings 表示 XHTTP 传输层配置。
type xrayXHTTPSettings struct {
	Mode string `json:"mode,omitempty"`
	Path string `json:"path,omitempty"`
	Host string `json:"host,omitempty"`
}

// xrayVLESSSettings 表示 VLESS 协议详细设置。
type xrayVLESSSettings struct {
	Address string            `json:"address,omitempty"`
	Port    uint16            `json:"port,omitempty"`
	UUID    string            `json:"uuid,omitempty"`
	Flow    string            `json:"flow,omitempty"`
	Vnext   []xrayVLESSServer `json:"vnext,omitempty"`
}

type xrayVLESSServer struct {
	Address string          `json:"address,omitempty"`
	Port    uint16          `json:"port,omitempty"`
	Users   []xrayVLESSUser `json:"users,omitempty"`
}

type xrayVLESSUser struct {
	ID         string `json:"id,omitempty"`
	Flow       string `json:"flow,omitempty"`
	Encryption string `json:"encryption,omitempty"`
	Level      int    `json:"level,omitempty"`
}

// xrayHysteriaSettings 表示 Hysteria / Hysteria 2 协议详细设置。
type xrayHysteriaSettings struct {
	Version  int                  `json:"version,omitempty"`
	Address  string               `json:"address,omitempty"`
	Port     uint16               `json:"port,omitempty"`
	Auth     string               `json:"auth,omitempty"`
	Password string               `json:"password,omitempty"`
	Up       jsontext.Value       `json:"up,omitempty"`
	Down     jsontext.Value       `json:"down,omitempty"`
	UpMbps   int                  `json:"up_mbps,omitempty"`
	DownMbps int                  `json:"down_mbps,omitempty"`
	Obfs     *xrayHysteriaObfs    `json:"obfs,omitempty"`
	Servers  []xrayHysteriaServer `json:"servers,omitempty"`
	Vnext    []xrayVLESSServer    `json:"vnext,omitempty"`
}

type xrayHysteriaObfs struct {
	Type     string `json:"type,omitempty"`
	Password string `json:"password,omitempty"`
}

type xrayHysteriaServer struct {
	Address  string            `json:"address,omitempty"`
	Port     uint16            `json:"port,omitempty"`
	Auth     string            `json:"auth,omitempty"`
	Password string            `json:"password,omitempty"`
	Obfs     *xrayHysteriaObfs `json:"obfs,omitempty"`
}

// xrayVMessSettings 表示 VMess 协议设置。
type xrayVMessSettings struct {
	Address string            `json:"address,omitempty"`
	Port    uint16            `json:"port,omitempty"`
	UUID    string            `json:"uuid,omitempty"`
	AlterId int               `json:"alterId,omitempty"`
	Vnext   []xrayVMessServer `json:"vnext,omitempty"`
}

type xrayVMessServer struct {
	Address string          `json:"address,omitempty"`
	Port    uint16          `json:"port,omitempty"`
	Users   []xrayVMessUser `json:"users,omitempty"`
}

type xrayVMessUser struct {
	ID       string `json:"id,omitempty"`
	AlterId  int    `json:"alterId,omitempty"`
	Security string `json:"security,omitempty"`
}

// xrayTrojanSettings 表示 Trojan 协议设置。
type xrayTrojanSettings struct {
	Address  string             `json:"address,omitempty"`
	Port     uint16             `json:"port,omitempty"`
	Password string             `json:"password,omitempty"`
	Servers  []xrayTrojanServer `json:"servers,omitempty"`
}

type xrayTrojanServer struct {
	Address  string `json:"address,omitempty"`
	Port     uint16 `json:"port,omitempty"`
	Password string `json:"password,omitempty"`
}

// xrayShadowsocksSettings 表示 Shadowsocks 协议设置。
type xrayShadowsocksSettings struct {
	Address  string                  `json:"address,omitempty"`
	Port     uint16                  `json:"port,omitempty"`
	Method   string                  `json:"method,omitempty"`
	Password string                  `json:"password,omitempty"`
	Servers  []xrayShadowsocksServer `json:"servers,omitempty"`
}

type xrayShadowsocksServer struct {
	Address  string `json:"address,omitempty"`
	Port     uint16 `json:"port,omitempty"`
	Method   string `json:"method,omitempty"`
	Password string `json:"password,omitempty"`
}

// ParseXrayJSON 解析 Xray / Happ / Incy JSON 订阅或配置文本，支持配置数组或单项配置，
// 将各协议节点及负载均衡器（URLTest）转换为 sing-box 节点文档。
func ParseXrayJSON(ctx context.Context, content string) (provider.Document, []provider.Diagnostic, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		diag := provider.Diagnostic{Code: "xray.empty", Message: "empty content"}
		return provider.Document{}, []provider.Diagnostic{diag}, &DiagnosticsError{Diagnostics: []provider.Diagnostic{diag}}
	}

	var items []xrayConfigItem
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &items); err != nil {
			diag := provider.Diagnostic{Code: "xray.invalid", Message: fmt.Sprintf("failed to parse Xray JSON array: %v", err)}
			return provider.Document{}, []provider.Diagnostic{diag}, &DiagnosticsError{Diagnostics: []provider.Diagnostic{diag}}
		}
	} else if strings.HasPrefix(trimmed, "{") {
		var single xrayConfigItem
		if err := json.Unmarshal([]byte(trimmed), &single); err != nil {
			diag := provider.Diagnostic{Code: "xray.invalid", Message: fmt.Sprintf("failed to parse Xray JSON object: %v", err)}
			return provider.Document{}, []provider.Diagnostic{diag}, &DiagnosticsError{Diagnostics: []provider.Diagnostic{diag}}
		}
		items = []xrayConfigItem{single}
	} else {
		diag := provider.Diagnostic{Code: "xray.invalid", Message: "content is neither a JSON array nor a JSON object"}
		return provider.Document{}, []provider.Diagnostic{diag}, &DiagnosticsError{Diagnostics: []provider.Diagnostic{diag}}
	}

	var document provider.Document
	var diagnostics []provider.Diagnostic
	usedTags := make(map[string]int)

	type convertedItemNode struct {
		outbound    option.Outbound
		originalTag string
	}

	for itemIndex, item := range items {
		remarks := strings.TrimSpace(item.Remarks)

		// 收集该配置项内全部合法代理出站，过滤内置服务出站（freedom / blackhole 等）
		var itemNodes []convertedItemNode
		for outboundIndex, ob := range item.Outbounds {
			proto := strings.ToLower(strings.TrimSpace(ob.Protocol))
			if isServiceOutbound(proto) {
				continue
			}

			parsedOutbound, err := convertXrayOutbound(ob)
			if err != nil {
				diagnostics = append(diagnostics, provider.Diagnostic{
					Index:   itemIndex*100 + outboundIndex + 1,
					Source:  firstNonEmpty(ob.Tag, remarks, fmt.Sprintf("item_%d_node_%d", itemIndex+1, outboundIndex+1)),
					Code:    "xray.node_invalid",
					Message: err.Error(),
				})
				continue
			}

			itemNodes = append(itemNodes, convertedItemNode{
				outbound:    parsedOutbound,
				originalTag: strings.TrimSpace(ob.Tag),
			})
		}

		if len(itemNodes) == 0 {
			continue
		}

		// 判断当前配置项是否属于集群 / 均衡器配置
		hasBalancers := item.Routing != nil && len(item.Routing.Balancers) > 0
		hasObservatory := (item.BurstObservatory != nil && len(item.BurstObservatory.SubjectSelector) > 0) ||
			(item.Observatory != nil && len(item.Observatory.SubjectSelector) > 0)
		isGroup := hasBalancers || hasObservatory || len(itemNodes) > 1

		if !isGroup {
			// 单节点模式：直接应用 remarks 作为节点标签
			node := itemNodes[0].outbound
			if remarks != "" {
				node.Tag = remarks
			} else if node.Tag == "" || node.Tag == "proxy" {
				node.Tag = defaultTagForOutbound(node)
			}
			node.Tag = ensureUniqueTag(node.Tag, usedTags)
			document.Outbounds = append(document.Outbounds, node)
		} else {
			// 集群 / 均衡器模式：
			// 1. 先为所有实际服务器节点分配可读标签并加入文档
			for i := range itemNodes {
				node := &itemNodes[i].outbound
				orig := itemNodes[i].originalTag
				var nodeTag string
				if orig != "" && orig != "proxy" && orig != "node" {
					if remarks != "" {
						nodeTag = fmt.Sprintf("%s - %s", remarks, orig)
					} else {
						nodeTag = orig
					}
				} else {
					if remarks != "" {
						nodeTag = fmt.Sprintf("%s #%d", remarks, i+1)
					} else {
						nodeTag = fmt.Sprintf("node_%d", i+1)
					}
				}
				node.Tag = ensureUniqueTag(nodeTag, usedTags)
				document.Outbounds = append(document.Outbounds, *node)
			}

			// 2. 提取测速地址与探针间隔
			testURL := "https://www.gstatic.com/generate_204"
			interval := badoption.Duration(3 * time.Minute)
			if obs := getObservatory(item); obs != nil && obs.PingConfig != nil {
				if obs.PingConfig.Destination != "" {
					testURL = obs.PingConfig.Destination
				} else if obs.PingConfig.Connectivity != "" {
					testURL = obs.PingConfig.Connectivity
				}
				if dur := parseDurationVal(obs.PingConfig.Interval); dur > 0 {
					interval = badoption.Duration(dur)
				}
			}

			// 3. 构建 URLTest 负载均衡出站
			if hasBalancers {
				for _, b := range item.Routing.Balancers {
					var selectedMemberTags []string
					for _, n := range itemNodes {
						if selectorMatches(n.originalTag, b.Selector) {
							selectedMemberTags = append(selectedMemberTags, n.outbound.Tag)
						}
					}
					if len(selectedMemberTags) == 0 {
						for _, n := range itemNodes {
							selectedMemberTags = append(selectedMemberTags, n.outbound.Tag)
						}
					}
					var balancerTag string
					if remarks != "" {
						if len(item.Routing.Balancers) == 1 {
							balancerTag = remarks
						} else {
							balancerTag = fmt.Sprintf("%s - %s", remarks, b.Tag)
						}
					} else if b.Tag != "" {
						balancerTag = b.Tag
					} else {
						balancerTag = "Auto"
					}
					balancerTag = ensureUniqueTag(balancerTag, usedTags)
					urlTestOutbound := option.Outbound{
						Type: C.TypeURLTest,
						Tag:  balancerTag,
						Options: &option.URLTestOutboundOptions{
							GroupCommonOption: option.GroupCommonOption{
								Outbounds: selectedMemberTags,
							},
							URL:       testURL,
							Interval:  interval,
							Tolerance: 50,
						},
					}
					document.Outbounds = append(document.Outbounds, urlTestOutbound)
				}
			} else {
				// 未显式声明 balancers 但存在多个节点或测速观测器
				var memberTags []string
				for _, n := range itemNodes {
					memberTags = append(memberTags, n.outbound.Tag)
				}
				balancerTag := remarks
				if balancerTag == "" {
					balancerTag = "Auto"
				}
				balancerTag = ensureUniqueTag(balancerTag, usedTags)
				urlTestOutbound := option.Outbound{
					Type: C.TypeURLTest,
					Tag:  balancerTag,
					Options: &option.URLTestOutboundOptions{
						GroupCommonOption: option.GroupCommonOption{
							Outbounds: memberTags,
						},
						URL:       testURL,
						Interval:  interval,
						Tolerance: 50,
					},
				}
				document.Outbounds = append(document.Outbounds, urlTestOutbound)
			}
		}
	}

	if len(document.Outbounds) == 0 {
		if len(diagnostics) == 0 {
			diagnostics = append(diagnostics, provider.Diagnostic{Code: "xray.no_nodes", Message: "no supported proxy nodes found in Xray JSON"})
		}
		return provider.Document{}, diagnostics, &DiagnosticsError{Diagnostics: diagnostics}
	}

	return document, diagnostics, nil
}

// isServiceOutbound 检查协议是否为内置非代理出站。
func isServiceOutbound(proto string) bool {
	switch proto {
	case "freedom", "blackhole", "direct", "block", "dns", "loopback", "":
		return true
	default:
		return false
	}
}

// convertXrayOutbound 将单个 Xray 出站对象转换为 sing-box 出站结构。
func convertXrayOutbound(ob xrayOutbound) (option.Outbound, error) {
	proto := strings.ToLower(strings.TrimSpace(ob.Protocol))
	tag := strings.TrimSpace(ob.Tag)

	switch proto {
	case "vless":
		return convertVLESS(ob, tag)
	case "hysteria", "hysteria2", "hy2":
		return convertHysteria(ob, tag)
	case "vmess":
		return convertVMess(ob, tag)
	case "trojan":
		return convertTrojan(ob, tag)
	case "shadowsocks":
		return convertShadowsocks(ob, tag)
	default:
		return option.Outbound{}, fmt.Errorf("unsupported protocol %q", ob.Protocol)
	}
}

// convertVLESS 转换 Xray VLESS 出站为 sing-box VLESS 节点。
func convertVLESS(ob xrayOutbound, tag string) (option.Outbound, error) {
	var settings xrayVLESSSettings
	if len(ob.Settings) > 0 {
		if err := json.Unmarshal(ob.Settings, &settings); err != nil {
			return option.Outbound{}, fmt.Errorf("invalid VLESS settings: %w", err)
		}
	}

	server := settings.Address
	serverPort := settings.Port
	uuid := settings.UUID
	flow := settings.Flow

	if len(settings.Vnext) > 0 {
		v := settings.Vnext[0]
		if v.Address != "" {
			server = v.Address
		}
		if v.Port != 0 {
			serverPort = v.Port
		}
		if len(v.Users) > 0 {
			u := v.Users[0]
			if u.ID != "" {
				uuid = u.ID
			}
			if u.Flow != "" {
				flow = u.Flow
			}
		}
	}

	server = strings.TrimSpace(server)
	if server == "" {
		return option.Outbound{}, errors.New("missing VLESS server address")
	}
	if serverPort == 0 {
		return option.Outbound{}, errors.New("missing VLESS server port")
	}
	if uuid == "" {
		return option.Outbound{}, errors.New("missing VLESS user UUID")
	}

	tlsOptions, err := buildTLSOptions(ob.StreamSettings, server)
	if err != nil {
		return option.Outbound{}, err
	}

	transport, err := buildTransportOptions(ob.StreamSettings)
	if err != nil {
		return option.Outbound{}, err
	}

	vlessOptions := &option.VLESSOutboundOptions{
		ServerOptions: option.ServerOptions{
			Server:     server,
			ServerPort: serverPort,
		},
		UUID: uuid,
		Flow: flow,
	}
	if tlsOptions != nil {
		vlessOptions.TLS = tlsOptions
	}
	if transport != nil {
		vlessOptions.Transport = transport
	}

	if tag == "" {
		tag = fmt.Sprintf("vless-%s:%d", server, serverPort)
	}

	return option.Outbound{
		Type:    C.TypeVLESS,
		Tag:     tag,
		Options: vlessOptions,
	}, nil
}

// convertHysteria 转换 Xray Hysteria / Hysteria 2 出站为 sing-box Hysteria 2 节点。
func convertHysteria(ob xrayOutbound, tag string) (option.Outbound, error) {
	var settings xrayHysteriaSettings
	if len(ob.Settings) > 0 {
		if err := json.Unmarshal(ob.Settings, &settings); err != nil {
			return option.Outbound{}, fmt.Errorf("invalid Hysteria settings: %w", err)
		}
	}

	server := settings.Address
	serverPort := settings.Port
	password := firstNonEmpty(settings.Password, settings.Auth)

	if len(settings.Servers) > 0 {
		s := settings.Servers[0]
		if s.Address != "" {
			server = s.Address
		}
		if s.Port != 0 {
			serverPort = s.Port
		}
		if p := firstNonEmpty(s.Password, s.Auth); p != "" {
			password = p
		}
	}

	if len(settings.Vnext) > 0 {
		v := settings.Vnext[0]
		if v.Address != "" {
			server = v.Address
		}
		if v.Port != 0 {
			serverPort = v.Port
		}
		if len(v.Users) > 0 && v.Users[0].ID != "" {
			password = v.Users[0].ID
		}
	}

	server = strings.TrimSpace(server)
	if server == "" {
		return option.Outbound{}, errors.New("missing Hysteria server address")
	}
	if serverPort == 0 {
		return option.Outbound{}, errors.New("missing Hysteria server port")
	}

	// 混淆设置
	var h2Obfs *option.Hysteria2Obfs
	if settings.Obfs != nil && settings.Obfs.Password != "" {
		obfsType := settings.Obfs.Type
		if obfsType == "" {
			obfsType = "salamander"
		}
		h2Obfs = &option.Hysteria2Obfs{
			Type:     obfsType,
			Password: settings.Obfs.Password,
		}
	} else if len(settings.Servers) > 0 && settings.Servers[0].Obfs != nil && settings.Servers[0].Obfs.Password != "" {
		obfsType := settings.Servers[0].Obfs.Type
		if obfsType == "" {
			obfsType = "salamander"
		}
		h2Obfs = &option.Hysteria2Obfs{
			Type:     obfsType,
			Password: settings.Servers[0].Obfs.Password,
		}
	}

	upMbps := settings.UpMbps
	if upMbps == 0 {
		upMbps = parseMbps(settings.Up)
	}
	downMbps := settings.DownMbps
	if downMbps == 0 {
		downMbps = parseMbps(settings.Down)
	}

	tlsOptions, err := buildTLSOptions(ob.StreamSettings, server)
	if err != nil {
		return option.Outbound{}, err
	}
	if tlsOptions == nil {
		tlsOptions = &option.OutboundTLSOptions{
			Enabled:    true,
			ServerName: server,
		}
	}

	hy2Options := &option.Hysteria2OutboundOptions{
		ServerOptions: option.ServerOptions{
			Server:     server,
			ServerPort: serverPort,
		},
		Password: password,
		UpMbps:   upMbps,
		DownMbps: downMbps,
		Obfs:     h2Obfs,
		OutboundTLSOptionsContainer: option.OutboundTLSOptionsContainer{
			TLS: tlsOptions,
		},
	}

	if tag == "" {
		tag = fmt.Sprintf("hy2-%s:%d", server, serverPort)
	}

	return option.Outbound{
		Type:    C.TypeHysteria2,
		Tag:     tag,
		Options: hy2Options,
	}, nil
}

// convertVMess 转换 Xray VMess 出站为 sing-box VMess 节点。
func convertVMess(ob xrayOutbound, tag string) (option.Outbound, error) {
	var settings xrayVMessSettings
	if len(ob.Settings) > 0 {
		if err := json.Unmarshal(ob.Settings, &settings); err != nil {
			return option.Outbound{}, fmt.Errorf("invalid VMess settings: %w", err)
		}
	}

	server := settings.Address
	serverPort := settings.Port
	uuid := settings.UUID
	alterId := settings.AlterId
	security := "auto"

	if len(settings.Vnext) > 0 {
		v := settings.Vnext[0]
		if v.Address != "" {
			server = v.Address
		}
		if v.Port != 0 {
			serverPort = v.Port
		}
		if len(v.Users) > 0 {
			u := v.Users[0]
			if u.ID != "" {
				uuid = u.ID
			}
			if u.AlterId != 0 {
				alterId = u.AlterId
			}
			if u.Security != "" {
				security = u.Security
			}
		}
	}

	server = strings.TrimSpace(server)
	if server == "" {
		return option.Outbound{}, errors.New("missing VMess server address")
	}
	if serverPort == 0 {
		return option.Outbound{}, errors.New("missing VMess server port")
	}
	if uuid == "" {
		return option.Outbound{}, errors.New("missing VMess user UUID")
	}

	tlsOptions, err := buildTLSOptions(ob.StreamSettings, server)
	if err != nil {
		return option.Outbound{}, err
	}
	transport, err := buildTransportOptions(ob.StreamSettings)
	if err != nil {
		return option.Outbound{}, err
	}

	vmessOptions := &option.VMessOutboundOptions{
		ServerOptions: option.ServerOptions{
			Server:     server,
			ServerPort: serverPort,
		},
		UUID:      uuid,
		AlterId:   alterId,
		Security:  security,
		TLS:       tlsOptions,
		Transport: transport,
	}

	if tag == "" {
		tag = fmt.Sprintf("vmess-%s:%d", server, serverPort)
	}

	return option.Outbound{
		Type:    C.TypeVMess,
		Tag:     tag,
		Options: vmessOptions,
	}, nil
}

// convertTrojan 转换 Xray Trojan 出站为 sing-box Trojan 节点。
func convertTrojan(ob xrayOutbound, tag string) (option.Outbound, error) {
	var settings xrayTrojanSettings
	if len(ob.Settings) > 0 {
		if err := json.Unmarshal(ob.Settings, &settings); err != nil {
			return option.Outbound{}, fmt.Errorf("invalid Trojan settings: %w", err)
		}
	}

	server := settings.Address
	serverPort := settings.Port
	password := settings.Password

	if len(settings.Servers) > 0 {
		s := settings.Servers[0]
		if s.Address != "" {
			server = s.Address
		}
		if s.Port != 0 {
			serverPort = s.Port
		}
		if s.Password != "" {
			password = s.Password
		}
	}

	server = strings.TrimSpace(server)
	if server == "" {
		return option.Outbound{}, errors.New("missing Trojan server address")
	}
	if serverPort == 0 {
		return option.Outbound{}, errors.New("missing Trojan server port")
	}
	if password == "" {
		return option.Outbound{}, errors.New("missing Trojan password")
	}

	tlsOptions, err := buildTLSOptions(ob.StreamSettings, server)
	if err != nil {
		return option.Outbound{}, err
	}
	if tlsOptions == nil {
		tlsOptions = &option.OutboundTLSOptions{
			Enabled:    true,
			ServerName: server,
		}
	}

	transport, err := buildTransportOptions(ob.StreamSettings)
	if err != nil {
		return option.Outbound{}, err
	}

	trojanOptions := &option.TrojanOutboundOptions{
		ServerOptions: option.ServerOptions{
			Server:     server,
			ServerPort: serverPort,
		},
		Password:  password,
		TLS:       tlsOptions,
		Transport: transport,
	}

	if tag == "" {
		tag = fmt.Sprintf("trojan-%s:%d", server, serverPort)
	}

	return option.Outbound{
		Type:    C.TypeTrojan,
		Tag:     tag,
		Options: trojanOptions,
	}, nil
}

// convertShadowsocks 转换 Xray Shadowsocks 出站为 sing-box Shadowsocks 节点。
func convertShadowsocks(ob xrayOutbound, tag string) (option.Outbound, error) {
	var settings xrayShadowsocksSettings
	if len(ob.Settings) > 0 {
		if err := json.Unmarshal(ob.Settings, &settings); err != nil {
			return option.Outbound{}, fmt.Errorf("invalid Shadowsocks settings: %w", err)
		}
	}

	server := settings.Address
	serverPort := settings.Port
	method := settings.Method
	password := settings.Password

	if len(settings.Servers) > 0 {
		s := settings.Servers[0]
		if s.Address != "" {
			server = s.Address
		}
		if s.Port != 0 {
			serverPort = s.Port
		}
		if s.Method != "" {
			method = s.Method
		}
		if s.Password != "" {
			password = s.Password
		}
	}

	server = strings.TrimSpace(server)
	if server == "" {
		return option.Outbound{}, errors.New("missing Shadowsocks server address")
	}
	if serverPort == 0 {
		return option.Outbound{}, errors.New("missing Shadowsocks server port")
	}
	if method == "" || password == "" {
		return option.Outbound{}, errors.New("missing Shadowsocks method or password")
	}

	ssOptions := &option.ShadowsocksOutboundOptions{
		ServerOptions: option.ServerOptions{
			Server:     server,
			ServerPort: serverPort,
		},
		Method:   method,
		Password: password,
	}

	if tag == "" {
		tag = fmt.Sprintf("ss-%s:%d", server, serverPort)
	}

	return option.Outbound{
		Type:    C.TypeShadowsocks,
		Tag:     tag,
		Options: ssOptions,
	}, nil
}

// buildTLSOptions 根据 streamSettings 构建 sing-box TLS / Reality 选项。
func buildTLSOptions(stream *xrayStreamSettings, defaultServer string) (*option.OutboundTLSOptions, error) {
	if stream == nil {
		return nil, nil
	}
	sec := strings.ToLower(stream.Security)
	if sec == "" {
		if stream.RealitySettings != nil {
			sec = "reality"
		} else if stream.TLSSettings != nil {
			sec = "tls"
		} else {
			return nil, nil
		}
	}
	if sec == "none" {
		return nil, nil
	}

	if sec == "reality" || stream.RealitySettings != nil {
		reality := stream.RealitySettings
		if reality == nil {
			reality = &xrayRealitySettings{}
		}
		serverName := firstNonEmpty(reality.ServerName, defaultServer)
		if stream.TLSSettings != nil && reality.ServerName == "" {
			serverName = firstNonEmpty(stream.TLSSettings.ServerName, defaultServer)
		}
		fp := reality.Fingerprint
		if fp == "" && stream.TLSSettings != nil {
			fp = stream.TLSSettings.Fingerprint
		}

		tlsOpts := &option.OutboundTLSOptions{
			Enabled:    true,
			ServerName: serverName,
			Reality: &option.OutboundRealityOptions{
				Enabled:   true,
				PublicKey: reality.PublicKey,
				ShortID:   reality.ShortId,
			},
		}
		if fp != "" {
			tlsOpts.UTLS = &option.OutboundUTLSOptions{
				Enabled:     true,
				Fingerprint: fp,
			}
		}
		return tlsOpts, nil
	}

	if sec == "tls" || stream.TLSSettings != nil {
		tls := stream.TLSSettings
		if tls == nil {
			tls = &xrayTLSSettings{}
		}
		serverName := firstNonEmpty(tls.ServerName, defaultServer)
		tlsOpts := &option.OutboundTLSOptions{
			Enabled:    true,
			ServerName: serverName,
			Insecure:   tls.AllowInsecure,
		}
		if len(tls.ALPN) > 0 {
			tlsOpts.ALPN = badoption.Listable[string](tls.ALPN)
		}
		if tls.Fingerprint != "" {
			tlsOpts.UTLS = &option.OutboundUTLSOptions{
				Enabled:     true,
				Fingerprint: tls.Fingerprint,
			}
		}
		return tlsOpts, nil
	}

	return nil, nil
}

// buildTransportOptions 根据 streamSettings 构建 sing-box 传输层选项。
func buildTransportOptions(stream *xrayStreamSettings) (*option.V2RayTransportOptions, error) {
	if stream == nil {
		return nil, nil
	}
	netType := strings.ToLower(stream.Network)
	switch netType {
	case "ws", "websocket":
		ws := stream.WSSettings
		if ws == nil {
			ws = &xrayWSSettings{}
		}
		var wsHeaders badoption.HTTPHeader
		if len(ws.Headers) > 0 {
			wsHeaders = make(badoption.HTTPHeader, len(ws.Headers))
			for k, v := range ws.Headers {
				wsHeaders[k] = badoption.Listable[string]{v}
			}
		}
		return &option.V2RayTransportOptions{
			Type: C.V2RayTransportTypeWebsocket,
			WebsocketOptions: option.V2RayWebsocketOptions{
				Path:    ws.Path,
				Headers: wsHeaders,
			},
		}, nil
	case "grpc":
		grpc := stream.GRPCSettings
		if grpc == nil {
			grpc = &xrayGRPCSettings{}
		}
		return &option.V2RayTransportOptions{
			Type: C.V2RayTransportTypeGRPC,
			GRPCOptions: option.V2RayGRPCOptions{
				ServiceName: grpc.ServiceName,
			},
		}, nil
	case "http":
		httpOpt := stream.HTTPSettings
		if httpOpt == nil {
			httpOpt = &xrayHTTPSettings{}
		}
		return &option.V2RayTransportOptions{
			Type: C.V2RayTransportTypeHTTP,
			HTTPOptions: option.V2RayHTTPOptions{
				Path: httpOpt.Path,
				Host: badoption.Listable[string](httpOpt.Host),
			},
		}, nil
	case "httpupgrade":
		hu := stream.HTTPUpgradeSettings
		if hu == nil {
			hu = &xrayHTTPUpgradeSettings{}
		}
		return &option.V2RayTransportOptions{
			Type: C.V2RayTransportTypeHTTPUpgrade,
			HTTPUpgradeOptions: option.V2RayHTTPUpgradeOptions{
				Path: hu.Path,
				Host: hu.Host,
			},
		}, nil
	case "xhttp":
		xh := stream.XHTTPSettings
		if xh == nil {
			xh = &xrayXHTTPSettings{}
		}
		var hosts []string
		if xh.Host != "" {
			hosts = []string{xh.Host}
		}
		return &option.V2RayTransportOptions{
			Type: C.V2RayTransportTypeHTTP,
			HTTPOptions: option.V2RayHTTPOptions{
				Path: xh.Path,
				Host: badoption.Listable[string](hosts),
			},
		}, nil
	default:
		return nil, nil
	}
}

// getObservatory 获取配置项中的突发观测器或普通观测器。
func getObservatory(item xrayConfigItem) *xrayBurstObservatory {
	if item.BurstObservatory != nil {
		return item.BurstObservatory
	}
	return item.Observatory
}

// selectorMatches 检查节点标签是否命中负载均衡器选择器规则。
func selectorMatches(tag string, selectors []string) bool {
	if len(selectors) == 0 {
		return true
	}
	if tag == "" {
		return false
	}
	for _, sel := range selectors {
		sel = strings.TrimSpace(sel)
		if sel == "" {
			continue
		}
		if sel == tag || strings.HasPrefix(tag, sel) || strings.Contains(tag, sel) {
			return true
		}
	}
	return false
}

// parseMbps 解析宽带字符串（如 "100 Mbps", "50m"）或纯数字为整数 Mbps。
func parseMbps(val jsontext.Value) int {
	if len(val) == 0 {
		return 0
	}
	s := strings.Trim(string(val), `"`)
	s = strings.TrimSpace(s)
	if s == "" || s == "null" {
		return 0
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	lower := strings.ToLower(s)
	if strings.Contains(lower, "kbps") || strings.Contains(lower, "k") {
		numStr := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(lower, "kbps", ""), "k", ""))
		if f, err := strconv.ParseFloat(numStr, 64); err == nil {
			return int(f / 1000)
		}
	}
	numStr := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(lower, "mbps", ""), "m", ""))
	if f, err := strconv.ParseFloat(numStr, 64); err == nil {
		return int(f)
	}
	return 0
}

// parseDurationVal 解析时长字段（如 "10s", "3m"）为 time.Duration。
func parseDurationVal(val jsontext.Value) time.Duration {
	if len(val) == 0 {
		return 0
	}
	s := strings.Trim(string(val), `"`)
	s = strings.TrimSpace(s)
	if s == "" || s == "null" {
		return 0
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Duration(n) * time.Second
	}
	return 0
}

// ensureUniqueTag 确保生成的节点标签在当前文档内全局唯一。
func ensureUniqueTag(tag string, used map[string]int) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		tag = "node"
	}
	used[tag]++
	if used[tag] == 1 {
		return tag
	}
	base := tag
	for suffix := used[tag]; ; suffix++ {
		candidate := fmt.Sprintf("%s_%d", base, suffix)
		if used[candidate] == 0 {
			used[candidate] = 1
			return candidate
		}
	}
}

// splitHostPort 安全拆分主机与端口。
func splitHostPort(addr string, defaultPort uint16) (string, uint16) {
	addr = strings.TrimSpace(addr)
	if h, p, err := net.SplitHostPort(addr); err == nil {
		if portNum, err := strconv.ParseUint(p, 10, 16); err == nil && portNum > 0 {
			return h, uint16(portNum)
		}
		return h, defaultPort
	}
	return addr, defaultPort
}

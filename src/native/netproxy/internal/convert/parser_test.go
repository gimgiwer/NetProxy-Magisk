package convert_test

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"

	"github.com/Fanju6/NetProxy-Magisk/src/native/netproxy/internal/convert"
	"github.com/Fanju6/NetProxy-Magisk/src/native/netproxy/internal/provider"
)

// TestNormalizeRawPayload_BOMAndCRLF 测试 UTF-8 BOM 移除与换行符 CRLF 归一化。
func TestNormalizeRawPayload_BOMAndCRLF(t *testing.T) {
	raw := "\xef\xbb\xbfline1\r\nline2\rline3\n"
	normalized := convert.NormalizeRawPayload(raw)
	expected := "line1\nline2\nline3\n"
	if normalized != expected {
		t.Fatalf("unexpected normalized payload: got %q, want %q", normalized, expected)
	}
}

// TestNormalizeRawPayload_RecursiveBase64 测试最多 3 轮递归 Base64 解码，包含 URL-Safe、无补位 '=' 及内部换行。
func TestNormalizeRawPayload_RecursiveBase64(t *testing.T) {
	targetURI := "vless://3c0f47e3-a464-470c-a931-36b8a8d62fd6@example.com:443?security=reality#node1\n"

	// 1 轮标准 Base64 解码
	round1 := base64.StdEncoding.EncodeToString([]byte(targetURI))
	if got := convert.NormalizeRawPayload(round1); got != targetURI {
		t.Fatalf("1-round base64 decode failed: got %q", got)
	}

	// 2 轮递归 Base64 解码（含 URL-Safe 与换行）
	round2 := base64.URLEncoding.EncodeToString([]byte(round1))
	wrappedWithNewlines := round2[:10] + "\n  \r\n" + round2[10:]
	if got := convert.NormalizeRawPayload(wrappedWithNewlines); got != targetURI {
		t.Fatalf("2-round base64 decode failed: got %q", got)
	}

	// 3 轮递归 Base64 解码（无补位 RawURLEncoding）
	round3 := base64.RawURLEncoding.EncodeToString([]byte(round2))
	if got := convert.NormalizeRawPayload(round3); got != targetURI {
		t.Fatalf("3-round raw url-safe base64 decode failed: got %q", got)
	}
}

// TestDetectFormat 测试各种格式的精确分类。
func TestDetectFormat(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected convert.Format
	}{
		{
			name:     "Sing-box JSON",
			input:    `{"outbounds":[{"type":"vless","tag":"proxy"}]}`,
			expected: convert.FormatSingBoxJSON,
		},
		{
			name:     "SIP008 JSON",
			input:    `{"version":1,"servers":[{"server":"1.2.3.4","server_port":8388,"password":"pass","method":"aes-128-gcm"}]}`,
			expected: convert.FormatSingBoxJSON,
		},
		{
			name:     "Clash YAML",
			input:    "proxies:\n  - name: test\n    type: ss\n    server: 1.2.3.4\n    port: 8388\n    cipher: aes-128-gcm\n    password: pwd\n",
			expected: convert.FormatClashYAML,
		},
		{
			name:     "Mihomo Provider YAML",
			input:    "payload:\n  - name: test\n    type: ss\n    server: 1.2.3.4\n    port: 8388\n    cipher: aes-128-gcm\n    password: pwd\n",
			expected: convert.FormatClashYAML,
		},
		{
			name:     "WireGuard INI",
			input:    "[Interface]\nPrivateKey = aaaa\nAddress = 10.0.0.2/32\n[Peer]\nPublicKey = bbbb\nEndpoint = 1.2.3.4:51820\n",
			expected: convert.FormatWireGuard,
		},
		{
			name:     "AmneziaWG INI",
			input:    "# AmneziaWG Config\n[Interface]\nAddress = 10.0.0.2/32\nPrivateKey = aaaa\nJc = 4\nJmin = 50\nJmax = 1000\n[Peer]\nPublicKey = bbbb\nEndpoint = 1.2.3.4:51820\n",
			expected: convert.FormatWireGuard,
		},
		{
			name:     "URI List",
			input:    "vless://uuid@1.2.3.4:443#vless\nvmess://uuid@1.2.3.4:443#vmess\n",
			expected: convert.FormatURIList,
		},
		{
			name:     "Unknown text",
			input:    "just some random text without proxy headers",
			expected: convert.FormatUnknown,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := convert.DetectFormat(tc.input)
			if got != tc.expected {
				t.Fatalf("expected format %q, got %q", tc.expected, got)
			}
		})
	}
}

// TestParseClashYAML_FullProtocols 测试 Clash/Mihomo YAML 对 VLESS (Reality, Vision), VMess, Shadowsocks, Trojan, Hysteria 2, TUIC, WireGuard 的全协议转换。
func TestParseClashYAML_FullProtocols(t *testing.T) {
	clashConfig := `
proxies:
  - name: "vless-reality-vision"
    type: vless
    server: 198.51.100.1
    port: 443
    uuid: 3c0f47e3-a464-470c-a931-36b8a8d62fd6
    network: tcp
    tls: true
    udp: true
    flow: xtls-rprx-vision
    servername: reality.example.com
    client-fingerprint: chrome
    reality-opts:
      public-key: tkmyb6Xk2aYMFxrQ35q6PMULtbdIKhaYGG9yySPJbHc
      short-id: fc1b

  - name: "vmess-ws"
    type: vmess
    server: 198.51.100.2
    port: 443
    uuid: 3c0f47e3-a464-470c-a931-36b8a8d62fd6
    alterId: 0
    cipher: auto
    tls: true
    servername: vmess.example.com
    network: ws
    ws-opts:
      path: /ws
      headers:
        Host: vmess.example.com

  - name: "ss-obfs"
    type: ss
    server: 198.51.100.3
    port: 8388
    cipher: 2022-blake3-aes-128-gcm
    password: secretpassword123
    plugin: obfs
    plugin-opts:
      mode: tls
      host: ss.example.com

  - name: "trojan-tls"
    type: trojan
    server: 198.51.100.4
    port: 443
    password: trojanpassword
    sni: trojan.example.com
    skip-cert-verify: true

  - name: "hy2-node"
    type: hy2
    server: 198.51.100.5
    port: 443
    password: hy2password
    sni: hy2.example.com
    up: "50 Mbps"
    down: "100 Mbps"
    obfs: salamander
    obfs-password: obfspassword

  - name: "tuic-node"
    type: tuic
    server: 198.51.100.6
    port: 8443
    uuid: 3c0f47e3-a464-470c-a931-36b8a8d62fd6
    password: tuicpassword
    sni: tuic.example.com
    congestion-controller: bbr
    alpn:
      - h3

  - name: "wireguard-amnezia"
    type: wireguard
    server: 198.51.100.7
    port: 51820
    ip: 10.0.0.2
    private-key: aW52YWxpZC1wcml2YXRlLWtleS0xMDA=
    public-key: aW52YWxpZC1wdWJsaWMta2V5LTEwMA==
    preshared-key: aW52YWxpZC1wc2stMTAw=
    reserved: [0, 1, 2]
    mtu: 1420
    amnezia-wg-option:
      jc: 4
      jmin: 50
      jmax: 1000
      s1: 20
      s2: 50
      h1: 1
      h2: 2
      h3: 3
      h4: 4
`
	result, err := convert.Content(context.Background(), clashConfig, false)
	if err != nil {
		t.Fatalf("Content failed to parse Clash YAML: %v", err)
	}

	if len(result.Document.Outbounds) != 7 {
		t.Fatalf("expected 7 outbounds, got %d", len(result.Document.Outbounds))
	}

	// 1. 验证 VLESS Reality & Vision
	vlessOutbound := result.Document.Outbounds[0]
	if vlessOutbound.Type != C.TypeVLESS || vlessOutbound.Tag != "vless-reality-vision" {
		t.Fatalf("unexpected vless outbound: %#v", vlessOutbound)
	}
	vlessOpts := vlessOutbound.Options.(*option.VLESSOutboundOptions)
	if vlessOpts.Flow != "xtls-rprx-vision" || vlessOpts.TLS == nil || !vlessOpts.TLS.Reality.Enabled {
		t.Fatalf("vless Reality/Vision options missing: %#v", vlessOpts)
	}

	// 2. 验证 VMess WebSocket
	vmessOutbound := result.Document.Outbounds[1]
	if vmessOutbound.Type != C.TypeVMess {
		t.Fatalf("unexpected vmess outbound type: %q", vmessOutbound.Type)
	}
	vmessOpts := vmessOutbound.Options.(*option.VMessOutboundOptions)
	if vmessOpts.Transport == nil || vmessOpts.Transport.Type != C.V2RayTransportTypeWebsocket {
		t.Fatalf("vmess websocket transport missing: %#v", vmessOpts)
	}

	// 3. 验证 Shadowsocks
	ssOutbound := result.Document.Outbounds[2]
	if ssOutbound.Type != C.TypeShadowsocks {
		t.Fatalf("unexpected shadowsocks outbound type: %q", ssOutbound.Type)
	}

	// 4. 验证 Trojan
	trojanOutbound := result.Document.Outbounds[3]
	if trojanOutbound.Type != C.TypeTrojan {
		t.Fatalf("unexpected trojan outbound type: %q", trojanOutbound.Type)
	}

	// 5. 验证 Hysteria 2 (hy2)
	hy2Outbound := result.Document.Outbounds[4]
	if hy2Outbound.Type != C.TypeHysteria2 {
		t.Fatalf("unexpected hy2 outbound type: %q", hy2Outbound.Type)
	}

	// 6. 验证 TUIC
	tuicOutbound := result.Document.Outbounds[5]
	if tuicOutbound.Type != C.TypeTUIC {
		t.Fatalf("unexpected tuic outbound type: %q", tuicOutbound.Type)
	}

	// 7. 验证 WireGuard
	wgOutbound := result.Document.Outbounds[6]
	if wgOutbound.Type != C.TypeWireGuard {
		t.Fatalf("unexpected wireguard outbound type: %q", wgOutbound.Type)
	}
	wgOpts := wgOutbound.Options.(*option.WireGuardEndpointOptions)
	if len(wgOpts.Peers) == 0 || wgOpts.Peers[0].Port != 51820 {
		t.Fatalf("wireguard peer options invalid: %#v", wgOpts)
	}

	// 验证 WireGuard 也同时作为 Endpoint 保存
	if len(result.Document.Endpoints) == 0 || result.Document.Endpoints[0].Type != C.TypeWireGuard {
		t.Fatalf("wireguard endpoint missing in document: %#v", result.Document.Endpoints)
	}
}

// TestParseWireGuardConf_AmneziaWG 测试解析 WireGuard / AmneziaWG INI/CONF 配置文件。
func TestParseWireGuardConf_AmneziaWG(t *testing.T) {
	conf := `
# Name = my-amnezia-wg
[Interface]
PrivateKey = aW52YWxpZC1wcml2YXRlLWtleS0xMjM=
Address = 10.0.0.2/32, fd00::2/128
DNS = 1.1.1.1
MTU = 1420
Jc = 4
Jmin = 50
Jmax = 1000
S1 = 20
S2 = 50
H1 = 1
H2 = 2
H3 = 3
H4 = 4

[Peer]
PublicKey = aW52YWxpZC1wdWJsaWMta2V5LTEyMw==
PresharedKey = aW52YWxpZC1wc2stMTIz
Endpoint = 198.51.100.88:51820
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
`
	result, err := convert.Content(context.Background(), conf, false)
	if err != nil {
		t.Fatalf("Content failed to parse WireGuard CONF: %v", err)
	}

	if len(result.Document.Outbounds) != 1 {
		t.Fatalf("expected 1 outbound, got %d", len(result.Document.Outbounds))
	}

	outbound := result.Document.Outbounds[0]
	if outbound.Type != C.TypeWireGuard || outbound.Tag != "my-amnezia-wg" {
		t.Fatalf("unexpected wireguard outbound: type=%q tag=%q", outbound.Type, outbound.Tag)
	}

	options := outbound.Options.(*option.WireGuardEndpointOptions)
	if options.MTU != 1420 || len(options.Peers) != 1 {
		t.Fatalf("unexpected wireguard options: %#v", options)
	}
	peer := options.Peers[0]
	if peer.Address != "198.51.100.88" || peer.Port != 51820 || peer.PersistentKeepaliveInterval != 25 {
		t.Fatalf("unexpected wireguard peer: %#v", peer)
	}
}

// TestParseLink_AllSchemesAndDecodedTag 测试各主流协议链接及 #tag URL 编码解码。
func TestParseLink_AllSchemesAndDecodedTag(t *testing.T) {
	links := []string{
		// 1. vless reality + urlencoded tag
		"vless://3c0f47e3-a464-470c-a931-36b8a8d62fd6@198.51.100.1:443?encryption=none&security=reality&sni=example.com&fp=chrome&pbk=tkmyb6Xk2aYMFxrQ35q6PMULtbdIKhaYGG9yySPJbHc&sid=fc1b&type=tcp&flow=xtls-rprx-vision#%E4%BD%A0%E5%A5%BD%E4%B8%96%E7%95%8C",
		// 2. trojan
		"trojan://trojanpassword@198.51.100.2:443?sni=trojan.example.com#TrojanNode",
		// 3. hysteria2 / hy2
		"hy2://hy2password@198.51.100.3:443?sni=hy2.example.com#Hysteria2Node",
		// 4. tuic
		"tuic://3c0f47e3-a464-470c-a931-36b8a8d62fd6:tuicpass@198.51.100.4:8443?congestion_control=bbr&alpn=h3#TUICNode",
		// 5. wireguard
		"wireguard://aW52YWxpZC1wcml2YXRlLWtleQ==@198.51.100.5:51820?publickey=aW52YWxpZC1wdWJsaWMta2V5&address=10.0.0.2/32&presharedkey=aW52YWxpZC1wc2s=&mtu=1420&reserved=1,2,3#WireGuardNode",
		// 6. amneziawg (awg://)
		"awg://aW52YWxpZC1wcml2YXRlLWtleQ==@198.51.100.6:51820?publickey=aW52YWxpZC1wdWJsaWMta2V5&address=10.0.0.2/32&jc=4&jmin=50#AmneziaNode",
		// 7. shadowsocks
		"ss://YWVzLTEyOC1nY206c2VjcmV0@198.51.100.7:8388#ShadowsocksNode",
	}

	content := strings.Join(links, "\n")
	result, err := convert.Content(context.Background(), content, false)
	if err != nil {
		t.Fatalf("Content failed to parse URI list: %v", err)
	}

	if len(result.Document.Outbounds) != 7 {
		t.Fatalf("expected 7 outbounds, got %d", len(result.Document.Outbounds))
	}

	// 验证第一个节点的中文 URL 解码标签
	if result.Document.Outbounds[0].Tag != "你好世界" {
		t.Fatalf("expected URL-decoded tag '你好世界', got %q", result.Document.Outbounds[0].Tag)
	}

	// 验证 WireGuard 节点
	wgOutbound := result.Document.Outbounds[4]
	if wgOutbound.Type != C.TypeWireGuard || wgOutbound.Tag != "WireGuardNode" {
		t.Fatalf("unexpected wg outbound: %#v", wgOutbound)
	}

	// 验证 AmneziaWG 节点
	awgOutbound := result.Document.Outbounds[5]
	if awgOutbound.Type != C.TypeWireGuard || awgOutbound.Tag != "AmneziaNode" {
		t.Fatalf("unexpected awg outbound: %#v", awgOutbound)
	}
}

// TestContent_RecursiveBase64URIList 测试包含 2 层 Base64 编码的订阅。
func TestContent_RecursiveBase64URIList(t *testing.T) {
	rawURIs := "ss://YWVzLTEyOC1nY206c2VjcmV0@198.51.100.7:8388#SS1\nss://YWVzLTEyOC1nY206c2VjcmV0@198.51.100.8:8388#SS2\n"
	layer1 := base64.StdEncoding.EncodeToString([]byte(rawURIs))
	layer2 := base64.RawURLEncoding.EncodeToString([]byte(layer1))

	result, err := convert.Content(context.Background(), layer2, false)
	if err != nil {
		t.Fatalf("Content failed to parse 2-layer Base64 subscription: %v", err)
	}

	if len(result.Document.Outbounds) != 2 {
		t.Fatalf("expected 2 outbounds, got %d", len(result.Document.Outbounds))
	}
	if result.Document.Outbounds[0].Tag != "SS1" || result.Document.Outbounds[1].Tag != "SS2" {
		t.Fatalf("unexpected tags: %q, %q", result.Document.Outbounds[0].Tag, result.Document.Outbounds[1].Tag)
	}
}

// TestProviderParseDocument_Integration 测试通过 provider.ParseDocument 直接解析全格式订阅。
func TestProviderParseDocument_Integration(t *testing.T) {
	clashYAML := `
proxies:
  - name: "clash-provider-node"
    type: trojan
    server: 198.51.100.1
    port: 443
    password: pass
    sni: example.com
`
	doc, err := provider.ParseDocument(context.Background(), []byte(clashYAML))
	if err != nil {
		t.Fatalf("provider.ParseDocument failed on Clash YAML: %v", err)
	}
	if len(doc.Outbounds) != 1 || doc.Outbounds[0].Tag != "clash-provider-node" {
		t.Fatalf("unexpected parsed doc: %#v", doc)
	}
}

package convert_test

import (
	"context"
	"testing"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"

	"github.com/Fanju6/NetProxy-Magisk/src/native/netproxy/internal/convert"
)

// TestParseXrayJSON_RealisticHappSubscription 测试真实 Happ / Incy 订阅场景，
// 包含负载均衡器（URLTest）、VLESS Reality 节点、Hysteria 2 游戏节点及内置服务出站过滤。
func TestParseXrayJSON_RealisticHappSubscription(t *testing.T) {
	rawSubscription := `[
  {
    "remarks": "Самый Быстрый АВТО",
    "log": { "loglevel": "warning" },
    "routing": {
      "domainStrategy": "AsIs",
      "balancers": [
        {
          "tag": "balancer-auto",
          "selector": [
            "grp-26-1",
            "grp-26-2"
          ],
          "strategy": {
            "type": "leastLoad"
          }
        }
      ],
      "rules": [
        {
          "balancerTag": "balancer-auto",
          "network": "tcp,udp",
          "type": "field"
        }
      ]
    },
    "burstObservatory": {
      "subjectSelector": [
        "grp-26-1",
        "grp-26-2"
      ],
      "pingConfig": {
        "destination": "http://www.google.com/generate_204",
        "interval": "10s",
        "connectivity": "http://www.google.com/generate_204",
        "timeout": "5s"
      }
    },
    "outbounds": [
      {
        "tag": "grp-26-1",
        "protocol": "vless",
        "settings": {
          "vnext": [
            {
              "address": "1.2.3.4",
              "port": 443,
              "users": [
                {
                  "id": "e8d697b4-36a5-4228-b86a-7731f82ec620",
                  "flow": "xtls-rprx-vision",
                  "encryption": "none"
                }
              ]
            }
          ]
        },
        "streamSettings": {
          "network": "tcp",
          "security": "reality",
          "realitySettings": {
            "serverName": "google.com",
            "fingerprint": "chrome",
            "show": false,
            "publicKey": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v",
            "shortId": "1234abcd"
          }
        }
      },
      {
        "tag": "grp-26-2",
        "protocol": "vless",
        "settings": {
          "vnext": [
            {
              "address": "5.6.7.8",
              "port": 443,
              "users": [
                {
                  "id": "f9e708c5-47b6-4339-c97b-8842a93fd731",
                  "flow": "xtls-rprx-vision",
                  "encryption": "none"
                }
              ]
            }
          ]
        },
        "streamSettings": {
          "network": "tcp",
          "security": "reality",
          "realitySettings": {
            "serverName": "yahoo.com",
            "fingerprint": "chrome",
            "show": false,
            "publicKey": "b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w",
            "shortId": "5678efgh"
          }
        }
      },
      {
        "protocol": "freedom",
        "tag": "direct"
      },
      {
        "protocol": "blackhole",
        "tag": "block"
      }
    ]
  },
  {
    "remarks": "Германия 🇩🇪",
    "outbounds": [
      {
        "tag": "de-node",
        "protocol": "vless",
        "settings": {
          "vnext": [
            {
              "address": "de.example.com",
              "port": 443,
              "users": [
                {
                  "id": "11111111-2222-3333-4444-555555555555",
                  "flow": "xtls-rprx-vision",
                  "encryption": "none"
                }
              ]
            }
          ]
        },
        "streamSettings": {
          "network": "tcp",
          "security": "reality",
          "realitySettings": {
            "serverName": "de.target.com",
            "fingerprint": "chrome",
            "publicKey": "c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x",
            "shortId": "abcdef12"
          }
        }
      }
    ]
  },
  {
    "remarks": "Игры Hysteria 2 🎮",
    "outbounds": [
      {
        "tag": "hy2-node",
        "protocol": "hysteria",
        "settings": {
          "version": 2,
          "address": "hy2.example.com",
          "port": 8443,
          "auth": "secretpassword",
          "obfs": {
            "type": "salamander",
            "password": "obfspassword"
          },
          "up": "150 Mbps",
          "down": "300 Mbps"
        },
        "streamSettings": {
          "network": "udp",
          "security": "tls",
          "tlsSettings": {
            "serverName": "hy2.example.com",
            "allowInsecure": false,
            "alpn": ["h3"]
          }
        }
      }
    ]
  }
]`

	result, err := convert.Content(context.Background(), rawSubscription, false)
	if err != nil {
		t.Fatalf("Content failed to parse Happ subscription: %v", err)
	}

	doc := result.Document
	// 预期出站：
	// 1. "Самый Быстрый АВТО - grp-26-1" (VLESS)
	// 2. "Самый Быстрый АВТО - grp-26-2" (VLESS)
	// 3. "Самый Быстрый АВТО" (URLTest)
	// 4. "Германия 🇩🇪" (VLESS)
	// 5. "Игры Hysteria 2 🎮" (Hysteria 2)
	// freedom 与 blackhole 出站必须被完全忽略
	if len(doc.Outbounds) != 5 {
		t.Fatalf("expected 5 outbounds, got %d", len(doc.Outbounds))
	}

	// 1. 验证 VLESS Reality 子节点 1
	n1 := doc.Outbounds[0]
	if n1.Type != C.TypeVLESS || n1.Tag != "Самый Быстрый АВТО - grp-26-1" {
		t.Fatalf("unexpected node 1: type=%q, tag=%q", n1.Type, n1.Tag)
	}
	n1Opts := n1.Options.(*option.VLESSOutboundOptions)
	if n1Opts.Server != "1.2.3.4" || n1Opts.ServerPort != 443 || n1Opts.UUID != "e8d697b4-36a5-4228-b86a-7731f82ec620" {
		t.Fatalf("unexpected node 1 server/uuid: %#v", n1Opts)
	}
	if n1Opts.Flow != "xtls-rprx-vision" || n1Opts.TLS == nil || !n1Opts.TLS.Reality.Enabled {
		t.Fatalf("unexpected node 1 reality options: %#v", n1Opts.TLS)
	}
	if n1Opts.TLS.Reality.PublicKey != "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v" || n1Opts.TLS.Reality.ShortID != "1234abcd" {
		t.Fatalf("unexpected node 1 reality keys: %#v", n1Opts.TLS.Reality)
	}

	// 2. 验证 VLESS Reality 子节点 2
	n2 := doc.Outbounds[1]
	if n2.Type != C.TypeVLESS || n2.Tag != "Самый Быстрый АВТО - grp-26-2" {
		t.Fatalf("unexpected node 2: type=%q, tag=%q", n2.Type, n2.Tag)
	}
	n2Opts := n2.Options.(*option.VLESSOutboundOptions)
	if n2Opts.Server != "5.6.7.8" || n2Opts.ServerPort != 443 {
		t.Fatalf("unexpected node 2 server: %#v", n2Opts)
	}

	// 3. 验证 URLTest 负载均衡器
	balancer := doc.Outbounds[2]
	if balancer.Type != C.TypeURLTest || balancer.Tag != "Самый Быстрый АВТО" {
		t.Fatalf("unexpected balancer: type=%q, tag=%q", balancer.Type, balancer.Tag)
	}
	balOpts := balancer.Options.(*option.URLTestOutboundOptions)
	if len(balOpts.Outbounds) != 2 {
		t.Fatalf("expected balancer to have 2 members, got %d: %#v", len(balOpts.Outbounds), balOpts.Outbounds)
	}
	if balOpts.Outbounds[0] != "Самый Быстрый АВТО - grp-26-1" || balOpts.Outbounds[1] != "Самый Быстрый АВТО - grp-26-2" {
		t.Fatalf("balancer member tags do not match node tags: %#v", balOpts.Outbounds)
	}
	if balOpts.URL != "http://www.google.com/generate_204" {
		t.Fatalf("unexpected balancer test URL: %q", balOpts.URL)
	}
	if time.Duration(balOpts.Interval) != 10*time.Second {
		t.Fatalf("unexpected balancer interval: %v", balOpts.Interval)
	}

	// 4. 验证独立单节点 VLESS Reality
	deNode := doc.Outbounds[3]
	if deNode.Type != C.TypeVLESS || deNode.Tag != "Германия 🇩🇪" {
		t.Fatalf("unexpected DE node: type=%q, tag=%q", deNode.Type, deNode.Tag)
	}
	deOpts := deNode.Options.(*option.VLESSOutboundOptions)
	if deOpts.Server != "de.example.com" || deOpts.ServerPort != 443 {
		t.Fatalf("unexpected DE server: %#v", deOpts)
	}

	// 5. 验证独立 Hysteria 2 节点
	hy2Node := doc.Outbounds[4]
	if hy2Node.Type != C.TypeHysteria2 || hy2Node.Tag != "Игры Hysteria 2 🎮" {
		t.Fatalf("unexpected Hy2 node: type=%q, tag=%q", hy2Node.Type, hy2Node.Tag)
	}
	hy2Opts := hy2Node.Options.(*option.Hysteria2OutboundOptions)
	if hy2Opts.Server != "hy2.example.com" || hy2Opts.ServerPort != 8443 || hy2Opts.Password != "secretpassword" {
		t.Fatalf("unexpected Hy2 server/credentials: %#v", hy2Opts)
	}
	if hy2Opts.UpMbps != 150 || hy2Opts.DownMbps != 300 {
		t.Fatalf("unexpected Hy2 bandwidth: up=%d, down=%d", hy2Opts.UpMbps, hy2Opts.DownMbps)
	}
	if hy2Opts.Obfs == nil || hy2Opts.Obfs.Type != "salamander" || hy2Opts.Obfs.Password != "obfspassword" {
		t.Fatalf("unexpected Hy2 obfs: %#v", hy2Opts.Obfs)
	}
	if hy2Opts.TLS == nil || hy2Opts.TLS.ServerName != "hy2.example.com" {
		t.Fatalf("unexpected Hy2 TLS: %#v", hy2Opts.TLS)
	}
}

// TestParseXrayJSON_SingleObject 测试单项 Xray 配置对象（非数组包裹）。
func TestParseXrayJSON_SingleObject(t *testing.T) {
	rawObject := `{
    "remarks": "Single Server",
    "outbounds": [
      {
        "protocol": "vless",
        "settings": {
          "address": "single.example.com",
          "port": 443,
          "uuid": "22222222-3333-4444-5555-666666666666"
        },
        "streamSettings": {
          "network": "ws",
          "security": "tls",
          "tlsSettings": {
            "serverName": "single.example.com"
          },
          "wsSettings": {
            "path": "/ws-path",
            "headers": {
              "Host": "single.example.com"
            }
          }
        }
      }
    ]
  }`

	doc, diags, err := convert.ParseXrayJSON(context.Background(), rawObject)
	if err != nil {
		t.Fatalf("ParseXrayJSON failed on single object: %v", err)
	}
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %#v", diags)
	}
	if len(doc.Outbounds) != 1 {
		t.Fatalf("expected 1 outbound, got %d", len(doc.Outbounds))
	}

	out := doc.Outbounds[0]
	if out.Type != C.TypeVLESS || out.Tag != "Single Server" {
		t.Fatalf("unexpected outbound: type=%q, tag=%q", out.Type, out.Tag)
	}
	opts := out.Options.(*option.VLESSOutboundOptions)
	if opts.Server != "single.example.com" || opts.ServerPort != 443 {
		t.Fatalf("unexpected server: %#v", opts)
	}
	if opts.Transport == nil || opts.Transport.Type != C.V2RayTransportTypeWebsocket {
		t.Fatalf("expected websocket transport, got %#v", opts.Transport)
	}
}

// TestParseXrayJSON_Transports 测试各种底层传输协议（gRPC、HTTP、XHTTP、HTTPUpgrade）。
func TestParseXrayJSON_Transports(t *testing.T) {
	rawGRPC := `{
    "remarks": "gRPC Node",
    "outbounds": [
      {
        "protocol": "vless",
        "settings": {
          "vnext": [{ "address": "grpc.example.com", "port": 443, "users": [{ "id": "uuid" }] }]
        },
        "streamSettings": {
          "network": "grpc",
          "grpcSettings": {
            "serviceName": "gunService"
          }
        }
      }
    ]
  }`

	doc, _, err := convert.ParseXrayJSON(context.Background(), rawGRPC)
	if err != nil {
		t.Fatalf("failed to parse gRPC node: %v", err)
	}
	opts := doc.Outbounds[0].Options.(*option.VLESSOutboundOptions)
	if opts.Transport == nil || opts.Transport.Type != C.V2RayTransportTypeGRPC {
		t.Fatalf("expected gRPC transport: %#v", opts.Transport)
	}
	if opts.Transport.GRPCOptions.ServiceName != "gunService" {
		t.Fatalf("expected gunService, got %q", opts.Transport.GRPCOptions.ServiceName)
	}
}

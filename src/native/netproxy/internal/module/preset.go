package module

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Fanju6/NetProxy-Magisk/src/native/netproxy/internal/paths"
)

// Preset 描述一个预设路由模板的元信息。
type Preset struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Descriptions map[string]string `json:"descriptions,omitempty"`
}

// PresetApplyResult 描述应用预设后的结果。
type PresetApplyResult struct {
	Preset   string `json:"preset"`
	Target   string `json:"target"`
	Revision string `json:"revision"`
	Reloaded bool   `json:"reloaded"`
}

type presetDocument struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Descriptions map[string]string `json:"descriptions"`
	DNS          jsontext.Value    `json:"dns"`
	Route        jsontext.Value    `json:"route"`
}

// 内置预设模板内容，用于缺失磁盘文件或测试环境下的兜底支持。
var builtinPresetFiles = map[string]string{
	"russia": `{
  "$schema": "https://raw.githubusercontent.com/reF1nd/sing-box/reF1nd-testing-next/docs/schema.json",
  "name": "russia",
  "description": "Оптимизировано для РФ: прямой доступ к .ru, банкам, Госуслугам; прокси для YouTube, Google, Discord, Telegram, заблокированных ресурсов",
  "descriptions": {
    "zh": "俄罗斯分流预设（直连 .ru、境内银行、政务与电商；代理 YouTube、Google、Discord、Telegram、AI 及受限站点）",
    "en": "Routing preset for Russia (direct .ru, banks, Gosuslugi, marketplaces; proxy YouTube, Google, Discord, Telegram, AI, blocked sites)",
    "ru": "Оптимизировано для РФ: прямой доступ к .ru, банкам, Госуслугам; прокси для YouTube, Google, Discord, Telegram, заблокированных ресурсов"
  },
  "dns": {
    "servers": [
      {
        "tag": "dns-proxy",
        "type": "group",
        "servers": ["cloudflare", "google"]
      },
      {
        "tag": "dns-direct",
        "type": "group",
        "servers": ["yandex", "direct-local"]
      },
      {
        "type": "hosts",
        "tag": "hosts",
        "predefined": {
          "common.dot.dns.yandex.net": ["77.88.8.8", "77.88.8.1"],
          "cloudflare-dns.com": ["1.1.1.1", "1.0.0.1"],
          "dns.google": ["8.8.8.8", "8.8.4.4"]
        }
      },
      {
        "tag": "cloudflare",
        "type": "https",
        "server": "cloudflare-dns.com",
        "domain_resolver": "hosts",
        "detour": "Proxy"
      },
      {
        "tag": "google",
        "type": "https",
        "server": "dns.google",
        "domain_resolver": "hosts",
        "detour": "Proxy"
      },
      {
        "tag": "yandex",
        "type": "https",
        "server": "common.dot.dns.yandex.net",
        "domain_resolver": "hosts"
      },
      {
        "tag": "direct-local",
        "type": "udp",
        "server": "77.88.8.8"
      }
    ],
    "rules": [
      { "clash_mode": "Global", "action": "route", "server": "dns-proxy" },
      { "clash_mode": "Direct", "action": "route", "server": "dns-direct" },
      {
        "type": "logical",
        "mode": "and",
        "rules": [
          { "clash_mode": "AllowAds", "invert": true },
          { "rule_set": ["block", "Ads_AWAvenue", "geosite/category-ads-all"] }
        ],
        "action": "reject"
      },
      {
        "rule_set": [
          "proxy",
          "geosite/youtube",
          "geosite/google",
          "geosite/discord",
          "geosite/telegram",
          "geosite/category-ai-!cn"
        ],
        "action": "route",
        "server": "dns-proxy"
      },
      {
        "domain_suffix": [
          "instagram.com", "cdninstagram.com", "facebook.com", "fbcdn.net",
          "twitter.com", "x.com", "t.co", "twimg.com", "linkedin.com",
          "rutracker.org", "flibusta.is", "notion.so", "medium.com",
          "torproject.org", "openai.com", "chatgpt.com", "anthropic.com", "claude.ai"
        ],
        "action": "route",
        "server": "dns-proxy"
      },
      {
        "rule_set": ["direct", "geosite/ru"],
        "action": "route",
        "server": "dns-direct"
      },
      {
        "domain_suffix": [
          ".ru", ".su", ".рф", ".xn--p1ai", ".by", ".kz",
          "gosuslugi.ru", "gosuslugi.xn--p1ai", "mos.ru", "nalog.gov.ru", "nalog.ru",
          "sbrf.ru", "sberbank.ru", "sber.ru", "tbank.ru", "tinkoff.ru", "vtb.ru",
          "alfabank.ru", "raiffeisen.ru", "gazprombank.ru", "open.ru", "psbank.ru",
          "rshb.ru", "sovcombank.ru", "nspk.ru", "mir-pay.ru", "sbp.ru", "yoomoney.ru",
          "wildberries.ru", "ozon.ru", "avito.ru", "yandex.ru", "ya.ru",
          "market.yandex.ru", "megamarket.ru", "aliexpress.ru", "lamoda.ru",
          "vk.com", "vkvideo.ru", "ok.ru", "mail.ru", "dzen.ru", "rutube.ru",
          "kinopoisk.ru", "hh.ru", "cian.ru", "2gis.ru", "dns-shop.ru", "mvideo.ru", "eldorado.ru"
        ],
        "action": "route",
        "server": "dns-direct"
      }
    ],
    "final": "dns-proxy",
    "optimistic": true,
    "strategy": "prefer_ipv4"
  },
  "route": {
    "default_domain_resolver": "dns-direct",
    "rule_set": [
      {
        "type": "local",
        "tag": ["block", "direct", "proxy"],
        "format": "source",
        "path": "./rules/local/{tag}.json"
      },
      {
        "type": "remote",
        "tag": "Ads_AWAvenue",
        "format": "source",
        "url": "https://raw.githubusercontent.com/TG-Twilight/AWAvenue-Ads-Rule/main/Filters/AWAvenue-Ads-Rule-Singbox.json",
        "update_interval": "24h",
        "path": "./rules/remote/Ads_AWAvenue.json"
      },
      {
        "type": "remote",
        "tag": [
          "geosite/category-ads-all",
          "geosite/category-ai-!cn",
          "geosite/google",
          "geosite/youtube",
          "geosite/discord",
          "geosite/telegram",
          "geosite/ru",
          "geoip/ru",
          "geoip/telegram"
        ],
        "format": "binary",
        "url": "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/{tag}.srs",
        "update_interval": "24h",
        "path": "./rules/remote/{tag}.srs"
      }
    ],
    "rules": [
      { "action": "sniff" },
      {
        "type": "logical",
        "mode": "or",
        "rules": [{ "port": 53 }, { "protocol": "dns" }],
        "action": "hijack-dns"
      },
      { "clash_mode": "Global", "action": "route", "outbound": "Proxy" },
      { "clash_mode": "Direct", "action": "route", "outbound": "direct" },
      { "rule_set": "proxy", "action": "route", "outbound": "Proxy" },
      { "rule_set": "direct", "action": "route", "outbound": "direct" },
      {
        "type": "logical",
        "mode": "and",
        "rules": [
          { "clash_mode": "AllowAds", "invert": true },
          { "rule_set": ["block", "Ads_AWAvenue", "geosite/category-ads-all"] }
        ],
        "action": "reject"
      },
      {
        "rule_set": [
          "geosite/youtube",
          "geosite/google",
          "geosite/discord",
          "geosite/telegram",
          "geosite/category-ai-!cn"
        ],
        "action": "route",
        "outbound": "Proxy"
      },
      {
        "domain_suffix": [
          "instagram.com", "cdninstagram.com", "facebook.com", "fbcdn.net",
          "twitter.com", "x.com", "t.co", "twimg.com", "linkedin.com",
          "rutracker.org", "flibusta.is", "notion.so", "medium.com",
          "torproject.org", "openai.com", "chatgpt.com", "anthropic.com", "claude.ai",
          "openvpn.net", "wireguard.com"
        ],
        "action": "route",
        "outbound": "Proxy"
      },
      {
        "rule_set": ["geosite/ru"],
        "action": "route",
        "outbound": "direct"
      },
      {
        "domain_suffix": [
          ".ru", ".su", ".рф", ".xn--p1ai", ".by", ".kz",
          "gosuslugi.ru", "gosuslugi.xn--p1ai", "mos.ru", "nalog.gov.ru", "nalog.ru",
          "sbrf.ru", "sberbank.ru", "sber.ru", "tbank.ru", "tinkoff.ru", "vtb.ru",
          "alfabank.ru", "raiffeisen.ru", "gazprombank.ru", "open.ru", "psbank.ru",
          "rshb.ru", "sovcombank.ru", "nspk.ru", "mir-pay.ru", "sbp.ru", "yoomoney.ru",
          "wildberries.ru", "ozon.ru", "avito.ru", "yandex.ru", "ya.ru",
          "market.yandex.ru", "megamarket.ru", "aliexpress.ru", "lamoda.ru",
          "vk.com", "vkvideo.ru", "ok.ru", "mail.ru", "dzen.ru", "rutube.ru",
          "kinopoisk.ru", "hh.ru", "cian.ru", "2gis.ru", "dns-shop.ru", "mvideo.ru", "eldorado.ru"
        ],
        "action": "route",
        "outbound": "direct"
      },
      { "action": "resolve", "match_only": true },
      { "rule_set": ["geoip/ru"], "action": "route", "outbound": "direct" },
      { "rule_set": ["geoip/telegram"], "action": "route", "outbound": "Proxy" },
      { "ip_is_private": true, "outbound": "direct" }
    ],
    "find_process": true,
    "auto_detect_interface": true,
    "final": "Proxy"
  }
}`,
	"bypass-lan": `{
  "$schema": "https://raw.githubusercontent.com/reF1nd/sing-box/reF1nd-testing-next/docs/schema.json",
  "name": "bypass-lan",
  "description": "Проксировать весь интернет, напрямую только локальные сети (RFC1918 / LAN)",
  "descriptions": {
    "zh": "绕过局域网预设（全局代理整个互联网，仅保留局域网与私有 IP 直连）",
    "en": "Bypass LAN preset (proxy all internet traffic, direct only local / RFC1918 private subnets)",
    "ru": "Проксировать весь интернет, напрямую только локальные сети (RFC1918 / LAN)"
  },
  "dns": {
    "servers": [
      {
        "tag": "dns-proxy",
        "type": "group",
        "servers": ["cloudflare", "google"]
      },
      {
        "tag": "dns-direct",
        "type": "udp",
        "server": "223.5.5.5"
      },
      {
        "type": "hosts",
        "tag": "hosts",
        "predefined": {
          "cloudflare-dns.com": ["1.1.1.1", "1.0.0.1"],
          "dns.google": ["8.8.8.8", "8.8.4.4"]
        }
      },
      {
        "tag": "cloudflare",
        "type": "https",
        "server": "cloudflare-dns.com",
        "domain_resolver": "hosts",
        "detour": "Proxy"
      },
      {
        "tag": "google",
        "type": "https",
        "server": "dns.google",
        "domain_resolver": "hosts",
        "detour": "Proxy"
      }
    ],
    "rules": [
      { "clash_mode": "Direct", "action": "route", "server": "dns-direct" },
      {
        "type": "logical",
        "mode": "and",
        "rules": [
          { "clash_mode": "AllowAds", "invert": true },
          { "rule_set": ["block", "Ads_AWAvenue", "geosite/category-ads-all"] }
        ],
        "action": "reject"
      },
      {
        "domain_suffix": [".local", ".lan", ".internal", ".home.arpa"],
        "action": "route",
        "server": "dns-direct"
      }
    ],
    "final": "dns-proxy",
    "optimistic": true,
    "strategy": "prefer_ipv4"
  },
  "route": {
    "default_domain_resolver": "dns-proxy",
    "rule_set": [
      {
        "type": "local",
        "tag": ["block", "direct", "proxy"],
        "format": "source",
        "path": "./rules/local/{tag}.json"
      },
      {
        "type": "remote",
        "tag": "Ads_AWAvenue",
        "format": "source",
        "url": "https://raw.githubusercontent.com/TG-Twilight/AWAvenue-Ads-Rule/main/Filters/AWAvenue-Ads-Rule-Singbox.json",
        "update_interval": "24h",
        "path": "./rules/remote/Ads_AWAvenue.json"
      },
      {
        "type": "remote",
        "tag": ["geosite/category-ads-all"],
        "format": "binary",
        "url": "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/{tag}.srs",
        "update_interval": "24h",
        "path": "./rules/remote/{tag}.srs"
      }
    ],
    "rules": [
      { "action": "sniff" },
      {
        "type": "logical",
        "mode": "or",
        "rules": [{ "port": 53 }, { "protocol": "dns" }],
        "action": "hijack-dns"
      },
      { "clash_mode": "Direct", "action": "route", "outbound": "direct" },
      {
        "type": "logical",
        "mode": "and",
        "rules": [
          { "clash_mode": "AllowAds", "invert": true },
          { "rule_set": ["block", "Ads_AWAvenue", "geosite/category-ads-all"] }
        ],
        "action": "reject"
      },
      { "rule_set": "direct", "action": "route", "outbound": "direct" },
      { "ip_is_private": true, "outbound": "direct" },
      {
        "domain": ["localhost"],
        "domain_suffix": [".local", ".lan", ".internal", ".home.arpa"],
        "action": "route",
        "outbound": "direct"
      },
      {
        "ip_cidr": [
          "127.0.0.0/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
          "169.254.0.0/16", "::1/128", "fc00::/7", "fe80::/10"
        ],
        "action": "route",
        "outbound": "direct"
      }
    ],
    "find_process": true,
    "auto_detect_interface": true,
    "final": "Proxy"
  }
}`,
	"china": `{
  "$schema": "https://raw.githubusercontent.com/reF1nd/sing-box/reF1nd-testing-next/docs/schema.json",
  "name": "china",
  "description": "经典中国大陆分流预设（直连国内域名与 IP，国外网站走代理）",
  "descriptions": {
    "zh": "经典中国大陆分流预设（直连国内域名与 IP，国外网站走代理）",
    "en": "Classic routing profile for China (direct cn domains/IPs, proxy international)",
    "ru": "Классический профиль для Китая (прямой доступ к geosite:cn и geoip:cn, остальное через прокси)"
  },
  "dns": {
    "servers": [
      {
        "tag": "dns-proxy",
        "type": "group",
        "servers": ["cloudflare", "google"]
      },
      {
        "tag": "dns-direct",
        "type": "group",
        "servers": ["ali", "tencent"]
      },
      {
        "type": "hosts",
        "tag": "hosts",
        "predefined": {
          "doh.pub": ["120.53.53.53", "1.12.12.12"],
          "cloudflare-dns.com": ["1.1.1.1", "1.0.0.1"],
          "dns.google": ["8.8.8.8", "8.8.4.4"],
          "dns.alidns.com": ["223.5.5.5", "223.6.6.6"]
        }
      },
      {
        "tag": "cloudflare",
        "type": "https",
        "server": "cloudflare-dns.com",
        "domain_resolver": "hosts",
        "detour": "Proxy"
      },
      {
        "tag": "google",
        "type": "https",
        "server": "dns.google",
        "domain_resolver": "hosts",
        "detour": "Proxy"
      },
      {
        "tag": "ali",
        "type": "https",
        "server": "dns.alidns.com",
        "domain_resolver": "hosts"
      },
      {
        "tag": "tencent",
        "type": "https",
        "server": "doh.pub",
        "domain_resolver": "hosts"
      }
    ],
    "rules": [
      { "clash_mode": "Global", "action": "route", "server": "dns-proxy" },
      { "clash_mode": "Direct", "action": "route", "server": "dns-direct" },
      {
        "rule_set": ["direct", "geosite/apple-cn"],
        "action": "route",
        "server": "dns-direct"
      },
      { "rule_set": ["proxy"], "action": "route", "server": "dns-proxy" },
      {
        "type": "logical",
        "mode": "and",
        "rules": [
          { "clash_mode": "AllowAds", "invert": true },
          { "rule_set": ["block", "Ads_AWAvenue", "geosite/category-ads-all"] }
        ],
        "action": "reject"
      },
      {
        "rule_set": ["geosite/category-ai-!cn", "geosite/google"],
        "action": "route",
        "server": "dns-proxy"
      },
      { "rule_set": ["geosite/cn"], "action": "route", "server": "dns-direct" },
      {
        "rule_set": ["geosite/geolocation-!cn"],
        "action": "route",
        "server": "dns-proxy"
      }
    ],
    "final": "dns-proxy",
    "optimistic": true,
    "strategy": "prefer_ipv4"
  },
  "route": {
    "default_domain_resolver": "dns-direct",
    "rule_set": [
      {
        "type": "local",
        "tag": ["block", "direct", "proxy"],
        "format": "source",
        "path": "./rules/local/{tag}.json"
      },
      {
        "type": "remote",
        "tag": "Ads_AWAvenue",
        "format": "source",
        "url": "https://raw.githubusercontent.com/TG-Twilight/AWAvenue-Ads-Rule/main/Filters/AWAvenue-Ads-Rule-Singbox.json",
        "update_interval": "24h",
        "path": "./rules/remote/Ads_AWAvenue.json"
      },
      {
        "type": "remote",
        "tag": [
          "geosite/category-ads-all",
          "geosite/apple-cn",
          "geosite/category-ai-!cn",
          "geosite/google",
          "geosite/geolocation-!cn",
          "geosite/cn",
          "geoip/cn",
          "geoip/telegram"
        ],
        "format": "binary",
        "url": "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/{tag}.srs",
        "update_interval": "24h",
        "path": "./rules/remote/{tag}.srs"
      }
    ],
    "rules": [
      { "action": "sniff" },
      {
        "type": "logical",
        "mode": "or",
        "rules": [{ "port": 53 }, { "protocol": "dns" }],
        "action": "hijack-dns"
      },
      { "clash_mode": "Global", "action": "route", "outbound": "Proxy" },
      { "clash_mode": "Direct", "action": "route", "outbound": "direct" },
      { "rule_set": "proxy", "action": "route", "outbound": "Proxy" },
      { "rule_set": "direct", "action": "route", "outbound": "direct" },
      {
        "type": "logical",
        "mode": "and",
        "rules": [
          { "clash_mode": "AllowAds", "invert": true },
          { "rule_set": ["block", "Ads_AWAvenue", "geosite/category-ads-all"] }
        ],
        "action": "reject"
      },
      {
        "rule_set": ["geosite/category-ai-!cn", "geosite/google"],
        "action": "route",
        "outbound": "Proxy"
      },
      {
        "rule_set": ["geosite/apple-cn", "geosite/cn"],
        "action": "route",
        "outbound": "direct"
      },
      {
        "rule_set": ["geosite/geolocation-!cn"],
        "action": "route",
        "outbound": "Proxy"
      },
      { "action": "resolve", "match_only": true },
      { "rule_set": "geoip/cn", "action": "route", "outbound": "direct" },
      { "rule_set": ["geoip/telegram"], "action": "route", "outbound": "Proxy" },
      { "ip_is_private": true, "outbound": "direct" }
    ],
    "find_process": true,
    "auto_detect_interface": true,
    "final": "Proxy"
  }
}`,
}

// selectPresetDescription 根据目标语言从多语言映射或单一描述中选择合适文本。
func selectPresetDescription(description string, descriptions map[string]string, lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if descriptions != nil {
		if val, exists := descriptions[lang]; exists && strings.TrimSpace(val) != "" {
			return val
		}
		if strings.HasPrefix(lang, "zh") {
			if val, exists := descriptions["zh"]; exists && strings.TrimSpace(val) != "" {
				return val
			}
		}
		if strings.HasPrefix(lang, "ru") {
			if val, exists := descriptions["ru"]; exists && strings.TrimSpace(val) != "" {
				return val
			}
		}
		if val, exists := descriptions["en"]; exists && strings.TrimSpace(val) != "" {
			return val
		}
	}
	return description
}

// ListPresets 返回所有可用的预设路由模板列表。
func ListPresets(options Options, lang string) ([]Preset, error) {
	if err := options.validate(); err != nil {
		return nil, err
	}
	presetMap := make(map[string]Preset)

	// 首先载入内置预设作为基准模板。
	for name, content := range builtinPresetFiles {
		var doc presetDocument
		if err := json.Unmarshal([]byte(content), &doc); err == nil {
			desc := selectPresetDescription(doc.Description, doc.Descriptions, lang)
			presetMap[name] = Preset{
				ID:           name,
				Name:         name,
				Description:  desc,
				Descriptions: doc.Descriptions,
			}
		}
	}

	// 随后读取磁盘目录中的预设文件，允许覆盖或扩充自定义模板。
	presetsDir := paths.SingBoxPresetsDir(options.SingBoxDir)
	entries, err := os.ReadDir(presetsDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".json")
			filePath := filepath.Join(presetsDir, entry.Name())
			content, readErr := os.ReadFile(filePath)
			if readErr != nil {
				continue
			}
			var doc presetDocument
			if jsonErr := json.Unmarshal(content, &doc); jsonErr == nil {
				descName := doc.Name
				if descName == "" {
					descName = name
				}
				desc := selectPresetDescription(doc.Description, doc.Descriptions, lang)
				presetMap[name] = Preset{
					ID:           name,
					Name:         descName,
					Description:  desc,
					Descriptions: doc.Descriptions,
				}
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	results := make([]Preset, 0, len(presetMap))
	for _, p := range presetMap {
		results = append(results, p)
	}

	// 保持常用预设的稳定排序：russia, bypass-lan, china 优先，其余按字母排序。
	sort.Slice(results, func(i, j int) bool {
		priority := map[string]int{"russia": 1, "bypass-lan": 2, "china": 3}
		pi := priority[results[i].ID]
		pj := priority[results[j].ID]
		if pi != 0 && pj != 0 {
			return pi < pj
		}
		if pi != 0 {
			return true
		}
		if pj != 0 {
			return false
		}
		return results[i].ID < results[j].ID
	})

	return results, nil
}

// readPresetContent 读取指定名称的预设文件内容（优先从磁盘读取，缺失时从内置模板兜底）。
func readPresetContent(options Options, name string) ([]byte, error) {
	cleanName := strings.TrimSuffix(strings.TrimSpace(name), ".json")
	if cleanName == "" {
		return nil, errors.New("预设名称不能为空")
	}

	// 1. 尝试从模块预设目录读取。
	presetsDir := paths.SingBoxPresetsDir(options.SingBoxDir)
	diskPath := filepath.Join(presetsDir, cleanName+".json")
	if content, err := os.ReadFile(diskPath); err == nil {
		return content, nil
	}

	// 2. 尝试从内置预设字典读取。
	if contentStr, exists := builtinPresetFiles[cleanName]; exists {
		return []byte(contentStr), nil
	}

	return nil, fmt.Errorf("预设 %q 不存在", cleanName)
}

// ApplyPreset 将指定的预设路由规则原子应用到当前主配置文件中，保留原有的出站节点与接口定义。
func ApplyPreset(ctx context.Context, options Options, name string, restartService bool) (PresetApplyResult, error) {
	cleanName := strings.TrimSuffix(strings.TrimSpace(name), ".json")
	if cleanName == "" {
		return PresetApplyResult{}, errors.New("预设名称不能为空")
	}

	if err := options.validate(); err != nil {
		return PresetApplyResult{}, err
	}

	presetBytes, err := readPresetContent(options, cleanName)
	if err != nil {
		return PresetApplyResult{}, err
	}

	presetObject, err := configObject(presetBytes)
	if err != nil {
		return PresetApplyResult{}, fmt.Errorf("解析预设内容失败: %w", err)
	}

	presetRoute, hasRoute := presetObject["route"]
	if !hasRoute || len(presetRoute) == 0 {
		return PresetApplyResult{}, errors.New("预设文件必须包含有效的 route 配置")
	}
	presetDNS, hasDNS := presetObject["dns"]

	// 读取现有主配置以保留用户自定义的接口、出站与其他分区。
	configPath := paths.SingBoxConfig(options.SingBoxDir)
	var currentObject map[string]jsontext.Value
	currentBytes, err := os.ReadFile(configPath)
	if err == nil {
		currentObject, _ = configObject(currentBytes)
	}
	if currentObject == nil {
		currentObject = make(map[string]jsontext.Value)
	}

	// 必须保留用户现有的接口（inbounds）、自定义出站（outbounds）与核心控制台服务（services）。
	// 若当前主配置中缺失相应字段，则从预设或默认模板中填充。
	if _, exists := currentObject["inbounds"]; !exists {
		if val, ok := presetObject["inbounds"]; ok {
			currentObject["inbounds"] = val
		}
	}
	if _, exists := currentObject["outbounds"]; !exists {
		if val, ok := presetObject["outbounds"]; ok {
			currentObject["outbounds"] = val
		}
	}
	if _, exists := currentObject["services"]; !exists {
		if val, ok := presetObject["services"]; ok {
			currentObject["services"] = val
		}
	}
	if _, exists := currentObject["experimental"]; !exists {
		if val, ok := presetObject["experimental"]; ok {
			currentObject["experimental"] = val
		}
	}
	if _, exists := currentObject["log"]; !exists {
		if val, ok := presetObject["log"]; ok {
			currentObject["log"] = val
		}
	}
	if _, exists := currentObject["http_clients"]; !exists {
		if val, ok := presetObject["http_clients"]; ok {
			currentObject["http_clients"] = val
		}
	}
	if _, exists := currentObject["$schema"]; !exists {
		if val, ok := presetObject["$schema"]; ok {
			currentObject["$schema"] = val
		}
	}

	// 替换 route 与 dns 为预设所定义的内容。
	currentObject["route"] = presetRoute
	if hasDNS && len(presetDNS) > 0 {
		currentObject["dns"] = presetDNS
	}

	// 移除预设可能包含的元数据字段，确保生成的主配置符合 sing-box schema 校验。
	delete(currentObject, "name")
	delete(currentObject, "description")
	delete(currentObject, "descriptions")
	delete(currentObject, "id")
	delete(currentObject, "_preset")

	mergedContent, err := json.Marshal(currentObject, json.Deterministic(true), jsontext.WithIndent("  "))
	if err != nil {
		return PresetApplyResult{}, fmt.Errorf("合并配置序列化失败: %w", err)
	}

	// 确保 sing-box 配置根目录存在。
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return PresetApplyResult{}, err
	}

	// 写入临时候选文件并通过 ApplyConfig 原子应用与事务热加载。
	candidate, err := os.CreateTemp(filepath.Dir(configPath), ".preset-candidate-")
	if err != nil {
		return PresetApplyResult{}, err
	}
	candidatePath := candidate.Name()
	defer os.Remove(candidatePath)

	if _, err := candidate.Write(mergedContent); err != nil {
		_ = candidate.Close()
		return PresetApplyResult{}, err
	}
	if err := candidate.Close(); err != nil {
		return PresetApplyResult{}, err
	}

	revision, err := ApplyConfig(ctx, options, "singbox/config.json", candidatePath, false, "")
	if err != nil {
		return PresetApplyResult{}, err
	}

	running := configProcessRunning(options.SingBoxPath)
	if restartService && running {
		// 若显式指定重启服务且当前正在运行，则执行服务重启流程。
		if _, restartErr := ManageService(ctx, options, "restart"); restartErr != nil {
			return PresetApplyResult{}, fmt.Errorf("配置已应用，但重启服务失败: %w", restartErr)
		}
	}

	return PresetApplyResult{
		Preset:   cleanName,
		Target:   "singbox/config.json",
		Revision: revision,
		Reloaded: running,
	}, nil
}

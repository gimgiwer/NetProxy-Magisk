package module

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/Fanju6/NetProxy-Magisk/src/native/netproxy/internal/paths"
)

// TestListPresets 验证预设列表读取与多语言描述支持。
func TestListPresets(t *testing.T) {
	root := t.TempDir()
	options := NewOptions(root)

	// 1. 验证在空目录下的内置预设兜底。
	presets, err := ListPresets(options, "zh")
	if err != nil {
		t.Fatalf("ListPresets(zh) 失败: %v", err)
	}
	if len(presets) < 3 {
		t.Fatalf("内置预设数量不足: 期望至少 3 个，实际 %d", len(presets))
	}

	foundRussia := false
	foundBypass := false
	foundChina := false
	for _, p := range presets {
		switch p.ID {
		case "russia":
			foundRussia = true
			if p.Description == "" {
				t.Errorf("russia 预设缺少中文描述")
			}
		case "bypass-lan":
			foundBypass = true
		case "china":
			foundChina = true
		}
	}
	if !foundRussia || !foundBypass || !foundChina {
		t.Errorf("内置预设列表缺失核心预设: russia=%v, bypass=%v, china=%v", foundRussia, foundBypass, foundChina)
	}

	// 2. 验证俄语和英语语言切换。
	presetsRu, err := ListPresets(options, "ru")
	if err != nil {
		t.Fatalf("ListPresets(ru) 失败: %v", err)
	}
	for _, p := range presetsRu {
		if p.ID == "russia" && p.Description == "" {
			t.Errorf("russia 预设在 ru 语言下缺少描述")
		}
	}

	// 3. 验证从磁盘新增自定义预设。
	presetsDir := paths.SingBoxPresetsDir(options.SingBoxDir)
	if err := os.MkdirAll(presetsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	customPreset := `{
		"name": "custom-vpn",
		"description": "自定义测试预设",
		"descriptions": {"ru": "Пользовательский пресет", "en": "Custom preset"},
		"route": {"rules": []}
	}`
	if err := os.WriteFile(filepath.Join(presetsDir, "custom-vpn.json"), []byte(customPreset), 0o600); err != nil {
		t.Fatal(err)
	}

	presetsWithCustom, err := ListPresets(options, "ru")
	if err != nil {
		t.Fatalf("读取包含自定义预设的列表失败: %v", err)
	}
	foundCustom := false
	for _, p := range presetsWithCustom {
		if p.ID == "custom-vpn" {
			foundCustom = true
			if p.Description != "Пользовательский пресет" {
				t.Errorf("自定义预设俄语描述匹配错误: %q", p.Description)
			}
		}
	}
	if !foundCustom {
		t.Errorf("未在列表中发现磁盘自定义预设 custom-vpn")
	}
}

// TestApplyPresetPreservesInboundsAndOutbounds 验证应用预设时正确保留用户自定 inbounds 与 outbounds。
func TestApplyPresetPreservesInboundsAndOutbounds(t *testing.T) {
	root := t.TempDir()
	options := newTestOptions(root)
	options.StateFile = filepath.Join(root, "state", "service.json")
	options.LogDir = filepath.Join(root, "logs")
	options.SingBoxPath = fakeSingBox(t)

	// 初始化基础配置目录与必要支撑文件。
	if err := os.MkdirAll(options.SingBoxDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(options.CatalogRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(options.ModuleConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(options.ModuleConfig, []byte("\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(options.EBPFConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(options.EBPFConfig, []byte("\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(options.StateFile), 0o700); err != nil {
		t.Fatal(err)
	}
	initialConfig := `{
		"inbounds": [
			{
				"type": "mixed",
				"tag": "user-mixed-in",
				"listen": "127.0.0.1",
				"listen_port": 10808
			},
			{
				"type": "tproxy",
				"tag": "user-tproxy-in",
				"listen_port": 10809
			}
		],
		"outbounds": [
			{
				"type": "direct",
				"tag": "my-direct"
			},
			{
				"type": "block",
				"tag": "my-block"
			}
		],
		"services": [
			{
				"type": "api",
				"listen": "127.0.0.1",
				"listen_port": 9999
			}
		],
		"route": {
			"rules": []
		}
	}`
	configPath := paths.SingBoxConfig(options.SingBoxDir)
	if err := os.WriteFile(configPath, []byte(initialConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	// 1. 应用 russia 预设。
	result, err := ApplyPreset(context.Background(), options, "russia", false)
	if err != nil {
		t.Fatalf("ApplyPreset(russia) 失败: %v", err)
	}
	if result.Preset != "russia" {
		t.Errorf("结果预设名称错误: 期望 russia，实际 %s", result.Preset)
	}
	if result.Revision == "" {
		t.Errorf("结果缺少 revision 标识")
	}

	// 读取应用后的配置文件内容并校验保留项。
	appliedBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("读取应用后配置文件失败: %v", err)
	}
	var appliedObject map[string]jsontext.Value
	if err := json.Unmarshal(appliedBytes, &appliedObject); err != nil {
		t.Fatalf("应用后配置不是有效 JSON: %v", err)
	}

	// 校验 inbounds 必须被完整保留。
	var inbounds []map[string]any
	if err := json.Unmarshal(appliedObject["inbounds"], &inbounds); err != nil {
		t.Fatalf("解析应用后 inbounds 失败: %v", err)
	}
	if len(inbounds) != 2 || inbounds[0]["tag"] != "user-mixed-in" || inbounds[1]["tag"] != "user-tproxy-in" {
		t.Errorf("用户自定义 inbounds 未被正确保留: %+v", inbounds)
	}

	// 校验 outbounds 必须被完整保留。
	var outbounds []map[string]any
	if err := json.Unmarshal(appliedObject["outbounds"], &outbounds); err != nil {
		t.Fatalf("解析应用后 outbounds 失败: %v", err)
	}
	if len(outbounds) != 2 || outbounds[0]["tag"] != "my-direct" {
		t.Errorf("用户自定义 outbounds 未被正确保留: %+v", outbounds)
	}

	// 校验 services 必须被保留。
	if _, exists := appliedObject["services"]; !exists {
		t.Errorf("用户自定义 services 丢失")
	}

	// 校验元数据字段未被写入生成的 config.json。
	for _, forbiddenKey := range []string{"name", "description", "descriptions", "id", "_preset"} {
		if _, exists := appliedObject[forbiddenKey]; exists {
			t.Errorf("生成的 config.json 中包含非法的预设元数据字段 %q", forbiddenKey)
		}
	}

	// 校验 route 分区已更新为 russia 规则。
	var route map[string]any
	if err := json.Unmarshal(appliedObject["route"], &route); err != nil {
		t.Fatalf("解析应用后 route 失败: %v", err)
	}
	rules, ok := route["rules"].([]any)
	if !ok || len(rules) == 0 {
		t.Errorf("应用后的 route.rules 为空")
	}

	// 2. 应用 bypass-lan 预设。
	resultBypass, err := ApplyPreset(context.Background(), options, "bypass-lan", false)
	if err != nil {
		t.Fatalf("ApplyPreset(bypass-lan) 失败: %v", err)
	}
	if resultBypass.Preset != "bypass-lan" {
		t.Errorf("结果预设名称错误: 期望 bypass-lan，实际 %s", resultBypass.Preset)
	}

	// 3. 验证应用不存在的预设时返回合理错误。
	_, errNonExistent := ApplyPreset(context.Background(), options, "non-existent-preset", false)
	if errNonExistent == nil {
		t.Errorf("应用不存在的预设应该报错，但未报错")
	}
}

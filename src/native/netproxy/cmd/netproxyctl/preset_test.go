package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	moduleapp "github.com/Fanju6/NetProxy-Magisk/src/native/netproxy/internal/module"
)

// withStdoutCapture 捕获标准输出用于验证 CLI JSON 契约。
func withStdoutCapture(run func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	run()

	_ = w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// TestPresetListCLI 验证 preset list 命令输出符合契约与多语言支持。
func TestPresetListCLI(t *testing.T) {
	root := t.TempDir()
	options := moduleapp.NewOptions(root)
	command := &cli{options: options}

	// 1. 默认语言 (en/ru/zh 根据环境)
	output := withStdoutCapture(func() {
		err := command.preset(context.Background(), []string{"list"})
		if err != nil {
			t.Fatalf("preset list 执行失败: %v", err)
		}
	})

	if !strings.Contains(output, `"code":"preset.list"`) {
		t.Errorf("输出缺失预期 code preset.list: %s", output)
	}
	if !strings.Contains(output, `"russia"`) || !strings.Contains(output, `"china"`) || !strings.Contains(output, `"bypass-lan"`) {
		t.Errorf("预设列表中缺失核心预设: %s", output)
	}

	// 2. 显式俄语输出 --lang ru
	outputRu := withStdoutCapture(func() {
		err := command.preset(context.Background(), []string{"list", "--lang", "ru"})
		if err != nil {
			t.Fatalf("preset list --lang ru 失败: %v", err)
		}
	})
	if !strings.Contains(outputRu, "Список доступных пресетов маршрутизации") {
		t.Errorf("俄语提示信息不匹配: %s", outputRu)
	}

	// 3. 显式中文输出 --lang zh
	outputZh := withStdoutCapture(func() {
		err := command.preset(context.Background(), []string{"list", "--lang", "zh"})
		if err != nil {
			t.Fatalf("preset list --lang zh 失败: %v", err)
		}
	})
	if !strings.Contains(outputZh, "预设路由规则列表") {
		t.Errorf("中文提示信息不匹配: %s", outputZh)
	}
}

// TestPresetApplyCLI 验证 preset apply 命令的执行与错误处理。
func TestPresetApplyCLI(t *testing.T) {
	root := t.TempDir()
	options := moduleapp.NewOptions(root)
	options.StateFile = filepath.Join(root, "state", "service.json")
	options.LogDir = filepath.Join(root, "logs")

	fakeBin := filepath.Join(root, "fake-sing-box")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	options.SingBoxPath = fakeBin

	// 准备 singbox 配置环境与空支持文件
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

	command := &cli{options: options}

	// 1. 缺少预设名称参数时报错。
	errMissing := command.preset(context.Background(), []string{"apply"})
	if errMissing == nil {
		t.Errorf("缺少预设名称时应返回错误")
	}

	// 2. 应用 russia 预设。
	outputApply := withStdoutCapture(func() {
		err := command.preset(context.Background(), []string{"apply", "russia", "--lang", "ru"})
		if err != nil {
			t.Fatalf("preset apply russia 失败: %v", err)
		}
	})
	if !strings.Contains(outputApply, `"code":"preset.applied"`) {
		t.Errorf("应用预设返回 code 错误: %s", outputApply)
	}
	if !strings.Contains(outputApply, "Пресет маршрутизации успешно применён") {
		t.Errorf("俄语提示文案不匹配: %s", outputApply)
	}

	// 3. 未知子操作返回错误。
	errUnknown := command.preset(context.Background(), []string{"unknown-op"})
	if errUnknown == nil {
		t.Errorf("未知操作应该报错")
	}
}

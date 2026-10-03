package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	moduleapp "github.com/Fanju6/NetProxy-Magisk/src/native/netproxy/internal/module"
)

// presetLocale 返回当前系统的首选语言代码 (zh, ru 或 en)。
func presetLocale(explicit string) string {
	if explicit != "" {
		code := strings.ToLower(strings.TrimSpace(explicit))
		switch {
		case strings.HasPrefix(code, "zh"):
			return "zh"
		case strings.HasPrefix(code, "ru"):
			return "ru"
		default:
			return "en"
		}
	}
	for _, env := range []string{"NETPROXY_LANG", "LC_ALL", "LC_MESSAGES", "LANG"} {
		val := strings.ToLower(strings.TrimSpace(os.Getenv(env)))
		if strings.HasPrefix(val, "zh") {
			return "zh"
		}
		if strings.HasPrefix(val, "ru") {
			return "ru"
		}
		if strings.HasPrefix(val, "en") {
			return "en"
		}
	}
	return "en"
}

// presetMessage 根据语言返回操作提示文本。
func presetMessage(code, lang string) string {
	switch code {
	case "preset.list":
		switch lang {
		case "zh":
			return "预设路由规则列表"
		case "ru":
			return "Список доступных пресетов маршрутизации"
		default:
			return "Routing preset list"
		}
	case "preset.applied":
		switch lang {
		case "zh":
			return "预设路由已应用"
		case "ru":
			return "Пресет маршрутизации успешно применён"
		default:
			return "Routing preset applied successfully"
		}
	default:
		return "操作完成"
	}
}

// parsePresetArgs 解析 preset 命令的子动作、选项和位置参数，支持选项任意位置传参。
func parsePresetArgs(args []string) (action string, positionals []string, lang string, restart bool, err error) {
	if len(args) == 0 {
		return "", nil, "", false, errors.New("缺少 preset 操作")
	}
	action = args[0]
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--restart":
			restart = true
		case arg == "--lang":
			if i+1 >= len(args) {
				return "", nil, "", false, errors.New("--lang 需要指定语言代码")
			}
			i++
			lang = args[i]
		case strings.HasPrefix(arg, "--lang="):
			lang = strings.TrimPrefix(arg, "--lang=")
		case strings.HasPrefix(arg, "-"):
			return "", nil, "", false, fmt.Errorf("未知选项: %s", arg)
		default:
			positionals = append(positionals, arg)
		}
	}
	return action, positionals, lang, restart, nil
}

// preset 处理 preset 子命令组，支持 list 与 apply。
func (c *cli) preset(ctx context.Context, args []string) error {
	action, positionals, langArg, restart, err := parsePresetArgs(args)
	if err != nil {
		return usageError("用法: netproxyctl preset list|apply <名称> [--restart] [--lang <zh|ru|en>]")
	}

	currentLang := presetLocale(langArg)

	switch action {
	case "list":
		presets, err := moduleapp.ListPresets(c.options, currentLang)
		if err != nil {
			return err
		}
		writeJSON(os.Stdout, result{
			Schema:  1,
			OK:      true,
			Code:    "preset.list",
			Message: presetMessage("preset.list", currentLang),
			Data:    presets,
		})
		return nil

	case "apply":
		if len(positionals) == 0 {
			switch currentLang {
			case "zh":
				return errors.New("preset apply 需要指定预设名称")
			case "ru":
				return errors.New("Команде preset apply требуется имя пресета")
			default:
				return errors.New("preset apply requires a preset name")
			}
		}

		name := positionals[0]
		applyResult, err := moduleapp.ApplyPreset(ctx, c.options, name, restart)
		if err != nil {
			return err
		}

		writeJSON(os.Stdout, result{
			Schema:  1,
			OK:      true,
			Code:    "preset.applied",
			Message: presetMessage("preset.applied", currentLang),
			Data:    applyResult,
		})
		return nil

	default:
		switch currentLang {
		case "zh":
			return fmt.Errorf("未知 preset 操作 %q", action)
		case "ru":
			return fmt.Errorf("Неизвестная операция preset %q", action)
		default:
			return fmt.Errorf("unknown preset action %q", action)
		}
	}
}

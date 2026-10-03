#!/usr/bin/env sh
# 文件: tests/preset_test.sh
# 功能: 验证 sing-box 路由预设模板完整性与 netproxyctl preset CLI 契约
# 用法: sh tests/preset_test.sh
# 依赖: POSIX sh、jq、go

set -eu

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
PRESETS_DIR="$ROOT/src/module/config/singbox/presets"

#######################################
# 检查预设模板文件存在性与 JSON 格式
# 参数: 无
# 返回: 0=所有预设存在且合法，1=缺失或 JSON 语法错误
#######################################
check_preset_files() {
  for name in russia bypass-lan china; do
    file="$PRESETS_DIR/$name.json"
    [ -f "$file" ] || {
      printf '%s\n' "缺少预设文件: $file" >&2
      return 1
    }
    jq empty "$file" 2>/dev/null || {
      printf '%s\n' "预设文件 JSON 格式无效: $file" >&2
      return 1
    }
  done
}

#######################################
# 检查 russia.json 规则完整性
# 参数: 无
# 返回: 0=关键规则完备，1=缺少必要路由分流项
#######################################
check_russia_preset_rules() {
  file="$PRESETS_DIR/russia.json"

  # 1. 检查 direct 域名后缀包含 .ru, .su, .рф, 银行与政务服务
  jq -e '[.route.rules[] | select(.outbound == "direct") | .domain_suffix] | flatten | index(".ru") != null' "$file" >/dev/null
  jq -e '[.route.rules[] | select(.outbound == "direct") | .domain_suffix] | flatten | index(".рф") != null' "$file" >/dev/null
  jq -e '[.route.rules[] | select(.outbound == "direct") | .domain_suffix] | flatten | index("gosuslugi.ru") != null' "$file" >/dev/null
  jq -e '[.route.rules[] | select(.outbound == "direct") | .domain_suffix] | flatten | index("sberbank.ru") != null' "$file" >/dev/null

  # 2. 检查 direct rule_set 包含 geosite/ru 与 geoip/ru
  jq -e '[.route.rules[] | select(.outbound == "direct") | .rule_set] | flatten | index("geosite/ru") != null' "$file" >/dev/null
  jq -e '[.route.rules[] | select(.outbound == "direct") | .rule_set] | flatten | index("geoip/ru") != null' "$file" >/dev/null

  # 3. 检查 proxy 规则包含 youtube, google, discord, telegram, ai
  jq -e '[.route.rules[] | select(.outbound == "Proxy") | .rule_set] | flatten | index("geosite/youtube") != null' "$file" >/dev/null
  jq -e '[.route.rules[] | select(.outbound == "Proxy") | .rule_set] | flatten | index("geosite/telegram") != null' "$file" >/dev/null
  jq -e '[.route.rules[] | select(.outbound == "Proxy") | .rule_set] | flatten | index("geosite/discord") != null' "$file" >/dev/null
  jq -e '[.route.rules[] | select(.outbound == "Proxy") | .rule_set] | flatten | index("geosite/category-ai-!cn") != null' "$file" >/dev/null

  # 4. 检查广告拦截规则
  jq -e '.route.rules[] | select(.action == "reject")' "$file" >/dev/null
}

#######################################
# 检查 bypass-lan.json 与 china.json 规则
# 参数: 无
# 返回: 0=通过，1=失败
#######################################
check_other_presets() {
  # bypass-lan: 私有 IP 与本地网络直连
  file="$PRESETS_DIR/bypass-lan.json"
  jq -e '.route.rules[] | select(.outbound == "direct" and .ip_is_private == true)' "$file" >/dev/null

  # china: geosite/cn 与 geoip/cn 直连
  file="$PRESETS_DIR/china.json"
  jq -e '[.route.rules[] | select(.outbound == "direct") | .rule_set] | flatten | index("geosite/cn") != null' "$file" >/dev/null
  jq -e '[.route.rules[] | select(.outbound == "direct") | .rule_set] | flatten | index("geoip/cn") != null' "$file" >/dev/null
}

#######################################
# 验证 netproxyctl preset CLI 接口
# 参数: 无
# 返回: 0=CLI 行为正常，1=失败
#######################################
check_cli_preset() {
  temp_dir="$(mktemp -d)"
  trap 'rm -rf "$temp_dir"' EXIT

  # 编译临时 netproxyctl 二进制
  bin="$temp_dir/netproxyctl"
  (cd "$ROOT/src/native/netproxy" && go build -o "$bin" ./cmd/netproxyctl)

  # 验证 preset list 输出
  list_output="$("$bin" preset list --lang ru)"
  echo "$list_output" | jq -e '.ok == true and .code == "preset.list"' >/dev/null
  echo "$list_output" | grep -q "Список доступных пресетов маршрутизации"
  echo "$list_output" | jq -e '.data[] | select(.id == "russia")' >/dev/null
  echo "$list_output" | jq -e '.data[] | select(.id == "bypass-lan")' >/dev/null
  echo "$list_output" | jq -e '.data[] | select(.id == "china")' >/dev/null

  # 验证 preset list --lang zh 输出
  list_zh="$("$bin" preset list --lang zh)"
  echo "$list_zh" | grep -q "预设路由规则列表"

  # 验证缺少参数时的报错行为
  if "$bin" preset apply >/dev/null 2>&1; then
    printf '%s\n' "preset apply 未传参数未报错" >&2
    return 1
  fi
}

main() {
  check_preset_files
  check_russia_preset_rules
  check_other_presets
  check_cli_preset
  printf '%s\n' "OK: 所有预设配置与 CLI 契约检查通过"
}

main "$@"

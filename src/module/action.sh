#!/system/bin/sh
#######################################
# 文件: action.sh
# 功能: 模块管理器中的操作按钮入口，交由 netproxyctl 切换服务状态。
# 用法: 由 Magisk/KernelSU/APatch 管理器点击模块操作按钮时调用。
# 依赖: netproxyctl、su
#######################################

readonly MODDIR="${0%/*}"
readonly NETPROXY_CTL="$MODDIR/netproxyctl"

#######################################
# 检测当前界面的显示语言 (zh / ru / en)。
# 参数: 无
# 返回: 标准输出打印语言代码，默认回退为 en。
#######################################
detect_locale() {
  local raw="${NETPROXY_LANG:-}"
  if [ -z "$raw" ] && command -v getprop > /dev/null 2>&1; then
    raw="$(getprop persist.sys.locale 2> /dev/null)"
    [ -n "$raw" ] || raw="$(getprop ro.product.locale 2> /dev/null)"
    [ -n "$raw" ] || raw="$(getprop persist.sys.language 2> /dev/null)"
    [ -n "$raw" ] || raw="$(getprop ro.product.locale.language 2> /dev/null)"
  fi
  case "$raw" in
    ru*|RU*) printf 'ru\n' ;;
    zh*|ZH*) printf 'zh\n' ;;
    *) printf 'en\n' ;;
  esac
}

case "$(detect_locale)" in
  ru)
    MSG_MISSING_CTL='Отсутствует netproxyctl, невозможно выполнить операцию сервиса.'
    MSG_MANAGER_ONLY='action.sh предназначен только для вызова из менеджера модулей.'
    MSG_HEADER=' Управление сервисом NetProxy'
    MSG_TOGGLE_OK=' Результат: состояние сервиса NetProxy переключено'
    MSG_TOGGLE_FAIL=' Результат: не удалось переключить состояние сервиса NetProxy'
    ;;
  zh)
    MSG_MISSING_CTL='缺少 netproxyctl，无法执行服务操作。'
    MSG_MANAGER_ONLY='action.sh 仅供模块管理器调用。'
    MSG_HEADER=' NetProxy 服务操作'
    MSG_TOGGLE_OK=' 操作结果: NetProxy 服务状态已切换'
    MSG_TOGGLE_FAIL=' 操作结果: NetProxy 服务切换失败'
    ;;
  *)
    MSG_MISSING_CTL='Missing netproxyctl, unable to perform service action.'
    MSG_MANAGER_ONLY='action.sh is only intended to be called by the module manager.'
    MSG_HEADER=' NetProxy Service Action'
    MSG_TOGGLE_OK=' Result: NetProxy service state toggled'
    MSG_TOGGLE_FAIL=' Result: Failed to toggle NetProxy service'
    ;;
esac

[ -x "$NETPROXY_CTL" ] || {
  printf '%s\n' "$MSG_MISSING_CTL" >&2
  exit 1
}

[ "$#" -eq 0 ] || {
  printf '%s\n' "$MSG_MANAGER_ONLY" >&2
  exit 2
}

printf '%s\n' '==================================='
printf '%s\n' "$MSG_HEADER"
printf '%s\n' '==================================='

# 服务状态与生命周期由 netproxyctl/Go 统一处理，Shell 不读取进程或 JSON 推断状态。
if command -v su > /dev/null 2>&1; then
  su -c "\"$NETPROXY_CTL\" service toggle" > /dev/null
  status=$?
else
  "$NETPROXY_CTL" service toggle > /dev/null
  status=$?
fi

if [ "$status" -eq 0 ]; then
  printf '%s\n' "$MSG_TOGGLE_OK"
else
  printf '%s\n' "$MSG_TOGGLE_FAIL"
fi
printf '%s\n' '==================================='
exit "$status"

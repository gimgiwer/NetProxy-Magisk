#!/system/bin/sh
#######################################
# 文件: emulated-soft-reboot.sh
# 功能: KernelSU 软重启前停止 Worker 与 sing-box，释放 eBPF 挂载。
# 用法: 由 KernelSU emulated-soft-reboot 阶段同步调用。
# 依赖: bin/netproxyctl
#######################################

readonly MODDIR="$(cd "$(dirname "$0")" && pwd)"
readonly NETPROXY_BIN="$MODDIR/bin/netproxyctl"

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
    MSG_BIN_NOT_EXEC='Не удалось остановить NetProxy перед мягкой перезагрузкой KernelSU: netproxyctl не является исполняемым.'
    MSG_STOP_WORKER_FAIL='Не удалось остановить фоновый воркер NetProxy перед мягкой перезагрузкой KernelSU.'
    MSG_STOP_SINGBOX_FAIL='Не удалось остановить sing-box перед мягкой перезагрузкой KernelSU.'
    ;;
  zh)
    MSG_BIN_NOT_EXEC='KernelSU 软重启前无法停止 NetProxy：netproxyctl 不可执行。'
    MSG_STOP_WORKER_FAIL='KernelSU 软重启前停止 NetProxy Worker 失败。'
    MSG_STOP_SINGBOX_FAIL='KernelSU 软重启前停止 sing-box 失败。'
    ;;
  *)
    MSG_BIN_NOT_EXEC='Cannot stop NetProxy before KernelSU soft-reboot: netproxyctl is not executable.'
    MSG_STOP_WORKER_FAIL='Failed to stop NetProxy Worker before KernelSU soft-reboot.'
    MSG_STOP_SINGBOX_FAIL='Failed to stop sing-box before KernelSU soft-reboot.'
    ;;
esac

[ -x "$NETPROXY_BIN" ] || {
  printf '%s\n' "$MSG_BIN_NOT_EXEC" >&2
  exit 1
}

status=0

NETPROXY_MODULE_DIR="$MODDIR" "$NETPROXY_BIN" __internal worker stop \
  --module-dir "$MODDIR" > /dev/null || {
  printf '%s\n' "$MSG_STOP_WORKER_FAIL" >&2
  status=1
}

NETPROXY_MODULE_DIR="$MODDIR" "$NETPROXY_BIN" service stop > /dev/null || {
  printf '%s\n' "$MSG_STOP_SINGBOX_FAIL" >&2
  status=1
}

exit "$status"

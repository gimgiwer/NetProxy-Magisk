#!/system/bin/sh
#######################################
# 文件: service.sh
# 功能: Magisk/KernelSU/APatch service 阶段启动桥接。
# 用法: 由模块框架无参数调用。
# 依赖: su、netproxyctl
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

[ "$#" -eq 0 ] || {
  case "$(detect_locale)" in
    ru)
      printf '%s\n' 'service.sh предназначен только для этапа запуска модуля; используйте netproxyctl service для управления сервисом.' >&2
      ;;
    zh)
      printf '%s\n' 'service.sh 仅供模块 service 阶段调用；请使用 netproxyctl service 管理服务。' >&2
      ;;
    *)
      printf '%s\n' 'service.sh is only for the module service stage; use netproxyctl service to manage the service.' >&2
      ;;
  esac
  exit 2
}

[ -x "$NETPROXY_BIN" ] || exit 1

# 通过 su 启动原生进程；sing-box 的最终 cgroup 归属由 Go 在启动时显式收敛。
if command -v su > /dev/null 2>&1; then
  exec su -c "\"$NETPROXY_BIN\" __internal boot --module-dir \"$MODDIR\""
fi
exec "$NETPROXY_BIN" __internal boot --module-dir "$MODDIR"

#!/bin/bash

# =============================
# 内部函数：输出到 FD3
# =============================
__emit_fd3() {
    local json="$1"
    local fd="${EWORK_RESULT_FD:-3}"

    # 优雅探测目标描述符是否打开并可写（避免本地调试泄漏 Bad file descriptor 警告）
    if { true >&$fd; } 2>/dev/null; then
        echo "$json" >&$fd
        return 0
    fi

    # 本地无对应 FD 时的调试 fallback（输出到 stderr 避免污染标准输出）
    echo "[DEBUG] FD $fd not available, result: $json" >&2
}

# =============================
# 模式 1: want_result 输出任务结果
# 用法 A（键值对）：want_result key1 value1 key2 value2 ...
# 用法 B（完整JSON）：want_result '{"status":"ok", "notifications":[...]}'
# =============================
want_result() {
    # 1. 单参数直通模式：直接传入合法的 JSON 对象或数组（如 jq 产物）
    if [ $# -eq 1 ]; then
        local trimmed="${1#"${1%%[![:space:]]*}"}"
        case "$trimmed" in
            "{"*|"["*)
                __emit_fd3 "$1"
                return 0
                ;;
        esac
    fi

    # 2. 键值对校验
    if [ $(($# % 2)) -ne 0 ]; then
        echo "want_result 参数必须是单个 JSON 字符串或成对的 key-value" >&2
        return 1
    fi

    # 3. 构造 JSON 字符串
    local json='{'
    local first=1

    while [ $# -gt 0 ]; do
        local key="$1"
        local value="$2"
        key=$(printf '%s' "$key" | sed 's/\"/\\\"/g')

        [ $first -eq 0 ] && json+=","

        # 若 value 本身为 JSON 对象或数组，保持原生嵌套结构不打外层双引号
        local val_trimmed="${value#"${value%%[![:space:]]*}"}"
        case "$val_trimmed" in
            "{"*"}"|"["*"]")
                json+="\"$key\":$value"
                ;;
            *)
                value=$(printf '%s' "$value" | sed 's/\"/\\\"/g')
                json+="\"$key\":\"$value\""
                ;;
        esac

        first=0
        shift 2
    done

    json+="}"
    __emit_fd3 "$json"
}

# =============================
# 模式 2: 逐步构建 JSON（兼容历史老代码）
# =============================
__result_json="{"

add_to_json() {
    local key="$1"
    local value="$2"

    [ "$__result_json" != "{" ] && __result_json+=","

    key=$(printf '%s' "$key" | sed 's/\"/\\\"/g')
    local val_trimmed="${value#"${value%%[![:space:]]*}"}"
    case "$val_trimmed" in
        "{"*"}"|"["*"]")
            __result_json+="\"$key\":$value"
            ;;
        *)
            value=$(printf '%s' "$value" | sed 's/\"/\\\"/g')
            __result_json+="\"$key\":\"$value\""
            ;;
    esac
}

finalize_json() {
    __result_json+="}"
    __emit_fd3 "$__result_json"
    __result_json="{"
}

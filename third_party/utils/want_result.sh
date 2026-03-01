#!/bin/bash

# =============================
# 内部函数：输出到 FD3
# =============================
__emit_fd3() {
    local json="$1"

    # 检查 FD 是否可用（兼容 Linux 和 macOS）
    if [ -n "$EWORK_RESULT_FD" ]; then
        # 尝试写入 FD，如果失败则 fallback
        if echo "$json" >&$EWORK_RESULT_FD 2>/dev/null; then
            return 0
        fi
    fi

    # 本地调试 fallback（输出到 stderr 避免污染 stdout）
    echo "[DEBUG] FD3 not available, result: $json" >&2
}

# =============================
# 模式 1: want_result 一次性传入所有键值对
# 用法：want_result key1 value1 key2 value2 ...
# =============================
want_result() {
    if [ $(($# % 2)) -ne 0 ]; then
        echo "want_result 参数必须是 key value 成对出现" >&2
        return 1
    fi

    # 直接构建扁平的 JSON 对象（不包含 type 和 data 层级）
    local json='{'
    local first=1

    while [ $# -gt 0 ]; do
        local key="$1"
        local value="$2"

        # JSON 转义（简单安全处理）
        key=$(printf '%s' "$key" | sed 's/\"/\\\"/g')
        value=$(printf '%s' "$value" | sed 's/\"/\\\"/g')

        if [ $first -eq 0 ]; then
            json+=","
        fi

        json+="\"$key\":\"$value\""
        first=0

        shift 2
    done

    json+="}"

    __emit_fd3 "$json"
}

# =============================
# 模式 2: 逐步构建 JSON（兼容旧代码）
# 用法：
#   add_to_json key1 value1
#   add_to_json key2 value2
#   finalize_json
# =============================

# 初始化 JSON 字符串（全局变量）
__result_json="{"

# 添加键值对到 JSON 对象
add_to_json() {
    local key="$1"
    local value="$2"

    # 添加逗号分隔符
    if [ "$__result_json" != "{" ]; then
        __result_json+=","
    fi

    # JSON 转义（简单安全处理）
    key=$(printf '%s' "$key" | sed 's/\"/\\\"/g')
    value=$(printf '%s' "$value" | sed 's/\"/\\\"/g')

    # 将键值对格式化为 JSON
    __result_json+="\"$key\":\"$value\""
}

# 结束 JSON 字符串并输出到 FD3
finalize_json() {
    __result_json+="}"
    __emit_fd3 "$__result_json"
    # 重置以便下次使用
    __result_json="{"
}

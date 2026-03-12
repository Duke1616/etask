#!/bin/bash

# ==============================================================================
# ETask 任务调试助手
# 作用: 模拟 Agent 环境运行归档的任务脚本
# ==============================================================================

set -o pipefail

# --- 样式定义 ---
readonly CLR_RED='\033[0;31m'
readonly CLR_GRN='\033[0;32m'
readonly CLR_BLU='\033[0;34m'
readonly CLR_YLW='\033[0;33m'
readonly CLR_NC='\033[0m'

# --- 辅助函数 ---
log() {
    local level=$1
    local msg=$2
    local color=${3:-$CLR_BLU}
    printf "${color}[%s]${CLR_NC} %s\n" "$level" "$msg"
}

print_divider() {
    printf "${CLR_BLU}%s${CLR_NC}\n" "------------------------------------------------------------"
}

# --- 核心逻辑 ---

# 运行 Shell 任务
execute_shell() {
    local dir=$1 script=$2 args=$3 trace=$4
    local vars_file="$dir/scripts.vars"
    local opts=("-e")

    [[ "$trace" == "true" ]] && opts+=("-x")
    [[ ! -f "$vars_file" ]] && vars_file=""

    log "引擎" "Shell (故障即止模式)"
    print_divider
    
    bash "${opts[@]}" "$script" "$args" "$vars_file"
}

# 运行 Python 任务
execute_python() {
    local dir=$1 script=$2 args=$3
    local vars_file="$dir/scripts.vars.json"
    local vars_content=""

    [[ -f "$vars_file" ]] && vars_content=$(cat "$vars_file")
    
    log "引擎" "Python 3"
    print_divider
    
    python3 "$script" "$args" "$vars_content"
}

# 调试主逻辑
run_debug() {
    local target_dir=$1
    local trace=$2

    # 1. 验证目录
    if [[ -z "$target_dir" || ! -d "$target_dir" ]]; then
        log "错误" "无效的目录路径: $target_dir" "$CLR_RED"
        return 1
    fi

    log "准备" "进入复现目录: $target_dir"

    # 2. 对齐 Agent 环境
    export EWORK_RESULT_FD=2
    export FORCE_COLOR=1
    export PYTHONUNBUFFERED=1

    # 3. 准备共用参数
    local args=""
    [[ -f "$target_dir/scripts.args" ]] && args=$(cat "$target_dir/scripts.args")

    # 4. 识别脚本类型并执行
    local rc=0
    if [[ -f "$target_dir/scripts.py" ]]; then
        execute_python "$target_dir" "$target_dir/scripts.py" "$args"
        rc=$?
    elif [[ -f "$target_dir/scripts.sh" ]]; then
        execute_shell "$target_dir" "$target_dir/scripts.sh" "$args" "$trace"
        rc=$?
    else
        log "错误" "未发现 scripts.sh 或 scripts.py" "$CLR_RED"
        return 1
    fi

    # 5. 结果反馈
    print_divider
    if [[ $rc -eq 0 ]]; then
        log "完成" "执行成功" "$CLR_GRN"
    else
        log "失败" "执行中途异常退出 (状态码: $rc)" "$CLR_RED"
    fi
}

# --- 程序入口 ---

usage() {
    echo "使用说明:"
    echo "  $0 [选项] <归档目录>"
    echo ""
    echo "选项:"
    echo "  -x, --trace    开启 Shell 指令追踪模式"
    echo "  -h, --help     显示此帮助信息"
    exit 1
}

main() {
    local target_dir=""
    local trace="false"

    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -x|--trace) trace="true"; shift ;;
            -h|--help) usage ;;
            *) target_dir="$1"; shift ;;
        esac
    done

    [[ -z "$target_dir" ]] && usage

    run_debug "$target_dir" "$trace"
}

main "$@"

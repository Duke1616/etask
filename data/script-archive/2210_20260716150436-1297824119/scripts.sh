#!/bin/bash

set -e

args=$(<"$ETASK_ARGS_FILE")
source "$ETASK_SYSTEM_ROOT/third_party/utils/want_result.sh"


# 脚本主体
main() {
    # 脚本的主要逻辑
    # want_result
    want_result \
        "test" "123" \
        "www" "com"
    echo "开始执行 SMOKE 任务"
    
    echo "执行中..."
    
    want_result \
        "SMOKE任务ID" "123" \
        "SMOKE报告地址" "https://baidu.com"
    
    echo "任务结束"
}

main $@

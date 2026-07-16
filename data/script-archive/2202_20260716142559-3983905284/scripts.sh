#!/bin/bash
# 脚本描述信息

## 传递工单提交信息
args=$1

pwd

## 为了防止重复编写脚本，设定环境变量机制，变量请通过 Runner 模块进行自定义配置
## 存储在临时文件中，通过 source 导入
vars=$2
source $vars
source ./third_party/utils/want_result.sh
# sleep 10

# 返回值
# want_result() {
#     echo $args
#     add_to_json "SMOKE任务ID" "123"
#     add_to_json "SMOKE报告地址" "https://baidu.com"
#     finalize_json
#     echo "$json"
# }



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

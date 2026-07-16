#!/bin/bash
# 脚本描述: 演示 ework-runner 的 Shell 任务执行能力

## 传递工单提交信息
args=$1

## 为了防止重复编写脚本,设定环境变量机制,变量请通过 Runner 模块进行自定义配置
## 存储在临时文件中,通过 source 导入
## 使用注入变量: KUBECONFIG_PATH, OPERATOR_NAME, ENVIRONMENT
vars=$2
source $vars

echo "========================================="
echo "开始执行 Shell 任务"
echo "========================================="

## 全局变量
date=$(date +%Y%m%d%H%M%S)
log_file="task-${date}.log"

## 从 JSON 参数中提取业务参数
pod_name=$(echo "$args" | jq -r '.pod_name')
namespace=$(echo "$args" | jq -r '.namespace')
user_info=$(echo "$args" | jq -r '.user_info')

echo "任务参数信息:"
echo "  - 命名空间: $namespace"
echo "  - Pod 名称: $pod_name"
echo "  - 用户信息: $user_info"
echo ""

echo "环境变量信息:"
echo "  - 操作人员: $OPERATOR_NAME"
echo "  - 运行环境: $ENVIRONMENT"
echo "  - Kubeconfig: $KUBECONFIG_PATH"
echo ""

## 模拟 Kubernetes 操作
echo "开始执行 Kubernetes 操作..."
echo "[1/5] 检查 Pod 状态"
# kubectl --kubeconfig=$KUBECONFIG_PATH get pod $pod_name -n $namespace
echo "  ✓ Pod 状态检查完成"
sleep 1

echo "[2/5] 获取 Pod 详细信息"
# kubectl --kubeconfig=$KUBECONFIG_PATH describe pod $pod_name -n $namespace
echo "  ✓ Pod 详细信息获取完成"
sleep 1

echo "[3/5] 查看 Pod 日志"
# kubectl --kubeconfig=$KUBECONFIG_PATH logs $pod_name -n $namespace --tail=100
echo "  ✓ Pod 日志查看完成"
sleep 1

echo "[4/5] 检查资源使用情况"
# kubectl --kubeconfig=$KUBECONFIG_PATH top pod $pod_name -n $namespace
echo "  ✓ 资源使用情况检查完成"
sleep 1

echo "[5/5] 生成操作报告"
cat > $log_file <<EOF
操作报告
========================================
操作时间: $(date '+%Y-%m-%d %H:%M:%S')
操作人员: $OPERATOR_NAME
运行环境: $ENVIRONMENT
命名空间: $namespace
Pod 名称: $pod_name
操作状态: 成功
========================================
EOF
echo "  ✓ 操作报告生成完成: $log_file"

echo ""
echo "========================================="
echo "Shell 任务执行完成!"
echo "========================================="
exit 0

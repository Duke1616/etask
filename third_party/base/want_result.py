# -*- coding: utf-8 -*-
"""
ETask / EWork 任务执行结果输出工具（FD3 协议）
100% 向后兼容历史的所有调用模式，增强类型序列化与传参自适应能力。
"""

import json
import os
import sys
from datetime import date, datetime


def _json_default(obj):
    """序列化安全兜底：自动处理 datetime、set 等常用对象，防止脚本意外报错退出"""
    if isinstance(obj, (datetime, date)):
        return obj.isoformat()
    if isinstance(obj, set):
        return list(obj)
    if hasattr(obj, "__dict__"):
        return obj.__dict__
    return str(obj)


def _emit_fd3(data):
    """内部函数：通过原子系统调用将 JSON 写回 FD3 通道"""
    try:
        json_bytes = (json.dumps(data, ensure_ascii=False, default=_json_default) + '\n').encode('utf-8')
    except Exception as e:
        json_bytes = json.dumps({"error": f"JSON 序列化失败: {e}"}, ensure_ascii=False).encode('utf-8') + b'\n'

    # 1. 优先读取系统注入的结果文件描述符（默认为 3）
    fd_str = os.environ.get('EWORK_RESULT_FD', '3')
    if fd_str and fd_str.isdigit():
        try:
            os.write(int(fd_str), json_bytes)
            return
        except OSError:
            pass

    # 2. 本地无 FD3 时的调试 fallback（输出到 stderr 避免污染标准输出）
    print(f"[DEBUG] FD3 not available, result: {json_bytes.decode('utf-8', errors='ignore').strip()}", file=sys.stderr)


def want_result(*args, **kwargs):
    """
    写回 FD3 任务执行产物（完全兼容全部调用模式）：
    - 历史模式 1: want_result(status="ok", count=10)
    - 历史模式 2: want_result({"status": "ok"})
    - 进阶模式 3: want_result("AD巡检完成", notifications=[...]) -> 首个字符串自动作为 summary
    - 进阶模式 4: want_result([...]) -> 传入纯列表时自动收敛为 {"notifications": [...]}
    """
    payload = {}
    if args:
        first = args[0]
        if isinstance(first, dict):
            payload.update(first)
        elif isinstance(first, (list, tuple)):
            payload["notifications"] = list(first)
        elif isinstance(first, str):
            payload["summary"] = first
        else:
            payload["data"] = first

    payload.update(kwargs)
    _emit_fd3(payload)
    return payload


class JsonBuilder:
    """
    模式 2: 逐步构建 JSON（完全保持与老版本兼容）
    用法：
        builder = JsonBuilder()
        builder.add_to_json('key1', 'value1')
        builder.finalize_json()
    """
    def __init__(self):
        self.data = {}

    def add_to_json(self, key, value):
        self.data[key] = value

    def finalize_json(self):
        _emit_fd3(self.data)
        res = self.data
        self.data = {}
        return res

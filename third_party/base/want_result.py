# -*- coding: utf-8 -*-
"""ETask 任务执行结果输出工具（FD3 协议）"""

import json
import os
import sys
from datetime import date, datetime


def _json_default(obj):
    """基础对象安全序列化兜底"""
    if isinstance(obj, (datetime, date)):
        return obj.isoformat()
    if isinstance(obj, set):
        return list(obj)
    if hasattr(obj, "__dict__"):
        return obj.__dict__
    return str(obj)


def _emit_fd3(data):
    """原子写回 FD3 通道"""
    raw = json.dumps(data, ensure_ascii=False, default=_json_default) + "\n"
    try:
        fd = int(os.environ.get("EWORK_RESULT_FD", "3"))
        os.write(fd, raw.encode("utf-8"))
    except (ValueError, OSError):
        # 本地调试 fallback（无对应 FD 时打印到 stderr）
        print(f"[DEBUG] FD3 not available, result: {raw.strip()}", file=sys.stderr)


def want_result(*args, **kwargs):
    """
    通用写回 FD3 任务执行结果。
    完全兼容所有历史调用模式：
    - 键值对：want_result(status="ok", count=10)
    - 字典：want_result({"status": "ok"})
    - 文本带指标：want_result("巡检完成", total=10)
    - 原生列表：want_result([1, 2, 3])
    """
    if args:
        first = args[0]
        if isinstance(first, dict):
            payload = {**first, **kwargs}
        elif isinstance(first, str):
            payload = {"summary": first, **kwargs}
        else:
            payload = first
    else:
        payload = kwargs

    _emit_fd3(payload)
    return payload


class JsonBuilder:
    """兼容旧版渐进式构建模式"""

    def __init__(self):
        self.data = {}

    def add_to_json(self, key, value):
        self.data[key] = value

    def finalize_json(self):
        _emit_fd3(self.data)
        res, self.data = self.data, {}
        return res


__all__ = ["want_result", "JsonBuilder"]

import json
import os
import sys


def _emit_fd3(data):
    """内部函数：输出到 FD3"""
    json_str = json.dumps(data, ensure_ascii=False)

    # 检查 FD 是否可用（兼容 Linux 和 macOS）
    fd = os.environ.get('EWORK_RESULT_FD')
    if fd:
        try:
            # 尝试写入 FD
            with os.fdopen(int(fd), 'w') as f:
                f.write(json_str + '\n')
                f.flush()
            return
        except (ValueError, OSError):
            pass

    # 本地调试 fallback（输出到 stderr 避免污染 stdout）
    print(f"[DEBUG] FD3 not available, result: {json_str}", file=sys.stderr)


def want_result(**kwargs):
    """
    模式 1: 一次性传入所有键值对
    用法：want_result(key1=value1, key2=value2, ...)
    """
    _emit_fd3(kwargs)


class JsonBuilder:
    """
    模式 2: 逐步构建 JSON（兼容旧代码）
    用法：
        builder = JsonBuilder()
        builder.add_to_json('key1', 'value1')
        builder.add_to_json('key2', 'value2')
        builder.finalize_json()
    """
    def __init__(self):
        # 初始化一个字典来存储 JSON 键值对
        self.data = {}

    def add_to_json(self, key, value):
        # 添加键值对到字典中
        self.data[key] = value

    def finalize_json(self):
        # 输出到 FD3 并重置
        __emit_fd3(self.data)
        self.data = {}

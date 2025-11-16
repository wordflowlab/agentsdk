#!/usr/bin/env python3
"""
examples/skills/workspace/skills/pdfmd/pdf_extract.py

使用 Python 从 PDF 中提取纯文本（不调用任何大模型），将结果输出到 stdout。

依赖:
  pip install pypdf

用法示例 (在 examples/skills 根目录执行):
  python workspace/skills/pdfmd/pdf_extract.py --input docs/report.pdf
  python workspace/skills/pdfmd/pdf_extract.py --input docs/report.pdf --pages "1-3,5"
"""

import argparse
import os
import sys
from typing import Optional, Set

from pypdf import PdfReader


def parse_page_range(page_range: str, max_page: int) -> Optional[Set[int]]:
    """
    将类似 "1-3,5,7-8" 的字符串解析为页码集合。
    页码从 1 开始；超出范围的页码会被忽略。
    """
    if not page_range:
        return None

    result: Set[int] = set()
    parts = page_range.split(",")
    for part in parts:
        part = part.strip()
        if not part:
            continue
        if "-" in part:
            start_str, end_str = part.split("-", 1)
            try:
                start = int(start_str)
                end = int(end_str)
            except ValueError:
                continue
            if start > end:
                start, end = end, start
            for p in range(start, end + 1):
                if 1 <= p <= max_page:
                    result.add(p)
        else:
            try:
                p = int(part)
            except ValueError:
                continue
            if 1 <= p <= max_page:
                result.add(p)
    return result or None


def extract_pdf_text(path: str, page_range: str) -> str:
    """
    使用 pypdf 提取 PDF 文本，并按页加上分隔标记。
    """
    reader = PdfReader(path)
    page_count = len(reader.pages)
    pages = parse_page_range(page_range, page_count)

    chunks = []
    for i, page in enumerate(reader.pages, start=1):
        if pages is not None and i not in pages:
            continue
        text = page.extract_text() or ""
        chunks.append(f"\n===== 第 {i} 页 =====\n{text}")
    return "\n".join(chunks)


def main() -> None:
    parser = argparse.ArgumentParser(description="Extract plain text from a PDF file (no LLM calls).")
    parser.add_argument("--input", "-i", required=True, help="输入 PDF 文件路径")
    parser.add_argument("--pages", "-p", default="", help="页面范围，如 '1-3,5,7-8'")

    args = parser.parse_args()

    pdf_path = args.input
    if not os.path.exists(pdf_path):
        print(f"输入文件不存在: {pdf_path}", file=sys.stderr)
        sys.exit(1)

    try:
        pdf_text = extract_pdf_text(pdf_path, args.pages)
    except Exception as e:
        print(f"提取 PDF 文本失败: {e}", file=sys.stderr)
        sys.exit(1)

    # 直接输出到 stdout，供上层 bash_run 工具捕获。
    sys.stdout.write(pdf_text)


if __name__ == "__main__":
    main()

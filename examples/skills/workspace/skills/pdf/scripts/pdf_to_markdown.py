#!/usr/bin/env python3
"""
PDF to Markdown Converter
将PDF文档转换为Markdown格式
"""

import sys
import os
import argparse
from pathlib import Path

try:
    import pypdf
    import pdfplumber
except ImportError as e:
    print(f"错误：缺少必要的依赖 {e}")
    print("请运行：pip install pypdf pdfplumber")
    sys.exit(1)

def extract_text_with_pdfplumber(pdf_path):
    """使用pdfplumber提取文本（更好的布局保持）"""
    text_content = []

    try:
        with pdfplumber.open(pdf_path) as pdf:
            for page_num, page in enumerate(pdf.pages, 1):
                # 提取文本
                text = page.extract_text()
                if text:
                    text_content.append(f"\n## 第 {page_num} 页\n\n")
                    text_content.append(text)
                    text_content.append("\n---\n")

    except Exception as e:
        print(f"pdfplumber提取失败: {e}")
        return None

    return '\n'.join(text_content)

def extract_text_with_pypdf(pdf_path):
    """使用pypdf提取文本（备用方案）"""
    text_content = []

    try:
        with open(pdf_path, 'rb') as file:
            pdf_reader = pypdf.PdfReader(file)

            for page_num, page in enumerate(pdf_reader.pages, 1):
                text = page.extract_text()
                if text.strip():
                    text_content.append(f"\n## 第 {page_num} 页\n\n")
                    text_content.append(text)
                    text_content.append("\n---\n")

    except Exception as e:
        print(f"pypdf提取失败: {e}")
        return None

    return '\n'.join(text_content)

def extract_metadata(pdf_path):
    """提取PDF元数据"""
    try:
        with open(pdf_path, 'rb') as file:
            pdf_reader = pypdf.PdfReader(file)

            metadata = {
                'title': '',
                'author': '',
                'subject': '',
                'creator': '',
                'producer': '',
                'creation_date': '',
                'modification_date': '',
                'page_count': len(pdf_reader.pages)
            }

            if pdf_reader.metadata:
                meta = pdf_reader.metadata
                metadata.update({
                    'title': meta.get('/Title', ''),
                    'author': meta.get('/Author', ''),
                    'subject': meta.get('/Subject', ''),
                    'creator': meta.get('/Creator', ''),
                    'producer': meta.get('/Producer', ''),
                    'creation_date': str(meta.get('/CreationDate', '')),
                    'modification_date': str(meta.get('/ModDate', '')),
                })

            return metadata

    except Exception as e:
        print(f"提取元数据失败: {e}")
        return None

def pdf_to_markdown(pdf_path, output_path=None):
    """将PDF转换为Markdown"""

    pdf_path = Path(pdf_path)
    if not pdf_path.exists():
        print(f"错误：找不到PDF文件 {pdf_path}")
        return False

    # 确定输出路径
    if output_path is None:
        output_path = pdf_path.with_suffix('.md')
    else:
        output_path = Path(output_path)

    print(f"正在处理: {pdf_path}")
    print(f"输出文件: {output_path}")

    # 提取元数据
    print("正在提取元数据...")
    metadata = extract_metadata(pdf_path)

    # 提取文本内容
    print("正在提取文本内容...")

    # 首先尝试使用pdfplumber（更好的布局保持）
    text_content = extract_text_with_pdfplumber(pdf_path)

    # 如果pdfplumber失败，使用pypdf作为备用
    if text_content is None:
        print("pdfplumber失败，尝试使用pypdf...")
        text_content = extract_text_with_pypdf(pdf_path)

    if text_content is None:
        print("错误：无法提取PDF内容")
        return False

    # 构建Markdown内容
    markdown_content = []

    # 添加标题和元数据
    markdown_content.append(f"# {metadata.get('title', pdf_path.stem)}\n")

    if metadata:
        markdown_content.append("## 文档信息\n")
        markdown_content.append(f"- **文件名**: {pdf_path.name}\n")
        markdown_content.append(f"- **页数**: {metadata.get('page_count', 'N/A')}\n")

        if metadata.get('author'):
            markdown_content.append(f"- **作者**: {metadata['author']}\n")
        if metadata.get('subject'):
            markdown_content.append(f"- **主题**: {metadata['subject']}\n")
        if metadata.get('creation_date'):
            markdown_content.append(f"- **创建日期**: {metadata['creation_date']}\n")

        markdown_content.append("\n---\n")

    # 添加文档内容
    markdown_content.append("## 文档内容\n")
    markdown_content.append(text_content)

    # 写入文件
    try:
        with open(output_path, 'w', encoding='utf-8') as f:
            f.write(''.join(markdown_content))

        print(f"✅ 成功转换为Markdown: {output_path}")
        print(f"📄 文件大小: {output_path.stat().st_size:,} 字节")
        return True

    except Exception as e:
        print(f"❌ 写入文件失败: {e}")
        return False

def main():
    parser = argparse.ArgumentParser(description='将PDF文档转换为Markdown格式')
    parser.add_argument('pdf_file', help='PDF文件路径')
    parser.add_argument('-o', '--output', help='输出Markdown文件路径（可选）')
    parser.add_argument('-v', '--verbose', action='store_true', help='显示详细信息')

    args = parser.parse_args()

    if args.verbose:
        print(f"PDF文件: {args.pdf_file}")
        if args.output:
            print(f"输出文件: {args.output}")

    success = pdf_to_markdown(args.pdf_file, args.output)

    if success:
        print("🎉 转换完成！")
        sys.exit(0)
    else:
        print("💥 转换失败！")
        sys.exit(1)

if __name__ == '__main__':
    main()
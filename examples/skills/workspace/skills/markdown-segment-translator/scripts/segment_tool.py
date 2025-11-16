#!/usr/bin/env python3
"""
Markdown分段工具（纯工具函数，不调用API）
供Agent SDK使用：Agent负责翻译，此脚本只负责分段和合并
"""

import sys
import os
import json
import argparse
from pathlib import Path
from typing import List, Dict

class MarkdownSegmentTool:
    """Markdown分段和合并工具"""
    
    def __init__(self, output_dir: str = "output"):
        self.output_dir = Path(output_dir)
        self.segments_dir = self.output_dir / "segments"
        self.translations_dir = self.output_dir / "translations"
        self.final_dir = self.output_dir / "final"
        self.metadata_dir = self.output_dir / "metadata"
        
        # 创建目录
        for dir_path in [self.segments_dir, self.translations_dir, 
                         self.final_dir, self.metadata_dir]:
            dir_path.mkdir(parents=True, exist_ok=True)
    
    def segment_document(self, input_file: str, segment_size: int = 1000, 
                        max_segments: int = None) -> Dict:
        """
        将文档分段
        返回分段信息的JSON
        """
        print(f"正在分析文档: {input_file}")
        
        with open(input_file, 'r', encoding='utf-8') as f:
            lines = f.readlines()
        
        total_lines = len(lines)
        print(f"文档总行数: {total_lines}")
        
        # 计算分段 - 严格按照segment_size分段
        num_segments = (total_lines + segment_size - 1) // segment_size
        if max_segments and num_segments > max_segments:
            num_segments = max_segments
            # 保持segment_size不变，只限制segment数量
        
        actual_total_lines = min(num_segments * segment_size, total_lines)
        print(f"将分成 {num_segments} 个段落，每段 {segment_size} 行（最多处理 {actual_total_lines} 行）")
        
        segments_info = []
        
        # 创建分段文件
        for i in range(num_segments):
            start_line = i * segment_size
            end_line = min((i + 1) * segment_size, total_lines)
            
            segment_file = self.segments_dir / f"segment_{i+1}.md"
            with open(segment_file, 'w', encoding='utf-8') as f:
                f.writelines(lines[start_line:end_line])
            
            segment_info = {
                "index": i + 1,
                "file": str(segment_file),
                "start_line": start_line + 1,
                "end_line": end_line,
                "line_count": end_line - start_line
            }
            segments_info.append(segment_info)
            print(f"创建分段 {i+1}: segment_{i+1}.md ({segment_info['line_count']} 行)")
        
        # 保存元数据
        metadata = {
            "source_file": input_file,
            "total_lines": total_lines,
            "num_segments": num_segments,
            "segment_size": segment_size,
            "segments": segments_info
        }
        
        metadata_file = self.metadata_dir / "segments_info.json"
        with open(metadata_file, 'w', encoding='utf-8') as f:
            json.dump(metadata, f, ensure_ascii=False, indent=2)
        
        print(f"\n=== 分段完成 ===")
        print(f"分段文件位于: {self.segments_dir}")
        print(f"元数据保存至: {metadata_file}")
        
        return metadata
    
    def merge_translations(self, source_file: str = None) -> str:
        """
        合并翻译结果
        """
        print("\n正在合并翻译结果...")
        
        # 读取元数据
        metadata_file = self.metadata_dir / "segments_info.json"
        if not metadata_file.exists():
            raise FileNotFoundError(f"找不到元数据文件: {metadata_file}")
        
        with open(metadata_file, 'r', encoding='utf-8') as f:
            metadata = json.load(f)
        
        # 获取源文件名
        if not source_file:
            source_file = metadata.get('source_file', 'document')
        source_name = Path(source_file).stem
        
        # 合并所有翻译段落
        merged_content = []
        merged_content.append(f"# {source_name} 中文翻译\n\n")
        merged_content.append("**翻译信息**\n")
        merged_content.append(f"- 原文件: {source_file}\n")
        merged_content.append(f"- 总行数: {metadata['total_lines']}\n")
        merged_content.append(f"- 翻译段落数: {metadata['num_segments']}\n")
        
        from datetime import datetime
        merged_content.append(f"- 完成时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
        merged_content.append(f"- 翻译工具: Markdown分段翻译器\n\n")
        merged_content.append("---\n\n")
        
        # 按顺序读取每个翻译段落
        for seg_info in metadata['segments']:
            seg_index = seg_info['index']
            trans_file = self.translations_dir / f"translated_segment_{seg_index}.md"
            
            if not trans_file.exists():
                print(f"⚠️  警告: 找不到翻译文件 {trans_file}，跳过此段")
                continue
            
            with open(trans_file, 'r', encoding='utf-8') as f:
                content = f.read()
            
            merged_content.append(f"\n## 第 {seg_index} 段 (第 {seg_info['start_line']}-{seg_info['end_line']} 行)\n\n")
            merged_content.append(content)
            merged_content.append("\n---\n")
            
            print(f"合并段落 {seg_index}")
        
        # 保存合并结果
        output_file = self.final_dir / f"complete_translated_{source_name}.md"
        with open(output_file, 'w', encoding='utf-8') as f:
            f.writelines(merged_content)
        
        print(f"\n=== 合并完成 ===")
        print(f"最终翻译文件: {output_file}")
        
        return str(output_file)

def main():
    parser = argparse.ArgumentParser(description='Markdown分段工具（不调用API）')
    parser.add_argument('command', choices=['segment', 'merge'], 
                       help='命令: segment=分段, merge=合并')
    parser.add_argument('--input', required=False, help='输入文件路径（分段时需要）')
    parser.add_argument('--segment-size', type=int, default=1000, help='每段行数')
    parser.add_argument('--max-segments', type=int, help='最大段落数')
    parser.add_argument('--output-dir', default='output', help='输出目录')
    
    args = parser.parse_args()
    
    tool = MarkdownSegmentTool(args.output_dir)
    
    try:
        if args.command == 'segment':
            if not args.input:
                print("错误: 分段命令需要 --input 参数")
                sys.exit(1)
            metadata = tool.segment_document(args.input, args.segment_size, args.max_segments)
            print(f"\n✅ 分段成功！共 {metadata['num_segments']} 个段落")
            
        elif args.command == 'merge':
            output_file = tool.merge_translations(args.input)
            print(f"\n✅ 合并成功！结果: {output_file}")
    
    except Exception as e:
        print(f"❌ 错误: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

if __name__ == "__main__":
    main()

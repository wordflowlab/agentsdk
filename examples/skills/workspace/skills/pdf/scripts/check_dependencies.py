#!/usr/bin/env python3
"""
检查PDF技能所需的Python依赖
"""
import sys

def check_dependency(module_name, package_name=None):
    """检查单个依赖"""
    if package_name is None:
        package_name = module_name

    try:
        __import__(module_name)
        print(f"✓ {package_name} - 可用")
        return True
    except ImportError:
        print(f"✗ {package_name} - 未安装 (pip install {package_name})")
        return False

def main():
    print("PDF技能依赖检查")
    print("=" * 30)

    dependencies = [
        ("pypdf", "pypdf"),
        ("pdfplumber", "pdfplumber"),
        ("reportlab", "reportlab"),
        ("PIL", "Pillow"),  # for image processing
        ("json", None),  # standard library
        ("argparse", None),  # standard library
        ("os", None),  # standard library
    ]

    available = 0
    total = len(dependencies)

    for module, package in dependencies:
        if check_dependency(module, package):
            available += 1

    print(f"\n依赖检查完成: {available}/{total} 可用")

    if available < total:
        print("\n安装缺失的依赖:")
        print("pip install pypdf pdfplumber reportlab Pillow")
    else:
        print("\n所有依赖都已安装，PDF技能可以正常使用！")

if __name__ == "__main__":
    main()
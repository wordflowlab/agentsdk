---
name: pdf
description: 全面的PDF处理工具包，支持文本提取、表格处理、PDF创建、文档合并/分割以及表单处理。当需要填写PDF表单或以编程方式处理、生成或分析PDF文档时使用。
allowed-tools: ["Bash", "Write", "Read"]
triggers:
  - type: keyword
    keywords:
      - "pdf"
      - "pdf 处理"
      - "pdf 提取"
      - "pdf 合并"
      - "pdf 分割"
      - "pdf 表单"
      - "提取 pdf"
      - "pdf 转换"
      - "pdf 文本"
      - "pdf 表格"
      - "填写表单"
      - "处理 pdf"
      - "分析 pdf"
---

# PDF 处理技能完整指南

## ⚠️ 重要执行指导

**必须使用 `Bash` 工具执行Python脚本，不要试图直接处理PDF文件！**

### PDF转Markdown（最常用）
```bash
python workspace/skills/pdf/scripts/pdf_to_markdown.py [PDF文件名]
```

**示例**：
```bash
python workspace/skills/pdf/scripts/pdf_to_markdown.py 2407.14333v5.pdf
```

### 其他PDF操作
```bash
# 提取表单字段信息
python workspace/skills/pdf/scripts/extract_form_field_info.py [PDF文件名]

# 检查可填写字段
python workspace/skills/pdf/scripts/check_fillable_fields.py [PDF文件名]

# PDF转图像
python workspace/skills/pdf/scripts/convert_pdf_to_images.py [PDF文件名] [输出目录]
```

## 执行步骤
1. 使用 `Bash` 工具执行上述命令
2. 等待脚本执行完成
3. 使用 `Read` 查看生成的结果
4. 使用 `Write` 保存处理结果到指定位置

## 概述

本技能提供全面的PDF处理功能，包括文本提取、表格处理、PDF创建、文档合并/分割以及表单处理。技能包含8个专用的Python脚本，可处理各种PDF相关任务。

### 技能功能
- **文本提取**: 使用pypdf和pdfplumber提取PDF文本内容
- **表格处理**: 提取和转换PDF中的表格数据
- **PDF操作**: 合并、分割、旋转、加密PDF文档
- **表单处理**: 检测、填写和提取PDF表单字段
- **图像处理**: PDF转图像、边界框检查
- **高级功能**: OCR支持、水印添加、元数据处理

### 工作目录结构
```
workspace/skills/pdf/
├── SKILL.md                 # 本技能定义文件
├── reference.md             # 高级功能参考文档
├── forms.md                 # 表单处理专门指南
├── LICENSE.txt              # 许可证文件
└── scripts/                 # Python脚本目录
    ├── check_bounding_boxes.py      # 边界框检查
    ├── check_bounding_boxes_test.py # 边界框检查测试
    ├── check_fillable_fields.py     # 检查可填写字段
    ├── convert_pdf_to_images.py     # PDF转图像
    ├── create_validation_image.py   # 创建验证图像
    ├── extract_form_field_info.py   # 提取表单字段信息
    ├── fill_fillable_fields.py      # 填写可填写字段
    └── fill_pdf_form_with_annotations.py # 用注释填充表单
```

## Agentsdk 使用方法

### 使用 `Bash` 执行PDF脚本

通过 `Bash` 工具调用Python脚本：

```bash
# 检查PDF中的可填写字段
python skills/pdf/scripts/check_fillable_fields.py <PDF文件路径>

# 提取表单字段信息
python skills/pdf/scripts/extract_form_field_info.py <PDF文件路径>

# 将PDF转换为图像
python skills/pdf/scripts/convert_pdf_to_images.py <PDF文件路径> <输出目录>

# 填写PDF表单
python skills/pdf/scripts/fill_fillable_fields.py <PDF文件路径> <字段数据JSON>

# 检查边界框
python skills/pdf/scripts/check_bounding_boxes.py <PDF文件路径> <检查数据JSON>
```

### 使用 `Read` 和 `Write` 处理文件

```bash
# 读取PDF文件进行分析
# 使用 Read 工具读取PDF文件路径

# 保存处理结果
# 使用 Write 工具保存生成的Markdown、JSON或其他格式的结果
```

### 常见使用场景

#### 1. PDF转Markdown（推荐）
```bash
# 将PDF转换为Markdown格式（完整保留格式和结构）
python skills/pdf/scripts/pdf_to_markdown.py 2407.14333v5.pdf
```

#### 2. PDF文本提取和分析
```bash
# 提取PDF文本内容
python skills/pdf/scripts/extract_form_field_info.py document.pdf
```

#### 3. PDF表单处理
```bash
# 检查表单字段
python skills/pdf/scripts/check_fillable_fields.py form.pdf

# 提取字段信息
python skills/pdf/scripts/extract_form_field_info.py form.pdf

# 填写表单
python skills/pdf/scripts/fill_fillable_fields.py form.pdf field_data.json
```

#### 3. PDF图像处理
```bash
# PDF转图像
python skills/pdf/scripts/convert_pdf_to_images.py document.pdf output_images/
```

### 错误处理

- 确保PDF文件路径正确
- 检查Python依赖是否已安装（pypdf, pdfplumber, reportlab等）
- 验证输出目录权限
- 处理加密PDF时需要提供密码

---

## 传统PDF操作指南

以下为传统的PDF处理操作指南，适用于直接编程使用：

## Quick Start

```python
from pypdf import PdfReader, PdfWriter

# Read a PDF
reader = PdfReader("document.pdf")
print(f"Pages: {len(reader.pages)}")

# Extract text
text = ""
for page in reader.pages:
    text += page.extract_text()
```

## Python Libraries

### pypdf - Basic Operations

#### Merge PDFs
```python
from pypdf import PdfWriter, PdfReader

writer = PdfWriter()
for pdf_file in ["doc1.pdf", "doc2.pdf", "doc3.pdf"]:
    reader = PdfReader(pdf_file)
    for page in reader.pages:
        writer.add_page(page)

with open("merged.pdf", "wb") as output:
    writer.write(output)
```

#### Split PDF
```python
reader = PdfReader("input.pdf")
for i, page in enumerate(reader.pages):
    writer = PdfWriter()
    writer.add_page(page)
    with open(f"page_{i+1}.pdf", "wb") as output:
        writer.write(output)
```

#### Extract Metadata
```python
reader = PdfReader("document.pdf")
meta = reader.metadata
print(f"Title: {meta.title}")
print(f"Author: {meta.author}")
print(f"Subject: {meta.subject}")
print(f"Creator: {meta.creator}")
```

#### Rotate Pages
```python
reader = PdfReader("input.pdf")
writer = PdfWriter()

page = reader.pages[0]
page.rotate(90)  # Rotate 90 degrees clockwise
writer.add_page(page)

with open("rotated.pdf", "wb") as output:
    writer.write(output)
```

### pdfplumber - Text and Table Extraction

#### Extract Text with Layout
```python
import pdfplumber

with pdfplumber.open("document.pdf") as pdf:
    for page in pdf.pages:
        text = page.extract_text()
        print(text)
```

#### Extract Tables
```python
with pdfplumber.open("document.pdf") as pdf:
    for i, page in enumerate(pdf.pages):
        tables = page.extract_tables()
        for j, table in enumerate(tables):
            print(f"Table {j+1} on page {i+1}:")
            for row in table:
                print(row)
```

#### Advanced Table Extraction
```python
import pandas as pd

with pdfplumber.open("document.pdf") as pdf:
    all_tables = []
    for page in pdf.pages:
        tables = page.extract_tables()
        for table in tables:
            if table:  # Check if table is not empty
                df = pd.DataFrame(table[1:], columns=table[0])
                all_tables.append(df)

# Combine all tables
if all_tables:
    combined_df = pd.concat(all_tables, ignore_index=True)
    combined_df.to_excel("extracted_tables.xlsx", index=False)
```

### reportlab - Create PDFs

#### Basic PDF Creation
```python
from reportlab.lib.pagesizes import letter
from reportlab.pdfgen import canvas

c = canvas.Canvas("hello.pdf", pagesize=letter)
width, height = letter

# Add text
c.drawString(100, height - 100, "Hello World!")
c.drawString(100, height - 120, "This is a PDF created with reportlab")

# Add a line
c.line(100, height - 140, 400, height - 140)

# Save
c.save()
```

#### Create PDF with Multiple Pages
```python
from reportlab.lib.pagesizes import letter
from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer, PageBreak
from reportlab.lib.styles import getSampleStyleSheet

doc = SimpleDocTemplate("report.pdf", pagesize=letter)
styles = getSampleStyleSheet()
story = []

# Add content
title = Paragraph("Report Title", styles['Title'])
story.append(title)
story.append(Spacer(1, 12))

body = Paragraph("This is the body of the report. " * 20, styles['Normal'])
story.append(body)
story.append(PageBreak())

# Page 2
story.append(Paragraph("Page 2", styles['Heading1']))
story.append(Paragraph("Content for page 2", styles['Normal']))

# Build PDF
doc.build(story)
```

## Command-Line Tools

### pdftotext (poppler-utils)
```bash
# Extract text
pdftotext input.pdf output.txt

# Extract text preserving layout
pdftotext -layout input.pdf output.txt

# Extract specific pages
pdftotext -f 1 -l 5 input.pdf output.txt  # Pages 1-5
```

### qpdf
```bash
# Merge PDFs
qpdf --empty --pages file1.pdf file2.pdf -- merged.pdf

# Split pages
qpdf input.pdf --pages . 1-5 -- pages1-5.pdf
qpdf input.pdf --pages . 6-10 -- pages6-10.pdf

# Rotate pages
qpdf input.pdf output.pdf --rotate=+90:1  # Rotate page 1 by 90 degrees

# Remove password
qpdf --password=mypassword --decrypt encrypted.pdf decrypted.pdf
```

### pdftk (if available)
```bash
# Merge
pdftk file1.pdf file2.pdf cat output merged.pdf

# Split
pdftk input.pdf burst

# Rotate
pdftk input.pdf rotate 1east output rotated.pdf
```

## Common Tasks

### Extract Text from Scanned PDFs
```python
# Requires: pip install pytesseract pdf2image
import pytesseract
from pdf2image import convert_from_path

# Convert PDF to images
images = convert_from_path('scanned.pdf')

# OCR each page
text = ""
for i, image in enumerate(images):
    text += f"Page {i+1}:\n"
    text += pytesseract.image_to_string(image)
    text += "\n\n"

print(text)
```

### Add Watermark
```python
from pypdf import PdfReader, PdfWriter

# Create watermark (or load existing)
watermark = PdfReader("watermark.pdf").pages[0]

# Apply to all pages
reader = PdfReader("document.pdf")
writer = PdfWriter()

for page in reader.pages:
    page.merge_page(watermark)
    writer.add_page(page)

with open("watermarked.pdf", "wb") as output:
    writer.write(output)
```

### Extract Images
```bash
# Using pdfimages (poppler-utils)
pdfimages -j input.pdf output_prefix

# This extracts all images as output_prefix-000.jpg, output_prefix-001.jpg, etc.
```

### Password Protection
```python
from pypdf import PdfReader, PdfWriter

reader = PdfReader("input.pdf")
writer = PdfWriter()

for page in reader.pages:
    writer.add_page(page)

# Add password
writer.encrypt("userpassword", "ownerpassword")

with open("encrypted.pdf", "wb") as output:
    writer.write(output)
```

## Quick Reference

| Task | Best Tool | Command/Code |
|------|-----------|--------------|
| Merge PDFs | pypdf | `writer.add_page(page)` |
| Split PDFs | pypdf | One page per file |
| Extract text | pdfplumber | `page.extract_text()` |
| Extract tables | pdfplumber | `page.extract_tables()` |
| Create PDFs | reportlab | Canvas or Platypus |
| Command line merge | qpdf | `qpdf --empty --pages ...` |
| OCR scanned PDFs | pytesseract | Convert to image first |
| Fill PDF forms | pdf-lib or pypdf (see forms.md) | See forms.md |

## Next Steps

- For advanced pypdfium2 usage, see reference.md
- For JavaScript libraries (pdf-lib), see reference.md
- If you need to fill out a PDF form, follow the instructions in forms.md
- For troubleshooting guides, see reference.md

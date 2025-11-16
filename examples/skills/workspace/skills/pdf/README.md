# PDF 处理技能

## 简介

这是一个完整的PDF处理技能，从Claude的document-skills复制并适配到WordflowLab AgentSDK。该技能提供了全面的PDF处理功能，包括文本提取、表格处理、PDF创建、文档操作和表单处理。

## 功能特性

### 核心功能
- ✅ **文本提取**: 使用pypdf和pdfplumber提取PDF文本内容
- ✅ **表格处理**: 提取和转换PDF中的表格数据
- ✅ **PDF操作**: 合并、分割、旋转、加密PDF文档
- ✅ **表单处理**: 检测、填写和提取PDF表单字段
- ✅ **图像处理**: PDF转图像、边界框检查
- ✅ **高级功能**: OCR支持、水印添加、元数据处理

### 包含的Python脚本
- `check_bounding_boxes.py` - 边界框检查和验证
- `check_fillable_fields.py` - 检查PDF中的可填写字段
- `convert_pdf_to_images.py` - 将PDF转换为图像文件
- `create_validation_image.py` - 创建验证用的图像
- `extract_form_field_info.py` - 提取表单字段详细信息
- `fill_fillable_fields.py` - 填写PDF表单字段
- `fill_pdf_form_with_annotations.py` - 使用注释填充表单
- `check_dependencies.py` - 检查所需依赖是否已安装

## 安装要求

### Python依赖
```bash
pip install pypdf pdfplumber reportlab Pillow
```

### 系统要求
- Python 3.6+
- WordflowLab AgentSDK

## 使用方法

### 在AgentSDK中使用
技能已配置为在以下关键词触发时自动激活：
- "pdf", "pdf 处理", "pdf 提取", "pdf 合并", "pdf 分割"
- "pdf 表单", "提取 pdf", "pdf 转换", "pdf 文本", "pdf 表格"
- "填写表单", "处理 pdf", "分析 pdf"

### 直接使用脚本
```bash
# 检查依赖
python skills/pdf/scripts/check_dependencies.py

# 检查PDF表单字段
python skills/pdf/scripts/check_fillable_fields.py document.pdf

# 提取表单信息
python skills/pdf/scripts/extract_form_field_info.py document.pdf

# PDF转图像
python skills/pdf/scripts/convert_pdf_to_images.py document.pdf output_dir/
```

## 文件结构

```
pdf/
├── SKILL.md                    # 技能定义文件
├── README.md                   # 本说明文件
├── reference.md                # 高级功能参考文档
├── forms.md                    # 表单处理专门指南
├── LICENSE.txt                 # 许可证文件
└── scripts/                    # Python脚本目录
    ├── check_bounding_boxes.py
    ├── check_bounding_boxes_test.py
    ├── check_fillable_fields.py
    ├── convert_pdf_to_images.py
    ├── create_validation_image.py
    ├── extract_form_field_info.py
    ├── fill_fillable_fields.py
    ├── fill_pdf_form_with_annotations.py
    └── check_dependencies.py
```

## 配置

技能已在 `main.go` 中启用：
```go
EnabledSkills: []string{
    "consistency-checker",
    "pdfmd",
    "pdf",  // ← 新添加的PDF技能
},
```

## 测试

1. 运行依赖检查：
   ```bash
   python skills/pdf/scripts/check_dependencies.py
   ```

2. 创建测试PDF（需要reportlab）：
   ```bash
   python test_simple_pdf.py
   ```

3. 测试脚本功能：
   ```bash
   python skills/pdf/scripts/check_fillable_fields.py test_document.pdf
   ```

## 许可证

本技能采用专有许可证，详见 LICENSE.txt 文件。

## 支持的功能

### 文本和表格处理
- PDF文本提取（支持中文）
- 表格数据提取和格式化
- OCR处理扫描文档
- 元数据提取

### PDF操作
- 文档合并和分割
- 页面旋转和重排
- 密码保护和加密
- 水印添加

### 表单处理
- 可填写字段检测
- 表单数据提取
- 自动表单填写
- 表单验证

### 图像处理
- PDF转图像转换
- 边界框检查
- 图像质量验证
- 批量处理支持

## 故障排除

### 常见问题
1. **依赖缺失**: 运行 `check_dependencies.py` 检查并安装所需依赖
2. **PDF损坏**: 确保PDF文件完整且未加密
3. **权限问题**: 确保对PDF文件有读取权限
4. **中文乱码**: 检查PDF字体是否支持中文字符

### 错误处理
所有脚本都包含适当的错误处理，会提供详细的错误信息来帮助诊断问题。

## 扩展

可以通过以下方式扩展技能功能：
1. 添加新的Python脚本到 `scripts/` 目录
2. 更新 `SKILL.md` 中的功能描述
3. 添加新的触发关键词
4. 集成额外的PDF处理库

## 贡献

如需添加新功能或修复问题，请确保：
1. 保持向后兼容性
2. 添加适当的错误处理
3. 更新文档说明
4. 测试所有功能正常工作
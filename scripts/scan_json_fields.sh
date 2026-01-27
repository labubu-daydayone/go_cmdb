#!/bin/bash
# 扫描所有 Go 文件中的 JSON 标签，检测是否符合 lowerCamelCase 规范

set -e

PROJECT_ROOT="/home/ubuntu/go_cmdb"
REPORT_FILE="/tmp/json_fields_report.txt"

echo "========================================="
echo "JSON 字段命名规范扫描"
echo "========================================="
echo ""

# 清空报告文件
> "$REPORT_FILE"

# 定义需要检查的目录
DIRS=(
    "$PROJECT_ROOT/api/v1"
    "$PROJECT_ROOT/internal/model"
    "$PROJECT_ROOT/internal/dto"
)

# 定义不符合规范的模式
# 1. snake_case: 包含下划线的字段名
# 2. 大写开头的驼峰（PascalCase）：MainIP, SubIPs 等

echo "扫描中..."
echo ""

TOTAL_ISSUES=0

for DIR in "${DIRS[@]}"; do
    if [ ! -d "$DIR" ]; then
        continue
    fi
    
    echo "检查目录: $DIR"
    
    # 查找所有 .go 文件中的 json 标签
    while IFS= read -r file; do
        # 跳过备份文件
        if [[ "$file" == *.bak ]]; then
            continue
        fi
        
        # 查找 snake_case 字段（包含下划线）
        SNAKE_CASE=$(grep -n 'json:"[^"]*_[^"]*"' "$file" || true)
        if [ -n "$SNAKE_CASE" ]; then
            echo "  [SNAKE_CASE] $file" >> "$REPORT_FILE"
            echo "$SNAKE_CASE" | while read -r line; do
                echo "    $line" >> "$REPORT_FILE"
                ((TOTAL_ISSUES++)) || true
            done
            echo "" >> "$REPORT_FILE"
        fi
        
    done < <(find "$DIR" -name "*.go" -type f)
done

echo ""
echo "========================================="
echo "扫描完成"
echo "========================================="
echo ""

if [ $TOTAL_ISSUES -eq 0 ]; then
    echo "✓ 未发现不符合规范的字段"
else
    echo "✗ 发现 $TOTAL_ISSUES 个不符合规范的字段"
    echo ""
    echo "详细报告已保存到: $REPORT_FILE"
    echo ""
    cat "$REPORT_FILE"
fi

exit 0

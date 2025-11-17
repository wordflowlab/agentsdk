#!/bin/bash
# 停止后端服务器

if [ -f /tmp/agentsdk.pid ]; then
    PID=$(cat /tmp/agentsdk.pid)
    echo "停止服务器 (PID: $PID)..."
    kill $PID 2>/dev/null || true
    rm /tmp/agentsdk.pid
    echo "✅ 服务器已停止"
else
    echo "⚠️  未找到运行的服务器 PID 文件"
    # 尝试通过端口查找并停止
    if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1 ; then
        echo "通过端口查找并停止..."
        kill $(lsof -t -i:8080) 2>/dev/null || true
        echo "✅ 服务器已停止"
    else
        echo "没有运行在 8080 端口的服务器"
    fi
fi

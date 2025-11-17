#!/bin/bash
# 启动后端服务器用于测试

cd /Users/coso/Documents/dev/ai/wordflowlab/agentsdk

echo "🚀 启动 AgentSDK 后端服务器..."

# 检查端口占用
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1 ; then
    echo "⚠️  端口 8080 已被占用"
    echo "停止现有进程..."
    kill $(lsof -t -i:8080) 2>/dev/null || true
    sleep 2
fi

# 启动服务器
echo "启动服务器..."
go run cmd/agentsdk/main.go serve &
SERVER_PID=$!
echo $SERVER_PID > /tmp/agentsdk.pid

# 等待服务器启动
echo "等待服务器就绪..."
for i in {1..30}; do
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo "✅ 服务器已启动 (PID: $SERVER_PID)"
        echo "访问地址: http://localhost:8080"
        exit 0
    fi
    echo -n "."
    sleep 1
done

echo ""
echo "❌ 服务器启动超时"
kill $SERVER_PID 2>/dev/null || true
exit 1

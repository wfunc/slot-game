#!/bin/bash

# WebSocket 连接测试脚本

echo "测试 WebSocket 连接..."

# 使用 curl 测试 WebSocket 握手
echo "1. 测试 WebSocket 握手:"
curl -i -N \
    -H "Connection: Upgrade" \
    -H "Upgrade: websocket" \
    -H "Sec-WebSocket-Version: 13" \
    -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
    http://localhost:8080/ws/game

echo ""
echo "2. 测试在线人数接口:"
curl -s http://localhost:8080/ws/online | jq .

echo ""
echo "测试完成！"
echo ""
echo "要进行完整的 WebSocket 测试，请在浏览器中打开："
echo "http://localhost:8080/static/websocket-test.html"
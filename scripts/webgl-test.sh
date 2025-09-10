#!/bin/bash

# WebGL 性能测试和诊断脚本
# 专门针对 Cocos Web Mobile 游戏优化

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "================================================"
echo -e "${GREEN}WebGL/Cocos 性能优化测试${NC}"
echo "================================================"
echo ""

# 检测 Chromium
CHROMIUM_PATH=""
if command -v chromium >/dev/null 2>&1; then
    CHROMIUM_PATH=$(which chromium)
elif command -v chromium-browser >/dev/null 2>&1; then
    CHROMIUM_PATH=$(which chromium-browser)
else
    echo -e "${RED}未找到 Chromium${NC}"
    exit 1
fi

echo -e "${GREEN}Chromium 路径: $CHROMIUM_PATH${NC}"

# 检测 GPU 信息
echo -e "\n${BLUE}系统 GPU 信息：${NC}"
echo "----------------------------------------"

# 检查 GPU 驱动
if command -v lspci >/dev/null 2>&1; then
    echo "PCI 设备："
    lspci | grep -i vga || echo "未找到 VGA 设备"
    lspci | grep -i 3d || echo "未找到 3D 控制器"
fi

# 检查 OpenGL/EGL 支持
echo -e "\n${BLUE}OpenGL/EGL 支持：${NC}"
if command -v glxinfo >/dev/null 2>&1; then
    echo "OpenGL 版本："
    glxinfo | grep "OpenGL version" || echo "无法获取 OpenGL 信息"
    echo "OpenGL 渲染器："
    glxinfo | grep "OpenGL renderer" || echo "无法获取渲染器信息"
else
    echo "glxinfo 未安装，尝试安装: sudo apt install mesa-utils"
fi

# 检查 EGL
if [ -f /usr/lib/aarch64-linux-gnu/libEGL.so ] || [ -f /usr/lib/arm-linux-gnueabihf/libEGL.so ]; then
    echo -e "${GREEN}✓ EGL 库已安装${NC}"
else
    echo -e "${YELLOW}⚠ EGL 库可能未安装${NC}"
fi

# 检查 GLES
if [ -f /usr/lib/aarch64-linux-gnu/libGLESv2.so ] || [ -f /usr/lib/arm-linux-gnueabihf/libGLESv2.so ]; then
    echo -e "${GREEN}✓ GLES 库已安装${NC}"
else
    echo -e "${YELLOW}⚠ GLES 库可能未安装${NC}"
fi

echo -e "\n${BLUE}WebGL 测试配置：${NC}"
echo "----------------------------------------"

# 创建 WebGL 测试 HTML
cat > /tmp/webgl-test.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>WebGL Test</title>
    <style>
        body { font-family: Arial; padding: 20px; background: #222; color: #fff; }
        canvas { border: 2px solid #0f0; }
        .info { margin: 20px 0; padding: 10px; background: #333; }
        .success { color: #0f0; }
        .error { color: #f00; }
    </style>
</head>
<body>
    <h1>WebGL Performance Test</h1>
    <div id="status" class="info"></div>
    <canvas id="canvas" width="800" height="600"></canvas>
    <div id="info" class="info"></div>
    
    <script>
        const status = document.getElementById('status');
        const info = document.getElementById('info');
        const canvas = document.getElementById('canvas');
        
        // 测试 WebGL 1
        let gl = canvas.getContext('webgl');
        if (gl) {
            status.innerHTML += '<p class="success">✓ WebGL 1.0 支持</p>';
            
            // 获取 WebGL 信息
            const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
            if (debugInfo) {
                const vendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
                const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                info.innerHTML += '<p>GPU 厂商: ' + vendor + '</p>';
                info.innerHTML += '<p>GPU 渲染器: ' + renderer + '</p>';
            }
            
            info.innerHTML += '<p>WebGL 版本: ' + gl.getParameter(gl.VERSION) + '</p>';
            info.innerHTML += '<p>GLSL 版本: ' + gl.getParameter(gl.SHADING_LANGUAGE_VERSION) + '</p>';
            info.innerHTML += '<p>最大纹理大小: ' + gl.getParameter(gl.MAX_TEXTURE_SIZE) + '</p>';
            info.innerHTML += '<p>最大顶点属性: ' + gl.getParameter(gl.MAX_VERTEX_ATTRIBS) + '</p>';
        } else {
            status.innerHTML += '<p class="error">✗ WebGL 1.0 不支持</p>';
        }
        
        // 测试 WebGL 2
        let gl2 = canvas.getContext('webgl2');
        if (gl2) {
            status.innerHTML += '<p class="success">✓ WebGL 2.0 支持</p>';
        } else {
            status.innerHTML += '<p class="error">✗ WebGL 2.0 不支持</p>';
        }
        
        // 性能测试
        if (gl) {
            let frameCount = 0;
            let lastTime = performance.now();
            
            function animate() {
                frameCount++;
                const currentTime = performance.now();
                
                if (currentTime - lastTime >= 1000) {
                    document.title = 'FPS: ' + frameCount;
                    frameCount = 0;
                    lastTime = currentTime;
                }
                
                // 简单的渲染
                gl.clearColor(Math.random(), Math.random(), Math.random(), 1.0);
                gl.clear(gl.COLOR_BUFFER_BIT);
                
                requestAnimationFrame(animate);
            }
            animate();
        }
    </script>
</body>
</html>
EOF

echo -e "${GREEN}创建了 WebGL 测试页面：/tmp/webgl-test.html${NC}"

echo -e "\n${BLUE}推荐的 Chromium 启动配置：${NC}"
echo "----------------------------------------"

echo -e "${YELLOW}配置 1: 标准 EGL + GLES（推荐 ARM64）${NC}"
cat << EOF
$CHROMIUM_PATH \\
    --kiosk \\
    --enable-webgl \\
    --enable-webgl2 \\
    --use-gl=egl \\
    --use-angle=gles \\
    --ignore-gpu-blocklist \\
    --enable-gpu-rasterization \\
    --enable-accelerated-2d-canvas \\
    --enable-zero-copy \\
    --no-sandbox \\
    --disable-dev-shm-usage \\
    file:///tmp/webgl-test.html
EOF

echo -e "\n${YELLOW}配置 2: 强制 GPU 加速（性能最大化）${NC}"
cat << EOF
$CHROMIUM_PATH \\
    --kiosk \\
    --enable-webgl \\
    --enable-webgl2 \\
    --use-gl=egl \\
    --enable-unsafe-webgpu \\
    --enable-features=VaapiVideoDecoder,WebGPU \\
    --ignore-gpu-blocklist \\
    --enable-gpu-rasterization \\
    --enable-accelerated-2d-canvas \\
    --enable-native-gpu-memory-buffers \\
    --enable-gpu-memory-buffer-video-frames \\
    --enable-zero-copy \\
    --force-gpu-mem-available-mb=512 \\
    --gpu-rasterization-msaa-sample-count=0 \\
    --no-sandbox \\
    file:///tmp/webgl-test.html
EOF

echo -e "\n${YELLOW}配置 3: 兼容模式（如果上述失败）${NC}"
cat << EOF
$CHROMIUM_PATH \\
    --kiosk \\
    --enable-webgl \\
    --use-gl=swiftshader \\
    --no-sandbox \\
    --disable-dev-shm-usage \\
    file:///tmp/webgl-test.html
EOF

echo -e "\n${BLUE}性能优化建议：${NC}"
echo "----------------------------------------"
echo "1. 确保安装 GPU 驱动："
echo "   sudo apt install mesa-utils mesa-utils-extra"
echo "   sudo apt install libgles2-mesa libgles2-mesa-dev"
echo "   sudo apt install libegl1-mesa libegl1-mesa-dev"
echo ""
echo "2. 对于 RK3566/RK3568 芯片："
echo "   sudo apt install rockchip-mali-midgard-dev"
echo "   或"
echo "   sudo apt install mali-g52-driver"
echo ""
echo "3. 检查 chrome://gpu 页面："
echo "   在浏览器中打开 chrome://gpu 查看 GPU 加速状态"
echo ""
echo "4. Cocos 游戏优化："
echo "   - 确保游戏使用 WebGL 1.0（更好的兼容性）"
echo "   - 减少 Draw Call"
echo "   - 使用纹理图集"
echo "   - 限制粒子效果数量"
echo ""
echo "5. 监控性能："
echo "   在游戏运行时按 Shift+Ctrl+I 打开开发者工具"
echo "   查看 Performance 标签页的 FPS"

echo -e "\n${GREEN}测试步骤：${NC}"
echo "----------------------------------------"
echo "1. 先测试 WebGL 支持："
echo "   $CHROMIUM_PATH --no-sandbox file:///tmp/webgl-test.html"
echo ""
echo "2. 如果 WebGL 正常，测试您的游戏："
echo "   $CHROMIUM_PATH --kiosk --enable-webgl --use-gl=egl --no-sandbox http://127.0.0.1:8080"
echo ""
echo "3. 更新服务文件："
echo "   sudo nano /etc/systemd/system/chromium-kiosk.service"
echo "   修改 ExecStart 使用上述成功的配置"
echo "   sudo systemctl daemon-reload"
echo "   sudo systemctl restart chromium-kiosk"
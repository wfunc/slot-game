const { chromium } = require('playwright');

// Protobuf 编码辅助函数
function encodeProtobuf(msgId, data = {}) {
    // 简单的Protobuf编码实现
    const encoder = new TextEncoder();
    const jsonStr = JSON.stringify(data);
    const jsonBytes = encoder.encode(jsonStr);

    // 构造消息: [长度(4字节)] + [消息ID(2字节)] + [数据]
    const totalLength = 2 + jsonBytes.length;
    const buffer = new ArrayBuffer(6 + jsonBytes.length);
    const view = new DataView(buffer);

    // 长度 (大端序)
    view.setUint32(0, totalLength, false);
    // 消息ID (大端序)
    view.setUint16(4, msgId, false);
    // 数据
    new Uint8Array(buffer, 6).set(jsonBytes);

    return buffer;
}

(async () => {
    const browser = await chromium.launch({ headless: false });
    const page = await browser.newPage();

    // 监听控制台消息
    page.on('console', msg => {
        console.log(`浏览器控制台: ${msg.type()}: ${msg.text()}`);
    });

    // 访问统一测试页面
    await page.goto('http://localhost:8080/static/unified-game-test.html');

    console.log('页面加载完成，开始测试WebSocket连接...');

    // 测试WebSocket连接
    const wsConnected = await page.evaluate(async () => {
        return new Promise((resolve) => {
            const ws = new WebSocket('ws://localhost:8080/ws/game');

            ws.onopen = () => {
                console.log('WebSocket连接成功');
                resolve(true);
            };

            ws.onerror = (error) => {
                console.error('WebSocket连接失败:', error);
                resolve(false);
            };

            ws.onmessage = (event) => {
                console.log('收到消息:', event.data);
            };

            setTimeout(() => {
                ws.close();
                resolve(false);
            }, 5000);
        });
    });

    if (!wsConnected) {
        console.error('WebSocket连接失败');
        await browser.close();
        return;
    }

    console.log('WebSocket连接成功，测试消息发送...');

    // 测试发送心跳消息（使用正确的消息ID 2099）
    await page.evaluate(() => {
        const ws = new WebSocket('ws://localhost:8080/ws/game');

        ws.onopen = () => {
            console.log('发送心跳消息 (2099)...');
            // 需要发送Protobuf编码的消息
            const msgId = 2099;
            const data = new ArrayBuffer(6);
            const view = new DataView(data);
            view.setUint32(0, 2, false); // 长度
            view.setUint16(4, msgId, false); // 消息ID
            ws.send(data);
        };

        ws.onmessage = (event) => {
            console.log('收到响应:', event.data);
            if (event.data instanceof Blob) {
                event.data.arrayBuffer().then(buffer => {
                    const view = new DataView(buffer);
                    const length = view.getUint32(0, false);
                    const msgId = view.getUint16(4, false);
                    console.log(`解析消息: 长度=${length}, 消息ID=${msgId}`);
                });
            }
        };
    });

    // 等待响应
    await page.waitForTimeout(3000);

    // 测试获取配置消息 (2001)
    console.log('测试获取配置消息...');
    await page.evaluate(() => {
        const ws = new WebSocket('ws://localhost:8080/ws/game');

        ws.onopen = () => {
            console.log('发送获取配置消息 (2001)...');
            const msgId = 2001;
            const data = new ArrayBuffer(6);
            const view = new DataView(data);
            view.setUint32(0, 2, false);
            view.setUint16(4, msgId, false);
            ws.send(data);
        };

        ws.onmessage = (event) => {
            console.log('收到配置响应:', event.data);
        };
    });

    await page.waitForTimeout(3000);

    // 测试Animal游戏连接
    console.log('测试Animal游戏连接...');
    await page.evaluate(() => {
        const ws = new WebSocket('ws://localhost:8080/ws/game?game=animal');

        ws.onopen = () => {
            console.log('Animal游戏连接成功');
            // 发送进入房间消息 (1801)
            const msgId = 1801;
            const data = new ArrayBuffer(6);
            const view = new DataView(data);
            view.setUint32(0, 2, false);
            view.setUint16(4, msgId, false);
            ws.send(data);
        };

        ws.onmessage = (event) => {
            console.log('Animal游戏响应:', event.data);
        };
    });

    await page.waitForTimeout(3000);

    console.log('测试完成');
    await browser.close();
})();
const { chromium } = require('playwright');

async function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

(async () => {
    const browser = await chromium.launch({ headless: false });
    const page = await browser.newPage();

    console.log('=== 测试修复后的后端服务 ===\n');

    // 监听控制台消息
    const logs = [];
    page.on('console', msg => {
        const text = msg.text();
        logs.push(text);
        console.log(`[浏览器] ${text}`);
    });

    // 访问测试页面
    await page.goto('http://localhost:8080/static/unified-game-test.html');
    await sleep(1000);

    console.log('\n🔧 测试1: Animal游戏连接和消息处理...');

    // 选择Animal游戏
    await page.selectOption('#gameType', 'animal');
    await page.click('#connectBtn');
    await sleep(2000);

    // 检查连接状态
    const animalStatus = await page.textContent('#status');
    console.log(`   连接状态: ${animalStatus}`);

    // 测试配置请求 (2001)
    console.log('   测试配置请求 (2001)...');
    await page.click('button:has-text("获取配置")');
    await sleep(1000);

    // 测试心跳 (2099)
    console.log('   测试心跳 (2099)...');
    await page.click('button:has-text("发送心跳")');
    await sleep(1000);

    // 断开Animal连接
    await page.click('#disconnectBtn');
    await sleep(1000);

    console.log('\n🎰 测试2: Slot游戏连接和消息处理...');

    // 选择Slot游戏
    await page.selectOption('#gameType', 'slot');
    await page.click('#connectBtn');
    await sleep(2000);

    // 检查连接状态
    const slotStatus = await page.textContent('#status');
    console.log(`   连接状态: ${slotStatus}`);

    // 测试进入房间 (1901)
    console.log('   测试进入房间 (1901)...');
    await page.fill('#customMsgId', '1901');
    await page.fill('#customData', '{}');
    await page.click('button:has-text("发送自定义消息")');
    await sleep(1000);

    // 测试开始游戏 (1902)
    console.log('   测试开始游戏 (1902)...');
    await page.fill('#customMsgId', '1902');
    await page.fill('#customData', '{"betVal": 10}');
    await page.click('button:has-text("发送自定义消息")');
    await sleep(1000);

    // 测试配置请求 (2001)
    console.log('   测试配置请求 (2001)...');
    await page.click('button:has-text("获取配置")');
    await sleep(1000);

    // 测试心跳 (2099)
    console.log('   测试心跳 (2099)...');
    await page.click('button:has-text("发送心跳")');
    await sleep(1000);

    // 测试之前报错的消息ID
    console.log('   测试1904消息 (应该有警告但不报错)...');
    await page.fill('#customMsgId', '1904');
    await page.fill('#customData', '{}');
    await page.click('button:has-text("发送自定义消息")');
    await sleep(1000);

    console.log('   测试1905消息 (应该有警告但不报错)...');
    await page.fill('#customMsgId', '1905');
    await page.fill('#customData', '{}');
    await page.click('button:has-text("发送自定义消息")');
    await sleep(1000);

    console.log('\n=== 测试结果汇总 ===');

    // 分析日志
    const errorLogs = logs.filter(log =>
        log.includes('错误') ||
        log.includes('error') ||
        log.includes('失败') ||
        log.includes('proto: cannot parse')
    );

    const successLogs = logs.filter(log =>
        log.includes('收到消息') ||
        log.includes('响应') ||
        log.includes('连接成功')
    );

    console.log(`✅ 成功消息: ${successLogs.length} 条`);
    console.log(`${errorLogs.length === 0 ? '✅' : '❌'} 错误消息: ${errorLogs.length} 条`);

    if (errorLogs.length > 0) {
        console.log('\n错误详情:');
        errorLogs.slice(0, 5).forEach(err => console.log(`  - ${err}`));
        if (errorLogs.length > 5) {
            console.log(`  ... 还有 ${errorLogs.length - 5} 条错误`);
        }
    }

    // 检查特定修复
    const uniqueErrors = logs.filter(log => log.includes('UNIQUE constraint'));
    const parseErrors = logs.filter(log => log.includes('proto: cannot parse'));
    const unknownMsgErrors = logs.filter(log => log.includes('未知消息ID'));

    console.log('\n📊 具体修复验证:');
    console.log(`${uniqueErrors.length === 0 ? '✅' : '❌'} UNIQUE约束错误: ${uniqueErrors.length} 条`);
    console.log(`${parseErrors.length === 0 ? '✅' : '❌'} Protobuf解析错误: ${parseErrors.length} 条`);
    console.log(`${unknownMsgErrors.length === 0 ? '✅' : '❌'} 未知消息ID错误: ${unknownMsgErrors.length} 条`);

    console.log('\n=== 修复验证完成 ===');

    await sleep(3000);
    await browser.close();
})();
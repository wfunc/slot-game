const { chromium } = require('playwright');

async function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

(async () => {
    const browser = await chromium.launch({ headless: false });
    const page = await browser.newPage();

    console.log('=== 开始测试修复后的HTML ===\n');

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

    console.log('\n1️⃣ 测试Slot游戏连接...');

    // 选择Slot游戏
    await page.selectOption('#gameType', 'slot');

    // 点击连接
    await page.click('#connectBtn');
    await sleep(2000);

    // 检查连接状态
    const slotStatus = await page.textContent('#status');
    console.log(`   连接状态: ${slotStatus}`);

    // 检查是否收到配置
    const playerId = await page.textContent('#playerId');
    console.log(`   玩家ID: ${playerId}`);

    // 手动发送心跳
    console.log('   发送心跳测试...');
    await page.click('button:has-text("发送心跳")');
    await sleep(1000);

    // 断开连接
    await page.click('#disconnectBtn');
    await sleep(1000);

    console.log('\n2️⃣ 测试Animal游戏连接...');

    // 选择Animal游戏
    await page.selectOption('#gameType', 'animal');

    // 点击连接
    await page.click('#connectBtn');
    await sleep(2000);

    // 检查连接状态
    const animalStatus = await page.textContent('#status');
    console.log(`   连接状态: ${animalStatus}`);

    // 检查彩金显示
    const jackpotVisible = await page.isVisible('#jackpotPanel');
    console.log(`   彩金面板显示: ${jackpotVisible}`);

    if (jackpotVisible) {
        const jackpotAmount = await page.textContent('#jackpotAmount');
        console.log(`   当前彩金: ${jackpotAmount}`);
    }

    // 测试Animal游戏操作
    console.log('   发送进入房间消息...');
    await page.click('button:has-text("进入房间")');
    await sleep(1000);

    console.log('   发送彩金历史请求...');
    await page.click('button:has-text("彩金历史")');
    await sleep(1000);

    // 检查心跳计数
    const heartbeatCount = await page.textContent('#heartbeatCount');
    console.log(`   心跳计数: ${heartbeatCount}`);

    console.log('\n3️⃣ 测试自定义消息发送...');

    // 发送自定义消息
    await page.fill('#customMsgId', '2099');
    await page.fill('#customData', '{}');
    await page.click('button:has-text("发送自定义消息")');
    await sleep(1000);

    console.log('\n=== 测试结果汇总 ===');

    // 分析日志
    const heartbeatLogs = logs.filter(log => log.includes('心跳'));
    const configLogs = logs.filter(log => log.includes('配置'));
    const errorLogs = logs.filter(log => log.includes('错误') || log.includes('error'));
    const jackpotLogs = logs.filter(log => log.includes('彩金'));

    console.log(`✅ 心跳消息: ${heartbeatLogs.length} 条`);
    console.log(`✅ 配置消息: ${configLogs.length} 条`);
    console.log(`✅ 彩金消息: ${jackpotLogs.length} 条`);
    console.log(`${errorLogs.length > 0 ? '❌' : '✅'} 错误消息: ${errorLogs.length} 条`);

    if (errorLogs.length > 0) {
        console.log('\n错误详情:');
        errorLogs.forEach(err => console.log(`  - ${err}`));
    }

    console.log('\n=== 测试完成 ===');

    // 保持页面开启5秒以观察
    await sleep(5000);

    await browser.close();
})();
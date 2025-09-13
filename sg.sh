#!/bin/bash

make build-arm64
scp release/slot-game-arm64/slot-game sg@192.168.10.113:~/slot-game/
echo "可以手动测试了"
scp release/slot-game-arm64.tar.gz ztl@192.168.10.113:~/

echo "可以一键安装了"

#!/bin/bash

make build-arm64
scp release/slot-game-arm64/slot-game sg@192.168.10.113:~/slot-game/

# Makefile for serial tester
BUILD_DIR=bin
CMD_DIR=cmd

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# Build flags
LDFLAGS=-ldflags "-s -w"

# All executables
EXECUTABLES=serial_tester protocol_tester serial_diagnose hardware_diagnose android_simulator stm32_simulator stm32_protocol serial_test_tool

all: build

build:
	@echo "Building for current platform..."
	@mkdir -p $(BUILD_DIR)
	@for exe in $(EXECUTABLES); do \
		echo "Building $$exe..."; \
		$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$$exe ./$(CMD_DIR)/$$exe; \
	done

linux:
	@echo "Building for Linux amd64..."
	@mkdir -p $(BUILD_DIR)
	@for exe in $(EXECUTABLES); do \
		echo "Building $$exe for Linux amd64..."; \
		GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$${exe}_linux_amd64 ./$(CMD_DIR)/$$exe; \
	done

linux-arm:
	@echo "Building for Linux ARM..."
	@mkdir -p $(BUILD_DIR)
	@for exe in $(EXECUTABLES); do \
		echo "Building $$exe for Linux ARM..."; \
		GOOS=linux GOARCH=arm $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$${exe}_linux_arm ./$(CMD_DIR)/$$exe; \
	done

linux-arm64:
	@echo "Building for Linux ARM64..."
	@mkdir -p $(BUILD_DIR)
	@for exe in $(EXECUTABLES); do \
		echo "Building $$exe for Linux ARM64..."; \
		GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$${exe}_linux_arm64 ./$(CMD_DIR)/$$exe; \
	done

all-linux: linux linux-arm linux-arm64
	@echo "All Linux builds complete!"
	@ls -lh $(BUILD_DIR)/*linux*

test:
	@echo "Running tests..."
	cd ../.. && $(GOTEST) -v ./examples/testserial/tests/...

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)/*
	rm -f *.exe

deps:
	@echo "Downloading dependencies..."
	$(GOGET) github.com/tarm/serial

.PHONY: all build linux linux-arm linux-arm64 all-linux test clean deps
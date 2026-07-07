APP_NAME    := LQBOT
CMD_PATH    := ./cmd/bot
GIT_COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_DATE    := $(shell git log -1 --format=%cd --date=format:'%Y%m%d' 2>/dev/null || echo "unknown")
VERSION     := $(shell grep 'Version\s*=' internal/version/version.go | head -1 | sed 's/.*"\(.*\)".*/\1/')
VERSION_PKG := github.com/Luoyangan/LQBOT/internal/version
LDFLAGS     := -s -w \
	-X '$(VERSION_PKG).Commit=$(GIT_COMMIT)' \
	-X '$(VERSION_PKG).Date=$(GIT_DATE)'

.PHONY: build build-windows build-linux build-all clean

run: configs/config.yaml  ## 开发运行（自动注入 git 信息）
	go run -ldflags="$(LDFLAGS)" $(CMD_PATH) -c configs/config.yaml

build:  ## 构建当前系统版本
	go build -ldflags="$(LDFLAGS)" -o $(APP_NAME)-v$(VERSION)-$(GIT_COMMIT) $(CMD_PATH)

build-windows:  ## 构建 Windows amd64
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(APP_NAME)-v$(VERSION)-$(GIT_COMMIT).exe $(CMD_PATH)

build-linux:  ## 构建 Linux amd64
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(APP_NAME)-v$(VERSION)-$(GIT_COMMIT)-linux $(CMD_PATH)

build-all: build-windows build-linux  ## 构建 Windows + Linux 双平台

clean:  ## 清理构建产物
	rm -f $(APP_NAME)-v* $(APP_NAME)-v*.exe

UI_DIST := frontend/dist
EMBED_DIST := backend/static/dist
BIN := bin

.PHONY: all ui server gui build run-server run-gui

all: build

# 构建前端并复制到后端 embed 目录
ui:
	cd frontend && npm install && npm run build
	mkdir -p $(EMBED_DIST)
	cp -R $(UI_DIST)/. $(EMBED_DIST)/

# server 模式:命令行 HTTP 服务(浏览器访问)
server: ui
	cd backend && go build -o ../$(BIN)/warehouse-server ./cmd/server

# gui 模式:内置服务 + 桌面 webview
gui: ui
	cd backend && go build -o ../$(BIN)/warehouse-gui ./cmd/gui

build: server gui

run-server: server
	$(BIN)/warehouse-server

run-gui: gui
	$(BIN)/warehouse-gui

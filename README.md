# 仓库台账

单人单仓进销存,所有写操作(商品增改、入库、出库、盘点)自动写入审计日志,日志只读不可篡改。

## 功能

- 库存总览:SKU 数、库存合计、今日出入库
- 商品管理:名称、规格、单位、分类
- 批次与价格:每次入库记录单价、货位、供应商,库存页显示参考均价与货位分布
- 出入库与盘点:入库生成批次,出库可选货位或自动先进先出(FIFO)扣减批次,盘点可选按货位或整商品调整,均校验库存
- 出库批次预览:出库时实时显示将按先进先出扣减的批次明细(自动货位或指定货位均可),流水记录关联批次号
- 货位余量:每个货位实时显示剩余数量,库存页货位列带各货位余量,随出入库/盘点自动增减
- 批量出入库:Excel 模板导入(浏览器端解析),整批事务提交;库存/批次/流水一键导出 Excel
- 批次视图:每个批次的入库量、剩余量、单价、货位、供应商,全程可追溯
- 流水记录:类型、数量变化、库存前后、对方/去向、备注
- 操作日志:每笔写操作留痕,含操作人、时间、动作与前后明细

## 启动

```bash
# 后端(默认端口 8080)
cd backend
WAREHOUSE_OPERATOR=你的名字 go run .

# 前端(默认端口 5173)
cd frontend
npm install
BACKEND_URL=http://127.0.0.1:8080 npm run dev
```

`WAREHOUSE_OPERATOR` 是留痕里的操作人,不设置默认 `admin`。
后端数据默认存在 `backend/warehouse.db`(SQLite),可用 `-db` 指定路径。

## 开发

- 后端:纯 Go 标准库 + `modernc.org/sqlite`,入口 `backend/main.go`;MVC 分层参照 `../disapp/backend`:
  - `pkg/web`:HTTP 中间件(Recoverer / Logger / RateLimit)与统一 `{code,msg,data}` 响应
  - `internal/router`:路由与中间件链
  - `internal/controller`:HTTP 层,解析参数并调用 Service
  - `internal/service`:业务层与 `service.Error` 错误映射
  - `internal/store`:SQLite 数据层
- 前端:Vue 3 + Vite,`frontend/`;代理 `/api` 到后端,可用 `BACKEND_URL` 覆盖

审计日志表 `audit_log` 与业务数据在同一 SQLite 事务中写入,保证操作与留痕一致。不提供删除/修改审计数据的能力。

基于 Go 实现的鸟类迁徙环志重捕证据复核 Web 项目，一款后端服务，完成环志重捕事件的时空排序校验、最大飞行速度约束下的可行迁徙边构建与异常重捕裁决冻结。

# BENZHI 评测说明

本项目为纯后端 Go 服务，对外暴露 `/api` 前缀的 HTTP 接口，使用 SQLite 持久化，
支持进程关闭后重新打开同一数据库恢复全部业务数据。

## 标准命令

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/birdbanding --smoke-test
go run ./cmd/birdbanding --addr :8080 --db birdbanding.db
```

- `--addr`：HTTP 监听地址，默认 `:8080`
- `--db`：SQLite 数据库文件路径，默认 `birdbanding.db`
- `--smoke-test`：不常驻；跑完端到端场景后关闭并重新打开数据库，退出码 0 表示通过

## 冒烟自测契约（--smoke-test）

创建临时数据库 → 导入环志/重捕事件 → 校验身份与时间单调性 → 构建可行迁徙边 →
异常裁决 → 冻结路径版本 → 关闭并重新打开数据库，校验批次/事件/边/版本全部持久化恢复后退出 0。

## Docker 构建与双架构验证

`Dockerfile` 与 `benzhi.Dockerfile` 内容完全一致。使用项目提供的
`build_benzhi_docker.sh` 构建评测镜像；Dockerfile 不声明端口，服务监听地址由
运行参数 `--addr` 指定。

```bash
./build_benzhi_docker.sh task253-birdbanding:amd64 linux/amd64
docker run --rm task253-birdbanding:amd64 --smoke-test

./build_benzhi_docker.sh task253-birdbanding:arm64 linux/arm64
docker run --rm task253-birdbanding:arm64 --smoke-test

docker run --rm -P task253-birdbanding:amd64 --addr :8080 --db ./app.db
```

## 核心 API（`/api` 前缀）

- 批次：录入/复核/发布/封存
- 事件：导入/校验/校正环号/排除
- 个体：关联/时间线/建边
- 迁徙边：确认/裁决
- 路径版本：创建/共享/冻结/替代
- 统计与自检：`GET /api/stats`、`GET /api/health`、`GET /api/selfcheck`

## 业务不变量

- 持久化：SQLite（modernc.org/sqlite，CGO 无关），重启同一数据库可恢复未发布版本与已冻结快照。
- 事件指纹幂等：相同指纹的重复导入被去重，不重复落库。
- 冻结版本不可变：对 `冻结` 状态的路径版本修改会被拒绝。
- 拒绝规则：环号格式错误、重捕时间倒退、地点精度缺失、冻结版本被修改，均以明确错误返回。

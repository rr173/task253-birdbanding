# birdbanding — 鸟类迁徙环志重捕证据复核台

面向鸟类学研究者的野外科学证据复核后端服务。导入环志与重捕事件及位置精度，按个体重建事件序列、检查季节与最大飞行速度约束，生成可行迁徙边与异常候选；研究者可校正环号、保留罕见路线并发布不可变路径版本。

## 业务闭环

1. 导入环志/重捕事件（含位置精度与季节）。
2. 校验个体身份与重捕时间单调性，排除格式错误/时间倒退/精度缺失。
3. 按个体排序事件，在最大飞行速度约束下构建可行迁徙边，标记超限与逆季节罕见。
4. 研究者对异常重捕裁决（保留罕见证据 / 排除），组装路径版本。
5. 冻结路径版本为不可变快照，供后续复核与替代。

## 状态机

- 观测批次：`录入中` → `待复核` → `已发布` → `封存`
- 环志/重捕事件：`待校验` → `有效` / `身份冲突` / `排除`
- 迁徙边：`候选` → `可行` / `超限` / `罕见` → `确认`
- 路径版本：`草稿` → `共享` → `冻结` → `替代`

## 标准命令

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/birdbanding --smoke-test
go run ./cmd/birdbanding --addr :8080 --db birdbanding.db
```

## 目录结构

仓库根目录就是源码根（本地 `env/` 原样推送，远程不再套一层 `env/`）：

```
cmd/birdbanding/           # 入口（--smoke-test 契约）
internal/model/            # 实体、错误、地理与状态机
internal/store/            # SQLite 持久化与迁移
internal/event/            # 事件导入与校验
internal/identity/         # 个体身份关联
internal/path/             # 可行迁徙边构建
internal/ruling/           # 异常裁决与版本冻结
internal/service/          # 编排层
internal/httpapi/          # /api HTTP 层
go.mod / go.sum
component-versions.json
Dockerfile / benzhi.Dockerfile / build_benzhi_docker.sh
BENZHI_README.md
```

详见 `BENZHI_README.md`。

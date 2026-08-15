# task011-cron

Cron 表达式解析与下次执行时间计算服务。

## 端点

- `GET /healthz` 健康检查
- `POST /api/cron/parse` `{"expr":"*/15 * * * *"}` 解析为各字段取值集合
- `POST /api/cron/next` `{"expr":"0 0 1 * *","from":"2026-08-14T12:00:00Z"}` 计算严格晚于基准时间的下次执行时间
- `POST /api/cron/validate` `{"expr":"0 0 1 * *"}` 合法性校验

## 运行

```bash
go run . server --addr :8080
go run . --smoke-test
```

## 构建

```bash
docker buildx build --platform linux/amd64 --load -t go-task-check:amd64 .
docker run --rm go-task-check:amd64 --smoke-test
```

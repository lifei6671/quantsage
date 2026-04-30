---
name: quantsage-go-datasource-conventions
description: Use when modifying QuantSage backend datasource code, browserfetch integrations, datasource tests, or related docs under apps/server, especially in internal/domain/datasource and internal/infra/browserfetch.
---

# quantsage-go-datasource-conventions

## 目标

统一 QuantSage 后端数据源相关改动的工程约束，减少目录越界、职责混乱和验证遗漏。

## 何时使用

满足任一情况时使用本 skill：

- 修改 `apps/server/internal/domain/datasource/**`
- 修改 `apps/server/internal/infra/browserfetch/**`
- 新增或调整数据源 parser、watcher、source、smoke test
- 调整数据源契约、抓取链路、回退逻辑、验证命令或相关文档

## 工作目录

- Go 代码、测试、lint、build：优先在 `apps/server` 下执行
- 跨应用联调：优先走根目录 `make`
- 不要在仓库根误跑 `go test ./...`

## 数据源改动约束

### 目录与职责

- 先看 `apps/server/internal/domain/datasource`
- 优先复用已有查询模型、错误码和 `browserfetch` 基础设施
- `parser` 只做解析和标准化，不混入浏览器动作
- `watcher` / `browserfetch` 负责页面动作、监听和抓取过程
- `List*` / `Stream*` 职责边界要清晰，不要把流式接口伪装成一次性接口

### Go 代码规范

- 公开函数/接口改动时同步补中文注释
- 错误统一用 `fmt.Errorf("操作描述: %w", err)` 包装
- 涉及 IO、DB、浏览器、外部调用的函数，第一个参数传 `context.Context`
- 对排障、日志和链路追踪有价值的核心入参：
  进入调用链后尽早通过 `infra/log.AddInfo(...)` 这类日志 KV 方法写入 `ctx`
- KV 字段只放稳定、可序列化、业务可读的关键信息，例如 `ts_code`、`index_code`、`market`、`job_name`
- 不要把这条规则理解成随意 `context.WithValue(...)` 塞业务对象；大对象、敏感信息或临时中间结果都不应放进去
- 不要忽略有意义的错误返回
- 需要有完善的中文注释

## browserfetch 约束

- 优先复用 `Run`、`RunWithActions`、`ObserveResponses`
- 站点如果对页面加载、子 tab、Cookie、懒加载敏感：
  先补 smoke test，再改实现
- 修改抓取行为时：
  优先补最小真实烟雾测试和对应单测

如果目标站点是百度财经或同类强前端耦合页面：

- **REQUIRED SUB-SKILL:** 使用 `baidu-browserfetch-pitfalls`

## 验证要求

后端改动至少执行与范围匹配的验证：

- `make test`
- 必要时加 `make race`
- 必要时加 `make build`
- 如果只在 `apps/server` 局部验证，也要至少跑对应 `go test -timeout 120s ...`
- 浏览器抓取或外站改动：补最小 smoke test

如果因为环境限制没跑某项验证，交付时必须明确写出：

- 没跑什么
- 为什么没跑

## 文档同步

以下变更通常要同步文档：

- 新增或修改数据源
- 修改后端 API、任务流、配置项、调度行为
- 修改浏览器抓取策略、验证方式或本地运行方式
- 新增真实 smoke test、联调入口或关键排障结论

优先检查：

- `README.md`
- `docs/architecture/*.md`
- `docs/superpowers/specs/*.md`
- `docs/superpowers/plans/*.md`

如果本次改动形成长期规则，再考虑回写 `AGENTS.md`。

## 禁止项

- 不要手动编辑 `go.sum`
- 不要为了单次任务随意新增第三方依赖，除非用户明确确认
- 不要在根目录自创和 `Makefile` 冲突的新验证约定
- 不要把临时日志、调试残留、取消掉的 `t.Skip(...)` 留在测试里

## 交付时最少说明

- 改了什么
- 为什么这么改
- 跑了哪些验证
- 哪些没验证
- 是否更新了文档；如果没更，理由是什么

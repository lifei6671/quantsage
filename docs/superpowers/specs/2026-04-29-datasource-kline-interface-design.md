# Datasource K-Line Interface Extension Design

## 1. 目标

扩展 `apps/server/internal/domain/datasource.Source`，让外部数据源在保留现有批量导入能力的前提下，支持“指定股票、多周期”的 K 线查询。

本次目标覆盖：

1. 指定 `TSCode` 查询日线、周线、月线、季线、年线与分钟线。
2. 对不支持该周期的数据源，直接返回明确错误。
3. 保持现有 `ListStocks`、`ListTradeCalendar`、`ListDailyBars` 语义不变，避免破坏导入任务。

本次设计不覆盖：

1. 在外部采集源接口上直接暴露复权查询参数。
2. 新的对外 HTTP API。
3. 将所有数据源强行补齐为完整分钟线能力。

## 2. 设计结论

采用“在 `Source` 上新增通用 K 线查询方法”的方案：

```go
type Source interface {
	ListStocks(ctx context.Context) ([]StockBasic, error)
	ListTradeCalendar(ctx context.Context, exchange string, startDate, endDate time.Time) ([]TradeDay, error)
	ListDailyBars(ctx context.Context, startDate, endDate time.Time) ([]DailyBar, error)
	ListKLines(ctx context.Context, query KLineQuery) ([]KLine, error)
	StreamKLines(ctx context.Context, query KLineQuery) (<-chan KLineStreamItem, error)
}
```

保留现有三类导入接口不变，新能力只负责单票查询，不与批量导入接口混用。

其中：

1. `ListKLines`
   用于一次性收口为完整 `[]KLine`。
2. `StreamKLines`
   用于支持未来页面驱动型或实时拉取型数据源持续回推 K 线批次。
3. 第一版里 `sample`、`tushare`、`eastmoney` 这三个已实现数据源都必须实现该方法，但如果自身不具备流式能力，则直接返回显式“不支持流式接口”错误，而不是静默降级。

## 2.1 页面驱动型数据源补充结论

对于新浪这类“打开股票详情页后，页面自身会持续请求后台接口并逐步渲染数据”的数据源，不再额外新增第二套对外接口，仍然收敛到现有 `Source` 上的两个公共方法：

```go
ListKLines(ctx context.Context, query KLineQuery) ([]KLine, error)
StreamKLines(ctx context.Context, query KLineQuery) (<-chan KLineStreamItem, error)
```

但允许数据源**在内部**引入一个基于浏览器页面监听的流式抓取能力，用来持续消费页面发起的接口响应，再把解析结果回推给上层聚合逻辑。

推荐内部辅助能力形状：

```go
type PageResponseStreamItem struct {
	URL        string
	ReceivedAt time.Time
	Body       []byte
	Err        error
}

type PageResponseWatcher interface {
	Watch(ctx context.Context, pageURL string, opts WatchOptions) (<-chan PageResponseStreamItem, error)
}
```

这里的关键约束是：

1. `PageResponseWatcher` 仍然是**内部实现细节**，即便系统对外新增了 `StreamKLines`，也不直接把页面监听细节暴露到公共接口里。
2. watcher 负责“持续监听页面网络响应并通过 `chan` 推送结果”，业务层负责把响应 body 解析成 K 线。
3. `ListKLines` 可以在内部消费这个 `chan`，直到拿到足够数据后返回标准 `[]KLine`；`StreamKLines` 则把解析后的批次继续向调用方透出。
4. 如果后续需要更强的流式场景，也应先在内部层扩展，不直接污染当前 `Source` 契约。

### 2.1.1 watcher 停止语义

页面监听型抓取必须支持以下停止条件：

1. 调用方 `ctx` 被取消或超时。
2. 页面被关闭，或底层浏览器标签页失效。
3. 页面在一段时间内再也没有命中目标接口响应。

其中第 3 点是这类实现的关键：因为页面型数据源的真实完成时机，往往不是“DOMContentLoaded”，而是“目标接口已经一段时间没有新响应”。因此 watcher 必须以内置 idle timeout 作为正常收口条件之一，而不是无限阻塞等待。

### 2.1.2 watcher 过滤与解析语义

为了避免把图片、埋点、静态资源等噪音响应混进 K 线链路，watcher 需要支持按以下维度过滤目标响应：

1. URL 模式
2. 资源类型（如 XHR / fetch）
3. 必要时按响应内容特征做二次判定

推荐职责拆分：

1. watcher
   只负责监听页面、过滤目标响应、输出原始 body。
2. parser
   负责把单条 body 解析成若干条候选 K 线。
3. collector
   负责去重、排序、截断，以及按 `KLineQuery` 的 `Limit / StartTime / EndTime` 语义收口。

## 2.2 复权处理结论

本次设计对复权采取“三层分离”策略：

1. 采集层
   `datasource.Source` 只返回第一手原始行情，默认不复权。
2. 清洗层
   清洗流程负责合并公司行为、生成复权因子或等价的复权支撑数据，并将其入库。
3. 使用层
   实际对外分析、回测或查询数据库时，再按业务需求选择 `none / qfq / hfq` 等复权视图。

这意味着：

1. 本次新增的 `Source.ListKLines` 默认返回原始不复权行情。
2. 系统不能在采集层隐式做前复权或后复权。
3. 若未来某个外部源原生支持复权，也不改变本层的默认契约；复权仍应视为清洗后或查询时的显式能力。

## 3. 数据结构

新增通用周期枚举：

```go
type Interval string

const (
	Interval1Min    Interval = "1m"
	Interval5Min    Interval = "5m"
	Interval15Min   Interval = "15m"
	Interval30Min   Interval = "30m"
	Interval60Min   Interval = "60m"
	IntervalDay     Interval = "1d"
	IntervalWeek    Interval = "1w"
	IntervalMonth   Interval = "1mo"
	IntervalQuarter Interval = "1q"
	IntervalYear    Interval = "1y"
)
```

新增通用 K 线结构：

```go
type KLine struct {
	TSCode    string
	TradeTime time.Time
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	PreClose  decimal.Decimal
	Change    decimal.Decimal
	PctChg    decimal.Decimal
	Vol       decimal.Decimal
	Amount    decimal.Decimal
	Source    string
}
```

新增查询参数：

```go
type KLineQuery struct {
	TSCode    string
	Interval  Interval
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

type KLineStreamItem struct {
	Items []KLine
	Err   error
}
```

### 3.1 设计原则

1. `KLine` 字段尽量与 `DailyBar` 对齐，减少日线实现复用成本。
2. 时间字段统一命名为 `TradeTime`，分钟线表示分钟时间，日/周/月线表示对应周期时间点。
3. `KLine` 表达原始不复权行情，不在采集接口层混入复权视图语义。
4. 复权因子不放进本次 `KLine` 结构，避免把清洗层和采集层耦合。

## 4. 查询语义

### 4.1 必填字段

以下字段必须提供：

1. `TSCode`
2. `Interval`

### 4.2 查询模式

`ListKLines` 支持两种模式：

1. 最近 N 条模式
   当 `Limit > 0` 时，按“截止到 `EndTime` 的最近 N 条”查询。
2. 时间范围模式
   当 `Limit <= 0` 时，按 `StartTime ~ EndTime` 查询。

### 4.3 默认值与优先级

1. `Limit > 0` 且 `EndTime` 为空时，以当前时间为截止点。
2. `Limit > 0` 时，以 `Limit` 语义为主，`StartTime` 忽略。
3. `Limit <= 0` 且 `StartTime`、`EndTime` 都为空时，返回参数错误。
4. 返回结果按 `TradeTime` 升序排列。

### 4.4 周期时间解释

1. 分钟线按精确时间处理。
2. 日/周/月/季/年线按周期时间边界处理。
3. 调用方不应依赖时间的具体时分秒语义，只应依赖其排序与周期归属。

### 4.5 复权查询语义

1. `ListKLines` 第一版只定义原始行情查询语义，不携带 `Adjust` 参数。
2. 调用方如果需要前复权、后复权视图，应在清洗入库后的数据库查询层完成。
3. 清洗层需要保证原始 K 线与复权因子可以关联重建出稳定的一致结果。

## 5. 错误语义

### 5.1 参数错误

以下场景返回 `apperror.CodeBadRequest`：

1. `TSCode` 为空。
2. `Interval` 为空。
3. `StartTime > EndTime`。
4. `Limit <= 0` 且未提供有效时间范围。

### 5.2 数据源能力不足

当数据源不支持指定周期时，返回 `apperror.CodeDatasourceUnavailable`，错误消息明确指出数据源与周期，例如：

```text
tushare datasource does not support interval 5m
```

如果调用方错误地假设采集层已返回复权数据，不单独新增错误码；系统通过文档契约明确 `ListKLines` 返回的是原始行情，避免隐式行为。

## 6. 各数据源落地策略

### 6.1 EastMoney

EastMoney 作为第一版主实现方，完整支持：

1. `1m`
2. `5m`
3. `15m`
4. `30m`
5. `60m`
6. `1d`
7. `1w`
8. `1mo`
9. `1q`
10. `1y`

实现策略：

1. 复用现有 `internal/domain/marketdata/eastmoney` 中的单票多周期查询能力。
2. 将现有 K 线映射与周期参数映射下沉或抽出为可被 `datasource/eastmoney.Source` 直接复用的内部能力。
3. `ListDailyBars` 继续保留为“全市场按日期导入”的实现，不改成 `ListKLines` 包装器。
4. 即使 EastMoney 底层原生支持复权参数，`datasource/eastmoney.Source` 第一版仍只返回原始不复权行情。

### 6.2 Tushare

第一版仅支持：

1. `1d`

实现策略：

1. 直接复用现有 `ListDailyBars` 查询链路。
2. 将单日线结果映射为 `KLine` 返回。
3. 对分钟线、周线、月线、季线、年线直接返回“不支持周期”错误。
4. 不因为下游需要复权而在 Tushare 采集接口层额外做复权换算。

### 6.3 Sample

第一版仅支持：

1. `1d`

实现策略：

1. 复用现有样例日线数据。
2. 将 `DailyBar` 直接映射为 `KLine`。
3. 不为样例源增加分钟线或周/月聚合，避免制造伪能力。
4. 样例源也保持原始不复权语义，测试数据若需复权，应通过独立的清洗/查询测试覆盖。

### 6.4 Sina 页面驱动型实现

新浪实现优先提供 `ListKLines` / `StreamKLines`，不要求它成为完整的批量导入源。当前实现允许数据源在单票 K 线能力上更强，而在批量导入接口上显式返回“不支持”。

如果新浪依赖浏览器上下文来拉取分钟线，则采用本设计第 `2.1` 节的内部 watcher 方案；当前已落地的最小实现不是被动等待股票详情页自己发请求，而是由浏览器直接访问新浪 JSONP K 线接口，再复用 watcher/collector 收口单次响应：

1. 构造带 `symbol / scale / datalen` 的新浪 JSONP K 线请求 URL。
2. 在浏览器上下文里访问该 URL，并监听命中响应。
3. 每拿到一条命中响应，就解析并推入内部 `chan`。
4. 当 `ctx` 结束、idle timeout 到达，或零命中超时触发时停止监听。
5. 由 collector 汇总这些流式结果，最终返回 `[]KLine`。

这样可以保证：

1. `datasource.Source` 对外接口保持稳定。
2. 新浪实现仍然复用浏览器上下文与统一 watcher/collector 分层，而不是在 `Source` 里散落一套独立脚本化抓取逻辑。
3. 后续如果别的数据源也需要同类能力，可以复用同一套 watcher/collector 分层，而不是把页面监听逻辑散落到各个 `Source` 实现里。

当前能力边界：

1. 只支持 `Limit > 0` 的 latest-N 查询。
2. 不支持 `Limit <= 0` 的显式历史时间窗查询。
3. 不支持调用方显式指定 `EndTime`；截止点固定为“当前请求时刻”。
4. 单次请求最大 `Limit` 为 `1023`，超过时显式报“不支持当前 datasource 能力”。

## 7. 兼容与边界

### 7.1 保持现有导入任务不变

以下能力保持语义不变：

1. `ListStocks`
2. `ListTradeCalendar`
3. `ListDailyBars`

导入任务、预加载样例、因子计算等现有流程继续使用这些批量接口。

### 7.1.1 流式能力矩阵

第一版流式能力约束如下：

1. `sample`
   必须实现 `StreamKLines`，但固定返回“不支持流式接口”错误。
2. `tushare`
   必须实现 `StreamKLines`，但固定返回“不支持流式接口”错误。
3. `eastmoney`
   必须实现 `StreamKLines`，但固定返回“不支持流式接口”错误。
4. `sina`
   已实现为页面驱动型 `StreamKLines` 数据源；当前支持通过 watcher 持续消费页面响应并向上游回推解析后的 K 线批次。

### 7.2 不强制复用日线接口

虽然 `1d` 理论上可以由 `ListDailyBars` 包装得到，但第一版不强行把所有实现统一成某一种底层形态。每个数据源只要满足同一输出契约即可。

### 7.3 不做第一版聚合

第一版不为 `sample` 或 `tushare` 实现周/月/季/年聚合，不把“可推导”误当成“已支持”。只有数据源真正具备该能力时才返回结果。

### 7.4 复权职责边界

系统应将复权职责固定在清洗与使用阶段，而不是采集阶段：

1. 采集阶段
   保存原始价格事实，不覆盖、不改写原始 K 线。
2. 清洗阶段
   补齐公司行为、生成复权因子，并与原始行情一起入库。
3. 使用阶段
   数据库查询或分析服务按业务需要选择不复权、前复权或后复权视图。

如果后续为了性能需要物化前复权表，也应视为清洗后的派生结果，而不是替代原始行情底表。

## 8. 测试策略

### 8.1 通用类型与参数校验

新增测试覆盖：

1. `Interval` 默认/非法值校验。
2. `KLineQuery` 参数校验逻辑。
3. `Limit` 与时间范围优先级规则。
4. `ListKLines` 返回原始不复权语义的文档化断言或注释说明保持一致。

### 8.2 EastMoney

新增测试覆盖：

1. 不同 `Interval` 到东财 `klt` 的映射。
2. 分钟线、日线、周线至少各一组成功映射用例。
3. 不合法 `TSCode`、不支持周期、空数据与回退场景。

### 8.3 Tushare 与 Sample

新增测试覆盖：

1. `IntervalDay` 成功返回。
2. 非日线周期返回 `CodeDatasourceUnavailable`。

## 9. 实施顺序

建议按以下顺序落地：

1. 扩展 `datasource/types.go`，新增 `Interval`、`KLine`、`KLineQuery` 与 `Source.ListKLines`。
2. 为 `sample` 增加仅支持日线的实现与测试。
3. 为 `tushare` 增加仅支持日线的实现与测试。
4. 为 `eastmoney` 复用现有 richer K 线能力，补齐 `ListKLines` 与测试。
5. 在后续清洗设计中补充复权因子表或等价公司行为模型。
6. 视需要再评估是否把 `marketdata/eastmoney` 与 `datasource/eastmoney` 的公共 K 线逻辑进一步抽取。

## 10. 本次推荐方案摘要

本次推荐方案是在不破坏现有导入接口语义的前提下，为 `datasource.Source` 增加一个统一的单票 K 线查询入口，并明确采集层只返回原始不复权行情。第一版以 EastMoney 作为完整实现，Tushare 与 Sample 只支持日线并对其他周期显式报错；复权能力放到清洗入库和后续数据库查询阶段处理。这一方案改动面可控、边界清晰，也为未来迁移 Sina 这类单票 K 线能力更强的数据源预留了稳定落点。

# Finscope Datasource Skeleton Design

## 1. 目标

新增一个暂命名为 `finscope` 的数据源目录骨架，为后续“完全基于无头浏览器监听页面请求与响应”的采集实现预留稳定边界。

本次先完成第一阶段初始化，并落地第一条真实抓取子链路：

1. 初始化浏览器驱动型数据源骨架；
2. 实现 `ListStocks` 下的第一个串行子方法，用于抓取“上证指数成分股”；
3. 其余能力仍保持未实现状态。

首版目标：

1. 新增 `apps/server/internal/domain/datasource/finscope/` 目录。
2. 提供完整的 `datasource.Source` 接口空实现。
3. 预留浏览器 watcher、解析器、内部配置等文件边界。
4. 让 `ListStocks` 可以按“子方法串行执行”的方式逐步扩市场分页能力。
5. 让后续实现可以先补 `StreamKLines`，再复用到 `ListKLines`。

## 2. 设计结论

采用“浏览器驱动型数据源完整骨架”的方案，结构上对齐 `sina` 的页面驱动能力，但首版不放具体站点逻辑。

目录结构：

```text
apps/server/internal/domain/datasource/finscope/
├── source.go
├── types.go
├── watcher.go
├── parser.go
└── source_test.go
```

## 3. 文件职责

### 3.1 `source.go`

负责对外暴露数据源入口：

1. 定义 `Source` 结构体。
2. 定义 `New(...)` 构造函数。
3. 实现 `datasource.Source` 五个接口方法。
4. 提供统一的未实现/不支持错误封装。
5. 提供 `ListStocks` 的串行子方法调度入口。

首版中：

首版中：

1. `ListStocks` 已接入第一条真实子方法：上证指数成分股；
2. `ListTradeCalendar`
3. `ListDailyBars`
4. `ListKLines`
5. `StreamKLines`

除 `ListStocks` 外，其余能力仍返回显式错误，不做假成功。

### 3.2 `types.go`

负责放 `finscope` 私有类型，避免 `source.go` 过早膨胀：

1. 常量，如 `sourceName`
2. 内部配置结构
3. 浏览器匹配元信息
4. 后续需要的内部流式事件结构

首版只放最小必要定义，不提前设计复杂 DTO。

### 3.3 `watcher.go`

负责浏览器监听边界，不承载业务解析逻辑。

首版只定义：

1. 页面驱动抓取的最小依赖边界
2. 观察单页成分股响应的请求入口
3. 浏览器未配置时的统一错误

不在这里直接写站点字段映射逻辑。

### 3.4 `parser.go`

负责解析层边界。

首版已经实现“百度财经成分股响应 -> `[]datasource.StockBasic`”的解析。

仍然保持约束：

1. parser 只负责响应体解析；
2. 不在 parser 中做浏览器监听；
3. 不在 parser 中串行翻页。

### 3.5 `source_test.go`

负责验证骨架级行为：

1. `Source` 满足 `datasource.Source` 接口
2. 构造函数可用
3. 空实现返回显式错误而不是空结果

## 4. 构造与依赖设计

构造函数设计为：

```go
func New(browser browserfetch.Runner, opts ...Option) *Source
```

其中：

1. `browser` 是未来无头浏览器抓取能力的唯一外部依赖。
2. `Option` 用于后续补充页面 URL、超时、站点模式等配置。
3. 首版允许 `browser == nil`，但相关能力调用时必须返回明确错误。

这样做的原因：

1. 现在先保留完整扩展位；
2. 后续接真实页面时不需要再重做构造函数；
3. 测试中可直接注入 stub watcher。

当前 `ListStocks` 的第一条子链路严格遵循真实页面流程，但实现上不再依赖 CDP `Network` 域抓包，而是改为在页面上下文里预注入脚本，拦截 `fetch/XMLHttpRequest` 返回体后再滚动触发后续分页。

百度财经这条链路还有一个站点级约束需要保留给维护者：

1. 不能先做浏览器“预热式”的空 `chromedp.Run(...)` 初始化；
2. 不能在第二个子 tab 里打开该页面；
3. 必须直接在独立浏览器进程的主页面 target 上执行底层 `page.Navigate`，否则页面会立刻返回 `context canceled`。

具体流程：

1. 在独立浏览器进程的主页面 target 上打开 `https://finance.baidu.com/index/ab-000001?mainTab=成分股`；
2. 在导航前向页面注入响应捕获脚本；
3. 使用底层 `page.Navigate` 发起导航，不等待整页 `load` 事件；
4. 由页面自身发出 `https://finance.pae.baidu.com/sapi/v1/constituents` 请求；
5. 在同一个页面上下文里持续向下滚动，触发后续分页懒加载；
6. 从页面缓存中收集多批响应体并逐批解析、最终去重收口。

## 5. 错误语义

首版区分两类错误：

1. **未实现**
   表示该能力未来计划支持，但当前骨架阶段尚未接入。
2. **不支持**
   表示该能力按当前数据源定位不提供。

当前建议：

1. `ListStocks` 的已实现子方法若失败，返回带子方法名和页码上下文的错误；
2. 未实现能力继续统一返回“未实现”错误；
3. 暂不把任何能力声明为永久不支持；
4. 等真实接入方案明确后，再决定哪些接口应改成 permanent unsupported。

## 6. 后续实现顺序

建议按以下顺序逐步完善：

1. 继续给 `ListStocks` 增加第二、第三个市场分页子方法；
2. 复用同一套“页面打开 + 页面脚本拦截 + 页面动作”能力接其它百度财经列表；
3. 再补 parser，确认其它响应如何提取结构化结果；
4. 优先实现 `StreamKLines`；
5. 最后让 `ListKLines` 复用流式结果做收口；
6. 视站点能力再决定是否补 `ListTradeCalendar` / `ListDailyBars`。

## 7. 验证范围

本次初始化完成后，只要求通过骨架级验证：

1. 新目录与文件结构存在；
2. `Source` 实现 `datasource.Source`；
3. 构造函数可调用；
4. `ListStocks` 能通过真实页面监听流程收集至少一批成分股响应；
5. 其余未实现方法继续返回显式错误；
6. 不引入未使用依赖和编译错误。

不要求：

1. 生产环境联调
2. 多市场全部接入
3. 页面结构突变后的自恢复策略
4. 生产可用

## 8. 范围外

本次不做：

1. 真实 Finscope 页面地址或协议分析
2. 真实请求参数拼装
3. 真实响应字段映射
4. 任务调度接入
5. 前端或 API 对接

---
name: baidu-browserfetch-pitfalls
description: Use when implementing or debugging browser-driven scraping for Baidu Finance or similar SPA pages that cancel child-tab navigations, hang in chromedp.Navigate, or require capturing in-page fetch/XMLHttpRequest responses during real page interaction.
---

# baidu-browserfetch-pitfalls

## 目标

把 QuantSage 在百度财经页面上踩过的真实坑沉淀成稳定操作规则，避免重复走弯路。

## 何时使用

满足任一情况时优先使用本 skill：

- 处理百度财经、强前端渲染、对子 tab 敏感的页面
- 需要在真实页面流程里拦截 `fetch` / `XMLHttpRequest` 返回体
- `chromedp.Navigate(...)` 长时间卡住、超时或误报 `context canceled`
- 页面需要滚动、懒加载后才会发出目标接口请求

不适用场景：

- 纯 HTTP API 抓取，无需真实浏览器参与
- 静态页面解析，不涉及前端异步请求

## 核心四元组

### 1. 页面内响应监听

- **Scene**：抓取百度财经页面并监听页面内 `fetch/XMLHttpRequest` 响应
- **Wrong**：先做一次空 `chromedp.Run(...)` 预热，再执行真正导航
- **Right**：直接让第一次真实页面动作启动浏览器，不要额外做预热初始化
- **Reason**：百度财经会把后续真实导航直接打成 `context canceled`

### 2. 业务页打开方式

- **Scene**：打开百度财经指数成分股页这类真实业务页面
- **Wrong**：在浏览器已创建后的第二个子 tab 里再开业务页面
- **Right**：优先使用独立浏览器进程的主页面 target 打开业务页面
- **Reason**：该站点对子 tab 很敏感，主页面 target 更稳定

### 3. 导航方式排障

- **Scene**：页面能打开，但 `chromedp.Navigate(...)` 长时间卡住或超时
- **Wrong**：默认继续怀疑 `WaitReady`、脚本注入或 parser
- **Right**：先验证底层 `page.Navigate` 是否可用，再决定是否切到原始导航模式
- **Reason**：`chromedp.Navigate` 等待的加载语义更重，容易把问题诊断带偏

### 4. 分页数据采集

- **Scene**：需要拦截分页接口并持续滚动收集多批数据
- **Wrong**：直接依赖 CDP `Network` 域抓包，或跳过真实页面流程手拼请求
- **Right**：在导航前注入页面脚本，拦截页面自身 `fetch/XMLHttpRequest` 返回体，再在同页上下文滚动触发懒加载
- **Reason**：更贴近真实站点行为，也更能绕开 Cookie、时序和懒加载耦合

## 推荐实现顺序

1. 先确认目标页面是否属于“强页面耦合站点”
2. 优先尝试主页面 target，而不是复用已有子 tab
3. 如果普通导航异常，先对照 `page.Navigate`
4. 如果目标数据来自页面内异步请求，优先用导航前脚本注入拦截
5. 改浏览器抓取行为前，先补最小 smoke test，再改实现

## 与 QuantSage 的对应点

- 浏览器基础设施优先复用 `apps/server/internal/infra/browserfetch`
- 需要“导航前注入 + 导航后动作”时，优先考虑 `RunWithActions`
- 需要页面内触发滚动、懒加载、持续收集时，不要把 parser 和浏览器动作混在一起

## 常见误判

- “先空跑一次浏览器更稳”：
  在百度财经这类页面上通常更不稳

- “子 tab 和主页面 target 没区别”：
  对普通站点可能成立，对百度财经不成立

- “能打开页面就说明导航链路没问题”：
  页面可见不等于请求监听时序正确

- “直接抓 CDP Network 一定最底层最稳”：
  对这类强耦合页面，页面内拦截常常更接近真实行为

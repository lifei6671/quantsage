CREATE TABLE stock_basic (
    ts_code TEXT PRIMARY KEY,
    symbol TEXT NOT NULL,
    name TEXT NOT NULL,
    area TEXT,
    industry TEXT,
    market TEXT,
    exchange TEXT NOT NULL,
    list_date DATE,
    delist_date DATE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    source TEXT NOT NULL,
    source_updated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE trade_calendar (
    exchange TEXT NOT NULL,
    cal_date DATE NOT NULL,
    is_open BOOLEAN NOT NULL,
    pretrade_date DATE,
    source TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (exchange, cal_date)
);

CREATE TABLE stock_daily (
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    trade_date DATE NOT NULL,
    open NUMERIC(18,4) NOT NULL,
    high NUMERIC(18,4) NOT NULL,
    low NUMERIC(18,4) NOT NULL,
    close NUMERIC(18,4) NOT NULL,
    pre_close NUMERIC(18,4),
    change NUMERIC(18,4),
    pct_chg NUMERIC(10,4),
    vol NUMERIC(20,4) NOT NULL DEFAULT 0,
    amount NUMERIC(20,4) NOT NULL DEFAULT 0,
    source TEXT NOT NULL,
    data_status TEXT NOT NULL DEFAULT 'clean',
    source_updated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (ts_code, trade_date)
);

CREATE TABLE adj_factor (
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    trade_date DATE NOT NULL,
    adj_factor NUMERIC(20,8) NOT NULL,
    source TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (ts_code, trade_date)
);

CREATE TABLE financial_indicator (
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    report_period DATE NOT NULL,
    ann_date DATE,
    end_date DATE NOT NULL,
    eps NUMERIC(18,6),
    diluted_eps NUMERIC(18,6),
    roe NUMERIC(18,6),
    roa NUMERIC(18,6),
    gross_margin NUMERIC(18,6),
    net_profit_margin NUMERIC(18,6),
    debt_to_assets NUMERIC(18,6),
    current_ratio NUMERIC(18,6),
    revenue_yoy NUMERIC(18,6),
    profit_yoy NUMERIC(18,6),
    source TEXT NOT NULL,
    version TEXT NOT NULL DEFAULT 'v1',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (ts_code, report_period, version)
);

CREATE TABLE stock_factor_daily (
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    trade_date DATE NOT NULL,
    ma5 NUMERIC(18,4),
    ma10 NUMERIC(18,4),
    ma20 NUMERIC(18,4),
    ma60 NUMERIC(18,4),
    ema12 NUMERIC(18,6),
    ema26 NUMERIC(18,6),
    macd_dif NUMERIC(18,6),
    macd_dea NUMERIC(18,6),
    macd_hist NUMERIC(18,6),
    rsi6 NUMERIC(10,4),
    rsi12 NUMERIC(10,4),
    rsi24 NUMERIC(10,4),
    volume_ma5 NUMERIC(20,4),
    volume_ma20 NUMERIC(20,4),
    volume_ratio NUMERIC(10,4),
    amplitude NUMERIC(10,4),
    upper_shadow_ratio NUMERIC(10,4),
    lower_shadow_ratio NUMERIC(10,4),
    close_above_ma5 BOOLEAN,
    close_above_ma10 BOOLEAN,
    close_above_ma20 BOOLEAN,
    ma_bullish BOOLEAN,
    volume_breakout BOOLEAN,
    price_breakout_20 BOOLEAN,
    factor_version TEXT NOT NULL DEFAULT 'v1',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (ts_code, trade_date, factor_version)
);

CREATE TABLE stock_event_tag (
    id BIGSERIAL PRIMARY KEY,
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    trade_date DATE NOT NULL,
    event_type TEXT NOT NULL,
    event_level TEXT NOT NULL DEFAULT 'info',
    score NUMERIC(10,4),
    description TEXT NOT NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    version TEXT NOT NULL DEFAULT 'v1',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE strategy_definition (
    id BIGSERIAL PRIMARY KEY,
    strategy_code TEXT NOT NULL,
    strategy_name TEXT NOT NULL,
    strategy_type TEXT NOT NULL,
    description TEXT,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    version TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (strategy_code, version)
);

CREATE TABLE strategy_signal (
    id BIGSERIAL PRIMARY KEY,
    strategy_code TEXT NOT NULL,
    strategy_version TEXT NOT NULL,
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    trade_date DATE NOT NULL,
    signal_type TEXT NOT NULL,
    signal_strength NUMERIC(10,4) NOT NULL DEFAULT 0,
    signal_level TEXT NOT NULL DEFAULT 'D',
    buy_price_ref NUMERIC(18,4),
    stop_loss_ref NUMERIC(18,4),
    take_profit_ref NUMERIC(18,4),
    invalidation_condition TEXT,
    reason TEXT NOT NULL,
    input_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (strategy_code, strategy_version, ts_code, trade_date, signal_type)
);

CREATE TABLE job_run_log (
    id BIGSERIAL PRIMARY KEY,
    job_name TEXT NOT NULL,
    biz_date DATE,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    error_code INT NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    retry_count INT NOT NULL DEFAULT 0,
    progress_current INT NOT NULL DEFAULT 0,
    progress_total INT NOT NULL DEFAULT 0,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE app_user (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    role TEXT NOT NULL DEFAULT 'user',
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE watchlist_group (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, name)
);

CREATE TABLE watchlist_item (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES watchlist_group(id) ON DELETE CASCADE,
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (group_id, ts_code)
);

CREATE TABLE user_position (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    position_date DATE NOT NULL,
    quantity NUMERIC(20,4) NOT NULL,
    cost_price NUMERIC(18,4) NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, ts_code, position_date)
);

CREATE TABLE announcement (
    id BIGSERIAL PRIMARY KEY,
    ts_code TEXT REFERENCES stock_basic(ts_code),
    title TEXT NOT NULL,
    announcement_type TEXT NOT NULL DEFAULT 'unknown',
    publish_time TIMESTAMPTZ,
    source TEXT NOT NULL,
    source_url TEXT,
    object_key TEXT,
    content_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE document_chunk (
    id BIGSERIAL PRIMARY KEY,
    document_type TEXT NOT NULL,
    document_id BIGINT NOT NULL,
    chunk_index INT NOT NULL,
    content TEXT NOT NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (document_type, document_id, chunk_index)
);

CREATE TABLE document_embedding (
    chunk_id BIGINT PRIMARY KEY REFERENCES document_chunk(id) ON DELETE CASCADE,
    embedding vector(1536),
    embedding_model TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ai_stock_analysis (
    id BIGSERIAL PRIMARY KEY,
    ts_code TEXT NOT NULL REFERENCES stock_basic(ts_code),
    analysis_date DATE NOT NULL,
    analysis_type TEXT NOT NULL,
    prompt_version TEXT NOT NULL,
    conclusion TEXT NOT NULL,
    evidence JSONB NOT NULL DEFAULT '[]'::jsonb,
    risks JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE stock_basic IS '股票基础信息表';
COMMENT ON COLUMN stock_basic.ts_code IS 'Tushare 股票代码，主键';
COMMENT ON COLUMN stock_basic.symbol IS '股票短代码';
COMMENT ON COLUMN stock_basic.name IS '股票名称';
COMMENT ON COLUMN stock_basic.area IS '所属地区';
COMMENT ON COLUMN stock_basic.industry IS '所属行业';
COMMENT ON COLUMN stock_basic.market IS '市场类型';
COMMENT ON COLUMN stock_basic.exchange IS '交易所代码';
COMMENT ON COLUMN stock_basic.list_date IS '上市日期';
COMMENT ON COLUMN stock_basic.delist_date IS '退市日期';
COMMENT ON COLUMN stock_basic.is_active IS '是否仍在交易';
COMMENT ON COLUMN stock_basic.source IS '数据来源';
COMMENT ON COLUMN stock_basic.source_updated_at IS '源数据更新时间';
COMMENT ON COLUMN stock_basic.created_at IS '记录创建时间';
COMMENT ON COLUMN stock_basic.updated_at IS '记录更新时间';

COMMENT ON TABLE trade_calendar IS '交易日历表';
COMMENT ON COLUMN trade_calendar.exchange IS '交易所代码';
COMMENT ON COLUMN trade_calendar.cal_date IS '日历日期';
COMMENT ON COLUMN trade_calendar.is_open IS '是否开市';
COMMENT ON COLUMN trade_calendar.pretrade_date IS '上一交易日';
COMMENT ON COLUMN trade_calendar.source IS '数据来源';
COMMENT ON COLUMN trade_calendar.created_at IS '记录创建时间';
COMMENT ON COLUMN trade_calendar.updated_at IS '记录更新时间';

COMMENT ON TABLE stock_daily IS '股票日线行情表';
COMMENT ON COLUMN stock_daily.ts_code IS '股票代码';
COMMENT ON COLUMN stock_daily.trade_date IS '交易日期';
COMMENT ON COLUMN stock_daily.open IS '开盘价';
COMMENT ON COLUMN stock_daily.high IS '最高价';
COMMENT ON COLUMN stock_daily.low IS '最低价';
COMMENT ON COLUMN stock_daily.close IS '收盘价';
COMMENT ON COLUMN stock_daily.pre_close IS '前收盘价';
COMMENT ON COLUMN stock_daily.change IS '涨跌额';
COMMENT ON COLUMN stock_daily.pct_chg IS '涨跌幅';
COMMENT ON COLUMN stock_daily.vol IS '成交量';
COMMENT ON COLUMN stock_daily.amount IS '成交额';
COMMENT ON COLUMN stock_daily.source IS '数据来源';
COMMENT ON COLUMN stock_daily.data_status IS '数据状态，例如 clean 或 dirty';
COMMENT ON COLUMN stock_daily.source_updated_at IS '源数据更新时间';
COMMENT ON COLUMN stock_daily.created_at IS '记录创建时间';
COMMENT ON COLUMN stock_daily.updated_at IS '记录更新时间';

COMMENT ON TABLE adj_factor IS '复权因子表';
COMMENT ON COLUMN adj_factor.ts_code IS '股票代码';
COMMENT ON COLUMN adj_factor.trade_date IS '交易日期';
COMMENT ON COLUMN adj_factor.adj_factor IS '复权因子';
COMMENT ON COLUMN adj_factor.source IS '数据来源';
COMMENT ON COLUMN adj_factor.created_at IS '记录创建时间';
COMMENT ON COLUMN adj_factor.updated_at IS '记录更新时间';

COMMENT ON TABLE financial_indicator IS '财务指标表';
COMMENT ON COLUMN financial_indicator.ts_code IS '股票代码';
COMMENT ON COLUMN financial_indicator.report_period IS '报告期';
COMMENT ON COLUMN financial_indicator.ann_date IS '公告日期';
COMMENT ON COLUMN financial_indicator.end_date IS '财报截止日期';
COMMENT ON COLUMN financial_indicator.eps IS '每股收益';
COMMENT ON COLUMN financial_indicator.diluted_eps IS '稀释每股收益';
COMMENT ON COLUMN financial_indicator.roe IS '净资产收益率';
COMMENT ON COLUMN financial_indicator.roa IS '总资产收益率';
COMMENT ON COLUMN financial_indicator.gross_margin IS '毛利率';
COMMENT ON COLUMN financial_indicator.net_profit_margin IS '净利率';
COMMENT ON COLUMN financial_indicator.debt_to_assets IS '资产负债率';
COMMENT ON COLUMN financial_indicator.current_ratio IS '流动比率';
COMMENT ON COLUMN financial_indicator.revenue_yoy IS '营业收入同比增长率';
COMMENT ON COLUMN financial_indicator.profit_yoy IS '净利润同比增长率';
COMMENT ON COLUMN financial_indicator.source IS '数据来源';
COMMENT ON COLUMN financial_indicator.version IS '指标口径版本';
COMMENT ON COLUMN financial_indicator.created_at IS '记录创建时间';
COMMENT ON COLUMN financial_indicator.updated_at IS '记录更新时间';

COMMENT ON TABLE stock_factor_daily IS '股票日频技术因子表';
COMMENT ON COLUMN stock_factor_daily.ts_code IS '股票代码';
COMMENT ON COLUMN stock_factor_daily.trade_date IS '交易日期';
COMMENT ON COLUMN stock_factor_daily.ma5 IS '5 日均线';
COMMENT ON COLUMN stock_factor_daily.ma10 IS '10 日均线';
COMMENT ON COLUMN stock_factor_daily.ma20 IS '20 日均线';
COMMENT ON COLUMN stock_factor_daily.ma60 IS '60 日均线';
COMMENT ON COLUMN stock_factor_daily.ema12 IS '12 日指数均线';
COMMENT ON COLUMN stock_factor_daily.ema26 IS '26 日指数均线';
COMMENT ON COLUMN stock_factor_daily.macd_dif IS 'MACD DIF';
COMMENT ON COLUMN stock_factor_daily.macd_dea IS 'MACD DEA';
COMMENT ON COLUMN stock_factor_daily.macd_hist IS 'MACD 柱值';
COMMENT ON COLUMN stock_factor_daily.rsi6 IS '6 日 RSI';
COMMENT ON COLUMN stock_factor_daily.rsi12 IS '12 日 RSI';
COMMENT ON COLUMN stock_factor_daily.rsi24 IS '24 日 RSI';
COMMENT ON COLUMN stock_factor_daily.volume_ma5 IS '5 日均量';
COMMENT ON COLUMN stock_factor_daily.volume_ma20 IS '20 日均量';
COMMENT ON COLUMN stock_factor_daily.volume_ratio IS '量比';
COMMENT ON COLUMN stock_factor_daily.amplitude IS '振幅';
COMMENT ON COLUMN stock_factor_daily.upper_shadow_ratio IS '上影线比例';
COMMENT ON COLUMN stock_factor_daily.lower_shadow_ratio IS '下影线比例';
COMMENT ON COLUMN stock_factor_daily.close_above_ma5 IS '收盘价是否站上 5 日均线';
COMMENT ON COLUMN stock_factor_daily.close_above_ma10 IS '收盘价是否站上 10 日均线';
COMMENT ON COLUMN stock_factor_daily.close_above_ma20 IS '收盘价是否站上 20 日均线';
COMMENT ON COLUMN stock_factor_daily.ma_bullish IS '均线是否多头排列';
COMMENT ON COLUMN stock_factor_daily.volume_breakout IS '是否放量';
COMMENT ON COLUMN stock_factor_daily.price_breakout_20 IS '是否突破 20 日新高';
COMMENT ON COLUMN stock_factor_daily.factor_version IS '因子版本';
COMMENT ON COLUMN stock_factor_daily.created_at IS '记录创建时间';
COMMENT ON COLUMN stock_factor_daily.updated_at IS '记录更新时间';

COMMENT ON TABLE stock_event_tag IS '股票事件标签表';
COMMENT ON COLUMN stock_event_tag.id IS '主键';
COMMENT ON COLUMN stock_event_tag.ts_code IS '股票代码';
COMMENT ON COLUMN stock_event_tag.trade_date IS '交易日期';
COMMENT ON COLUMN stock_event_tag.event_type IS '事件类型';
COMMENT ON COLUMN stock_event_tag.event_level IS '事件级别';
COMMENT ON COLUMN stock_event_tag.score IS '事件评分';
COMMENT ON COLUMN stock_event_tag.description IS '事件描述';
COMMENT ON COLUMN stock_event_tag.meta IS '事件补充信息';
COMMENT ON COLUMN stock_event_tag.version IS '标签版本';
COMMENT ON COLUMN stock_event_tag.created_at IS '记录创建时间';

COMMENT ON TABLE strategy_definition IS '策略定义表';
COMMENT ON COLUMN strategy_definition.id IS '主键';
COMMENT ON COLUMN strategy_definition.strategy_code IS '策略编码';
COMMENT ON COLUMN strategy_definition.strategy_name IS '策略名称';
COMMENT ON COLUMN strategy_definition.strategy_type IS '策略类型';
COMMENT ON COLUMN strategy_definition.description IS '策略说明';
COMMENT ON COLUMN strategy_definition.config IS '策略配置 JSON';
COMMENT ON COLUMN strategy_definition.enabled IS '是否启用';
COMMENT ON COLUMN strategy_definition.version IS '策略版本';
COMMENT ON COLUMN strategy_definition.created_at IS '记录创建时间';
COMMENT ON COLUMN strategy_definition.updated_at IS '记录更新时间';

COMMENT ON TABLE strategy_signal IS '策略信号结果表';
COMMENT ON COLUMN strategy_signal.id IS '主键';
COMMENT ON COLUMN strategy_signal.strategy_code IS '策略编码';
COMMENT ON COLUMN strategy_signal.strategy_version IS '策略版本';
COMMENT ON COLUMN strategy_signal.ts_code IS '股票代码';
COMMENT ON COLUMN strategy_signal.trade_date IS '信号日期';
COMMENT ON COLUMN strategy_signal.signal_type IS '信号类型';
COMMENT ON COLUMN strategy_signal.signal_strength IS '信号强度';
COMMENT ON COLUMN strategy_signal.signal_level IS '信号等级';
COMMENT ON COLUMN strategy_signal.buy_price_ref IS '买入参考价';
COMMENT ON COLUMN strategy_signal.stop_loss_ref IS '止损参考价';
COMMENT ON COLUMN strategy_signal.take_profit_ref IS '止盈参考价';
COMMENT ON COLUMN strategy_signal.invalidation_condition IS '失效条件';
COMMENT ON COLUMN strategy_signal.reason IS '信号原因说明';
COMMENT ON COLUMN strategy_signal.input_snapshot IS '输入快照';
COMMENT ON COLUMN strategy_signal.meta IS '补充信息';
COMMENT ON COLUMN strategy_signal.created_at IS '记录创建时间';

COMMENT ON TABLE job_run_log IS '任务执行日志表';
COMMENT ON COLUMN job_run_log.id IS '主键';
COMMENT ON COLUMN job_run_log.job_name IS '任务名称';
COMMENT ON COLUMN job_run_log.biz_date IS '业务日期';
COMMENT ON COLUMN job_run_log.status IS '任务状态';
COMMENT ON COLUMN job_run_log.started_at IS '开始时间';
COMMENT ON COLUMN job_run_log.finished_at IS '结束时间';
COMMENT ON COLUMN job_run_log.error_code IS '错误码';
COMMENT ON COLUMN job_run_log.error_message IS '错误信息';
COMMENT ON COLUMN job_run_log.retry_count IS '重试次数';
COMMENT ON COLUMN job_run_log.progress_current IS '当前进度';
COMMENT ON COLUMN job_run_log.progress_total IS '总进度';
COMMENT ON COLUMN job_run_log.meta IS '任务补充信息';
COMMENT ON COLUMN job_run_log.created_at IS '记录创建时间';

COMMENT ON TABLE app_user IS '系统预置账号表';
COMMENT ON COLUMN app_user.id IS '主键';
COMMENT ON COLUMN app_user.username IS '登录用户名，站内唯一';
COMMENT ON COLUMN app_user.display_name IS '展示名称';
COMMENT ON COLUMN app_user.password_hash IS '密码哈希，不保存明文';
COMMENT ON COLUMN app_user.status IS '账号状态，例如 active 或 disabled';
COMMENT ON COLUMN app_user.role IS '账号角色，例如 admin 或 user';
COMMENT ON COLUMN app_user.last_login_at IS '最近一次登录时间';
COMMENT ON COLUMN app_user.created_at IS '记录创建时间';
COMMENT ON COLUMN app_user.updated_at IS '记录更新时间';

COMMENT ON TABLE watchlist_group IS '用户自选分组表';
COMMENT ON COLUMN watchlist_group.id IS '主键';
COMMENT ON COLUMN watchlist_group.user_id IS '所属用户 ID';
COMMENT ON COLUMN watchlist_group.name IS '分组名称，同一用户下唯一';
COMMENT ON COLUMN watchlist_group.sort_order IS '排序值，越小越靠前';
COMMENT ON COLUMN watchlist_group.created_at IS '记录创建时间';
COMMENT ON COLUMN watchlist_group.updated_at IS '记录更新时间';

COMMENT ON TABLE watchlist_item IS '自选分组内股票表';
COMMENT ON COLUMN watchlist_item.id IS '主键';
COMMENT ON COLUMN watchlist_item.group_id IS '所属自选分组 ID';
COMMENT ON COLUMN watchlist_item.ts_code IS '股票代码';
COMMENT ON COLUMN watchlist_item.note IS '分组内备注';
COMMENT ON COLUMN watchlist_item.created_at IS '记录创建时间';

COMMENT ON TABLE user_position IS '用户持仓记录表';
COMMENT ON COLUMN user_position.id IS '主键';
COMMENT ON COLUMN user_position.user_id IS '所属用户 ID';
COMMENT ON COLUMN user_position.ts_code IS '股票代码';
COMMENT ON COLUMN user_position.position_date IS '持仓日期';
COMMENT ON COLUMN user_position.quantity IS '持仓数量';
COMMENT ON COLUMN user_position.cost_price IS '持仓成本价';
COMMENT ON COLUMN user_position.note IS '备注';
COMMENT ON COLUMN user_position.created_at IS '记录创建时间';
COMMENT ON COLUMN user_position.updated_at IS '记录更新时间';

COMMENT ON TABLE announcement IS '公告元数据表';
COMMENT ON COLUMN announcement.id IS '主键';
COMMENT ON COLUMN announcement.ts_code IS '股票代码';
COMMENT ON COLUMN announcement.title IS '公告标题';
COMMENT ON COLUMN announcement.announcement_type IS '公告类型';
COMMENT ON COLUMN announcement.publish_time IS '发布时间';
COMMENT ON COLUMN announcement.source IS '数据来源';
COMMENT ON COLUMN announcement.source_url IS '原始链接';
COMMENT ON COLUMN announcement.object_key IS '对象存储键';
COMMENT ON COLUMN announcement.content_hash IS '内容哈希';
COMMENT ON COLUMN announcement.created_at IS '记录创建时间';

COMMENT ON TABLE document_chunk IS '文档切片表';
COMMENT ON COLUMN document_chunk.id IS '主键';
COMMENT ON COLUMN document_chunk.document_type IS '文档类型';
COMMENT ON COLUMN document_chunk.document_id IS '文档主键 ID';
COMMENT ON COLUMN document_chunk.chunk_index IS '切片序号';
COMMENT ON COLUMN document_chunk.content IS '切片内容';
COMMENT ON COLUMN document_chunk.meta IS '切片补充信息';
COMMENT ON COLUMN document_chunk.created_at IS '记录创建时间';

COMMENT ON TABLE document_embedding IS '文档向量表';
COMMENT ON COLUMN document_embedding.chunk_id IS '切片主键 ID';
COMMENT ON COLUMN document_embedding.embedding IS '向量值';
COMMENT ON COLUMN document_embedding.embedding_model IS '向量模型名称';
COMMENT ON COLUMN document_embedding.created_at IS '记录创建时间';

COMMENT ON TABLE ai_stock_analysis IS 'AI 个股分析结果表';
COMMENT ON COLUMN ai_stock_analysis.id IS '主键';
COMMENT ON COLUMN ai_stock_analysis.ts_code IS '股票代码';
COMMENT ON COLUMN ai_stock_analysis.analysis_date IS '分析日期';
COMMENT ON COLUMN ai_stock_analysis.analysis_type IS '分析类型';
COMMENT ON COLUMN ai_stock_analysis.prompt_version IS '提示词版本';
COMMENT ON COLUMN ai_stock_analysis.conclusion IS '分析结论';
COMMENT ON COLUMN ai_stock_analysis.evidence IS '证据列表';
COMMENT ON COLUMN ai_stock_analysis.risks IS '风险列表';
COMMENT ON COLUMN ai_stock_analysis.created_at IS '记录创建时间';

CREATE INDEX idx_watchlist_group_user_id ON watchlist_group(user_id);
CREATE INDEX idx_watchlist_item_group_id ON watchlist_item(group_id);
CREATE INDEX idx_user_position_user_date ON user_position(user_id, position_date);

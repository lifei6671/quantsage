CREATE INDEX idx_stock_basic_symbol ON stock_basic(symbol);
CREATE INDEX idx_stock_basic_name_trgm ON stock_basic USING gin (name gin_trgm_ops);
CREATE INDEX idx_stock_daily_date ON stock_daily(trade_date);
CREATE INDEX idx_adj_factor_date ON adj_factor(trade_date);
CREATE INDEX idx_financial_indicator_period ON financial_indicator(report_period);
CREATE INDEX idx_stock_factor_daily_date ON stock_factor_daily(trade_date);
CREATE INDEX idx_stock_event_tag_code_date ON stock_event_tag(ts_code, trade_date);
CREATE INDEX idx_stock_event_tag_type_date ON stock_event_tag(event_type, trade_date);
CREATE INDEX idx_strategy_signal_date ON strategy_signal(trade_date, strategy_code, signal_type);
CREATE INDEX idx_job_run_log_name_date ON job_run_log(job_name, biz_date);
CREATE INDEX idx_announcement_code_time ON announcement(ts_code, publish_time DESC);
CREATE INDEX idx_document_embedding_vector ON document_embedding USING ivfflat (embedding vector_cosine_ops);

SELECT create_hypertable('stock_daily', 'trade_date', if_not_exists => TRUE);
SELECT create_hypertable('adj_factor', 'trade_date', if_not_exists => TRUE);
SELECT create_hypertable('stock_factor_daily', 'trade_date', if_not_exists => TRUE);

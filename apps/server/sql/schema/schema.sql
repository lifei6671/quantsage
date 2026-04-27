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

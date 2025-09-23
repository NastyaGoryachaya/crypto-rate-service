-- Таблица монет: натуральный ключ = symbol
CREATE TABLE IF NOT EXISTS coins (
    symbol TEXT PRIMARY KEY,        -- BTC, ETH
    name   TEXT NOT NULL DEFAULT ''
);

COMMENT ON TABLE coins IS 'Справочник криптовалют. Натуральный PK = symbol (BTC/ETH и т.п.)';
COMMENT ON COLUMN coins.symbol IS 'Уникальный символ монеты (натуральный ключ)';
COMMENT ON COLUMN coins.name   IS 'Название монеты';

-- Исторические цены: связь по символу
CREATE TABLE IF NOT EXISTS prices (
    coin_symbol TEXT NOT NULL REFERENCES coins(symbol) ON DELETE CASCADE,
    timestamp   TIMESTAMPTZ NOT NULL,         -- UTC
    value       NUMERIC(20,10) NOT NULL,
    CONSTRAINT value_positive CHECK (value > 0),
    PRIMARY KEY (coin_symbol, timestamp)
);

COMMENT ON TABLE prices IS 'Исторические цены криптовалют';
COMMENT ON COLUMN prices.coin_symbol IS 'Ссылка на coins(symbol)';
COMMENT ON COLUMN prices.timestamp   IS 'Момент фиксации цены (UTC)';
COMMENT ON COLUMN prices.value       IS 'Цена в NUMERIC(20,10)';

-- Индексы под частые запросы
CREATE INDEX IF NOT EXISTS idx_prices_symbol_ts_desc
    ON prices (coin_symbol, timestamp DESC);
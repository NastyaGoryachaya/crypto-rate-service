INSERT INTO coins (symbol, name) VALUES
    ('BTC', 'Bitcoin'),
    ('ETH', 'Ethereum')
ON CONFLICT (symbol) DO NOTHING;
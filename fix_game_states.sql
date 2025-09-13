-- 创建game_states表
CREATE TABLE IF NOT EXISTS game_states (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL UNIQUE,
    current_state TEXT NOT NULL DEFAULT 'idle',
    game_data TEXT,
    credits INTEGER DEFAULT 0,
    total_win INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_game_states_session_id ON game_states(session_id);
CREATE INDEX IF NOT EXISTS idx_game_states_updated_at ON game_states(updated_at);
CREATE INDEX IF NOT EXISTS idx_game_states_current_state ON game_states(current_state);
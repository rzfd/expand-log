CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('income', 'expense')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_categories_user_id ON categories(user_id);
CREATE INDEX idx_categories_user_type ON categories(user_id, type);

CREATE TABLE recurring_transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    type TEXT NOT NULL CHECK (type IN ('income', 'expense')),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    note TEXT NOT NULL DEFAULT '',
    frequency TEXT NOT NULL CHECK (frequency IN ('daily', 'weekly', 'monthly')),
    start_date DATE NOT NULL,
    end_date DATE NULL,
    next_run_date DATE NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recurring_transactions_user_id ON recurring_transactions(user_id);
CREATE INDEX idx_recurring_transactions_due ON recurring_transactions(active, next_run_date);

CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    recurring_transaction_id BIGINT NULL REFERENCES recurring_transactions(id) ON DELETE SET NULL,
    type TEXT NOT NULL CHECK (type IN ('income', 'expense')),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    note TEXT NOT NULL DEFAULT '',
    transaction_date DATE NOT NULL,
    source TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'recurring')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_user_date ON transactions(user_id, transaction_date DESC);
CREATE INDEX idx_transactions_user_type ON transactions(user_id, type);
CREATE INDEX idx_transactions_user_category ON transactions(user_id, category_id);
CREATE INDEX idx_transactions_user_amount ON transactions(user_id, amount_cents);
CREATE UNIQUE INDEX uniq_transactions_recurring_generated
    ON transactions(recurring_transaction_id, transaction_date);

CREATE TABLE budgets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    year INT NOT NULL CHECK (year >= 2000),
    month INT NOT NULL CHECK (month BETWEEN 1 AND 12),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, category_id, year, month)
);

CREATE INDEX idx_budgets_user_period ON budgets(user_id, year, month);

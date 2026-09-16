-- +goose Up
-- Core schema: users, saving goals, deposits, per-user scenario state.

CREATE TABLE users (
    id            bigint PRIMARY KEY, -- Telegram user id
    username      text NOT NULL DEFAULT '',
    first_name    text NOT NULL DEFAULT '',
    language_code text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE goals (
    id               bigserial PRIMARY KEY,
    user_id          bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title            text NOT NULL,
    deadline         timestamptz NOT NULL,
    timezone         text NOT NULL DEFAULT 'UTC', -- IANA name, e.g. 'Europe/Moscow'
    currency         text NOT NULL DEFAULT 'RUB', -- ISO-4217, currency of the targets
    savings_currency text NOT NULL DEFAULT 'RUB', -- ISO-4217, currency of the deposits
    target_min       bigint NOT NULL,             -- minor units (kopecks), never float
    target_ok        bigint NOT NULL,
    target_max       bigint NOT NULL,
    active           boolean NOT NULL DEFAULT true,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX goals_user_id_idx ON goals (user_id);

CREATE TABLE deposits (
    id          bigserial PRIMARY KEY,
    goal_id     bigint NOT NULL REFERENCES goals (id) ON DELETE CASCADE,
    amount      bigint NOT NULL, -- signed: + put aside, - taken back
    happened_at timestamptz NOT NULL DEFAULT now(),
    note        text NOT NULL DEFAULT ''
);
CREATE INDEX deposits_goal_id_idx ON deposits (goal_id, happened_at);

-- One row per user inside a scenario dialogue. Stored in the database so a bot
-- restart does not lose the dialogue position.
CREATE TABLE user_scenario_state (
    user_id       bigint PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    scenario_name text NOT NULL,
    node_id       text NOT NULL,
    entered_at    timestamptz NOT NULL DEFAULT now(),
    vars          jsonb NOT NULL DEFAULT '{}'::jsonb
);

-- +goose Down
DROP TABLE user_scenario_state;
DROP TABLE deposits;
DROP TABLE goals;
DROP TABLE users;

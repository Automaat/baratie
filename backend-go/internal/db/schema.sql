-- Baseline schema for Baratie. Applied by db.ApplySchema to an empty database
-- on first start (presence-checked on the `recipes` table). The `users` table
-- is created separately by auth.Store.EnsureSchema (additive, idempotent DDL
-- that also runs against existing databases).

CREATE TABLE recipes (
    id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name varchar(200) NOT NULL,
    description text NOT NULL DEFAULT '',
    instructions text NOT NULL DEFAULT '',
    ingredients text[] NOT NULL DEFAULT '{}',
    tags text[] NOT NULL DEFAULT '{}',
    servings integer NOT NULL DEFAULT 1,
    prep_minutes integer NOT NULL DEFAULT 0,
    cook_minutes integer NOT NULL DEFAULT 0,
    calories_kcal double precision NOT NULL DEFAULT 0,
    protein_g double precision NOT NULL DEFAULT 0,
    carbs_g double precision NOT NULL DEFAULT 0,
    fat_g double precision NOT NULL DEFAULT 0,
    created_at timestamp without time zone NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);

CREATE TABLE pantry_items (
    id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name varchar(200) NOT NULL,
    quantity double precision NOT NULL DEFAULT 0,
    unit varchar(50) NOT NULL DEFAULT '',
    category varchar(100) NOT NULL DEFAULT 'other',
    expires_on date,
    created_at timestamp without time zone NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);

CREATE TABLE meal_plan_entries (
    id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    plan_date date NOT NULL,
    meal_type varchar(20) NOT NULL,
    recipe_id integer REFERENCES recipes (id) ON DELETE SET NULL,
    note varchar(300) NOT NULL DEFAULT '',
    created_at timestamp without time zone NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);

CREATE INDEX idx_meal_plan_date ON meal_plan_entries (plan_date);
CREATE INDEX idx_pantry_category ON pantry_items (category);

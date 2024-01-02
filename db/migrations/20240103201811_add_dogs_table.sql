-- +goose Up
-- +goose StatementBegin
CREATE TABLE cats (
    cat_id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    age INTEGER NOT NULL,
    description TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE cats;
-- +goose StatementEnd

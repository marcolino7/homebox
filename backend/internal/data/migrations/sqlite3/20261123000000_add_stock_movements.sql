-- +goose Up
CREATE TABLE stock_movements (
    id             uuid     not null primary key,
    created_at     datetime not null,
    updated_at     datetime not null,
    movement_type  text     not null,
    quantity       real     not null,
    quantity_after real,
    occurred_at    datetime not null,
    comment        text,
    performed_by   text,
    entity_id      uuid     not null constraint stock_movements_entities_stock_movements references entities on delete cascade
);
CREATE INDEX stockmovement_entity_id_occurred_at ON stock_movements (entity_id, occurred_at);

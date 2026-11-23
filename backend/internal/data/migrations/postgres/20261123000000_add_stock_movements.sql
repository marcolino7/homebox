-- +goose Up
CREATE TABLE stock_movements (
    id             uuid             not null primary key,
    created_at     timestamptz      not null,
    updated_at     timestamptz      not null,
    movement_type  varchar(20)      not null,
    quantity       double precision not null,
    quantity_after double precision,
    occurred_at    timestamptz      not null,
    comment        varchar(500),
    performed_by   varchar(255),
    entity_id      uuid             not null constraint stock_movements_entities_stock_movements references entities on delete cascade
);
CREATE INDEX stockmovement_entity_id_occurred_at ON stock_movements (entity_id, occurred_at);

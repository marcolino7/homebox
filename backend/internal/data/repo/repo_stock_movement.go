package repo

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/entity"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/group"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/stockmovement"
)

// Movement types stored in a StockMovement.
const (
	StockMovementIn         = "in"
	StockMovementOut        = "out"
	StockMovementAdjustment = "adjustment"
	StockMovementInitial    = "initial"
)

// ErrInvalidStockMovement is returned when a movement request is not valid.
var ErrInvalidStockMovement = errors.New("invalid stock movement")

// StockMovementRepository keeps the quantity ledger of the entities.
type StockMovementRepository struct {
	db *ent.Client
}

// StockMovementCreate is the payload used to record a movement.
//
// By default the movement is applied to the entity quantity:
//   - in:         Quantity (> 0) is added
//   - out:        Quantity (> 0) is removed, the result can't go below zero
//   - adjustment: Quantity (>= 0) is the new quantity of the entity
//
// With ApplyToQuantity=false the entity quantity is left untouched and the
// entry is stored as is, which is meant to import an existing history: then
// Quantity is the signed change and QuantityAfter, PerformedBy and OccurredAt
// are taken from the payload.
type StockMovementCreate struct {
	Type            string     `json:"type"                      validate:"required,oneof=in out adjustment initial"`
	Quantity        float64    `json:"quantity"`
	Comment         string     `json:"comment"                   validate:"max=500"`
	OccurredAt      *time.Time `json:"occurredAt,omitempty"      extensions:"x-nullable,x-omitempty"`
	ApplyToQuantity *bool      `json:"applyToQuantity,omitempty" extensions:"x-nullable,x-omitempty"`
	QuantityAfter   *float64   `json:"quantityAfter,omitempty"   extensions:"x-nullable,x-omitempty"`
	PerformedBy     string     `json:"performedBy"               validate:"max=255"`
}

// StockMovement is one entry of the quantity ledger.
type StockMovement struct {
	ID            uuid.UUID `json:"id"`
	Type          string    `json:"type"`
	Quantity      float64   `json:"quantity"`
	QuantityAfter *float64  `json:"quantityAfter,omitempty" extensions:"x-nullable,x-omitempty"`
	OccurredAt    time.Time `json:"occurredAt"`
	Comment       string    `json:"comment"`
	PerformedBy   string    `json:"performedBy"`
	CreatedAt     time.Time `json:"createdAt"`
}

func mapStockMovement(m *ent.StockMovement) StockMovement {
	return StockMovement{
		ID:            m.ID,
		Type:          m.MovementType,
		Quantity:      m.Quantity,
		QuantityAfter: m.QuantityAfter,
		OccurredAt:    m.OccurredAt,
		Comment:       m.Comment,
		PerformedBy:   m.PerformedBy,
		CreatedAt:     m.CreatedAt,
	}
}

func (r *StockMovementRepository) owned(ctx context.Context, gid, entityID uuid.UUID) error {
	ok, err := r.db.Entity.Query().
		Where(entity.ID(entityID), entity.HasGroupWith(group.ID(gid))).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return &ent.NotFoundError{}
	}
	return nil
}

// List returns the movements of an entity, newest first.
func (r *StockMovementRepository) List(ctx context.Context, gid, entityID uuid.UUID) ([]StockMovement, error) {
	if err := r.owned(ctx, gid, entityID); err != nil {
		return nil, err
	}
	rows, err := r.db.StockMovement.Query().
		Where(stockmovement.EntityID(entityID)).
		Order(ent.Desc(stockmovement.FieldOccurredAt), ent.Desc(stockmovement.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]StockMovement, 0, len(rows))
	for _, m := range rows {
		out = append(out, mapStockMovement(m))
	}
	return out, nil
}

// Quantity returns the current quantity of an entity of the group.
func (r *StockMovementRepository) Quantity(ctx context.Context, gid, entityID uuid.UUID) (float64, error) {
	e, err := r.db.Entity.Query().
		Where(entity.ID(entityID), entity.HasGroupWith(group.ID(gid))).
		Select(entity.FieldQuantity).
		Only(ctx)
	if err != nil {
		return 0, err
	}
	return e.Quantity, nil
}

// Create records a movement, applying it to the entity quantity unless the
// payload asks to only store it (see StockMovementCreate).
func (r *StockMovementRepository) Create(ctx context.Context, gid, entityID uuid.UUID, performedBy string, in StockMovementCreate) (StockMovement, error) {
	if err := r.owned(ctx, gid, entityID); err != nil {
		return StockMovement{}, err
	}

	occurred := time.Now()
	if in.OccurredAt != nil && !in.OccurredAt.IsZero() {
		occurred = *in.OccurredAt
	}

	apply := in.ApplyToQuantity == nil || *in.ApplyToQuantity
	if !apply {
		by := in.PerformedBy
		if by == "" {
			by = performedBy
		}
		m, err := r.db.StockMovement.Create().
			SetEntityID(entityID).
			SetMovementType(in.Type).
			SetQuantity(in.Quantity).
			SetNillableQuantityAfter(in.QuantityAfter).
			SetOccurredAt(occurred).
			SetComment(in.Comment).
			SetPerformedBy(by).
			Save(ctx)
		if err != nil {
			return StockMovement{}, err
		}
		return mapStockMovement(m), nil
	}

	tx, err := r.db.Tx(ctx)
	if err != nil {
		return StockMovement{}, err
	}
	defer func() { _ = tx.Rollback() }()

	cur, err := tx.Entity.Query().Where(entity.ID(entityID)).Select(entity.FieldQuantity).Only(ctx)
	if err != nil {
		return StockMovement{}, err
	}

	var next float64
	switch in.Type {
	case StockMovementIn:
		if in.Quantity <= 0 {
			return StockMovement{}, fmt.Errorf("%w: quantity must be greater than zero", ErrInvalidStockMovement)
		}
		next = cur.Quantity + in.Quantity
	case StockMovementOut:
		if in.Quantity <= 0 {
			return StockMovement{}, fmt.Errorf("%w: quantity must be greater than zero", ErrInvalidStockMovement)
		}
		next = cur.Quantity - in.Quantity
		if next < 0 {
			return StockMovement{}, fmt.Errorf("%w: only %v available", ErrInvalidStockMovement, cur.Quantity)
		}
	case StockMovementAdjustment:
		if in.Quantity < 0 {
			return StockMovement{}, fmt.Errorf("%w: quantity can't be negative", ErrInvalidStockMovement)
		}
		next = in.Quantity
	default:
		return StockMovement{}, fmt.Errorf("%w: type %q can't be applied", ErrInvalidStockMovement, in.Type)
	}

	if err := tx.Entity.UpdateOneID(entityID).SetQuantity(next).Exec(ctx); err != nil {
		return StockMovement{}, err
	}
	m, err := tx.StockMovement.Create().
		SetEntityID(entityID).
		SetMovementType(in.Type).
		SetQuantity(next - cur.Quantity).
		SetQuantityAfter(next).
		SetOccurredAt(occurred).
		SetComment(in.Comment).
		SetPerformedBy(performedBy).
		Save(ctx)
	if err != nil {
		return StockMovement{}, err
	}
	if err := tx.Commit(); err != nil {
		return StockMovement{}, err
	}
	return mapStockMovement(m), nil
}

// RecordChange stores a movement for a quantity that was already changed by
// another operation (for example the +/- buttons or the edit form). It does
// nothing when the quantity did not change.
func (r *StockMovementRepository) RecordChange(ctx context.Context, entityID uuid.UUID, before, after float64, movementType, performedBy, comment string) error {
	if math.Abs(after-before) < 1e-9 {
		return nil
	}
	return r.db.StockMovement.Create().
		SetEntityID(entityID).
		SetMovementType(movementType).
		SetQuantity(after - before).
		SetQuantityAfter(after).
		SetOccurredAt(time.Now()).
		SetComment(comment).
		SetPerformedBy(performedBy).
		Exec(ctx)
}

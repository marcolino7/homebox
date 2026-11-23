package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/schema/mixins"
)

// StockMovement is one entry of the quantity ledger of an entity: stock in,
// stock out, a manual adjustment or the initial quantity.
type StockMovement struct {
	ent.Schema
}

func (StockMovement) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.BaseMixin{},
	}
}

func (StockMovement) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("entity_id", uuid.UUID{}),
		// One of: in, out, adjustment, initial.
		field.String("movement_type").
			MaxLen(20).
			NotEmpty(),
		// Signed change of the quantity (positive for in, negative for out).
		field.Float("quantity"),
		// Quantity of the entity right after this movement, when known.
		field.Float("quantity_after").
			Optional().
			Nillable(),
		// When the movement happened. Can differ from created_at for imported history.
		field.Time("occurred_at").
			Default(time.Now),
		field.String("comment").
			MaxLen(500).
			Optional(),
		field.String("performed_by").
			MaxLen(255).
			Optional(),
	}
}

func (StockMovement) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("entity", Entity.Type).
			Field("entity_id").
			Ref("stock_movements").
			Required().
			Unique(),
	}
}

func (StockMovement) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("entity_id", "occurred_at"),
	}
}

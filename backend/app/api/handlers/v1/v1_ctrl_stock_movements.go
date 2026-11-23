package v1

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/hay-kot/httpkit/errchain"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
	"github.com/sysadminsmedia/homebox/backend/internal/web/adapters"
)

// HandleStockMovementsGet godoc
//
//	@Summary	Get Stock Movements
//	@Tags		Item Stock Movements
//	@Produce	json
//	@Param		id	path	string	true	"Item ID"
//	@Success	200	{array}	repo.StockMovement
//	@Router		/v1/entities/{id}/stock-movements [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandleStockMovementsGet() errchain.HandlerFunc {
	fn := func(r *http.Request, ID uuid.UUID) ([]repo.StockMovement, error) {
		auth := services.NewContext(r.Context())
		return ctrl.repo.StockMovements.List(auth, auth.GID, ID)
	}

	return adapters.CommandID("id", fn, http.StatusOK)
}

// HandleStockMovementCreate godoc
//
//	@Summary	Create Stock Movement
//	@Tags		Item Stock Movements
//	@Produce	json
//	@Param		id		path		string						true	"Item ID"
//	@Param		payload	body		repo.StockMovementCreate	true	"Movement Data"
//	@Success	201		{object}	repo.StockMovement
//	@Router		/v1/entities/{id}/stock-movements [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandleStockMovementCreate() errchain.HandlerFunc {
	fn := func(r *http.Request, itemID uuid.UUID, body repo.StockMovementCreate) (repo.StockMovement, error) {
		auth := services.NewContext(r.Context())
		out, err := ctrl.repo.StockMovements.Create(auth, auth.GID, itemID, userDisplayName(auth), body)
		if errors.Is(err, repo.ErrInvalidStockMovement) {
			return repo.StockMovement{}, validate.NewRequestError(err, http.StatusUnprocessableEntity)
		}
		return out, err
	}

	return adapters.ActionID("id", fn, http.StatusCreated)
}

// userDisplayName returns the name to store as author of a stock movement.
func userDisplayName(auth services.Context) string {
	if auth.User != nil {
		return auth.User.Name
	}
	return ""
}

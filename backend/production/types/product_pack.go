package types

import (
	"app/core"
	"app/db"
	"fmt"
	"strings"
)

// Pack price types (docs/PACKS_PLAN.md §3.1).
const (
	PackPriceFixed      = int8(1) // PriceValue is the pack total, in PEN cents
	PackPricePercentOff = int8(2) // PriceValue is basis points off the components' sum (1500 = 15%)
	PackPriceAmountOff  = int8(3) // PriceValue is PEN cents off the components' sum
)

// Pack rounding modes: applied to the pack total, to the whole sol.
const (
	PackRoundingNone    = int8(0)
	PackRoundingUp      = int8(1)
	PackRoundingNearest = int8(2)
	PackRoundingDown    = int8(3)
)

// ProductPack is a bundle priced as a whole: "1 × [cat] Entrada + 1 × [cat] Plato Fondo 1 = S/ 18"
// (docs/PACKS_PLAN.md). The sales engine finds packs in an order automatically and prorates the
// pack price into the component lines.
type ProductPack struct {
	db.TableStruct[ProductPackTable, ProductPack]
	CompanyID int32  `cb:"1"`
	ID        int32  `cb:"2"`
	Name      string `cb:"3"`
	// Components is one colbin blob: a pack is always read whole.
	Components   []ProductPackComponent `cb:"4"`
	PriceType    int8                   `cb:"5"`
	PriceValue   int32                  `cb:"6"`
	RoundingMode int8                   `cb:"7"`
	Status       int8                   `json:"ss" cb:"ss,8"`
	Created      int32                  `cb:"9"`
	CreatedBy    int32                  `cb:"10"`
	Updated      int32                  `json:"upd" cb:"upd,11"`
	UpdatedBy    int32                  `cb:"12"`
	Version      int16                  `json:"ver" cb:"ver,13"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,14"`
}

type ProductPackTable struct {
	db.TableStruct[ProductPackTable, ProductPack]
	CompanyID    db.Col[*ProductPackTable, int32]
	ID           db.Col[*ProductPackTable, int32]
	Name         db.Col[*ProductPackTable, string]
	Components   db.Col[*ProductPackTable, []ProductPackComponent]
	PriceType    db.Col[*ProductPackTable, int8]
	PriceValue   db.Col[*ProductPackTable, int32]
	RoundingMode db.Col[*ProductPackTable, int8]
	Status       db.Col[*ProductPackTable, int8]
	Created      db.Col[*ProductPackTable, int32]
	CreatedBy    db.Col[*ProductPackTable, int32]
	Updated      db.Col[*ProductPackTable, int32]
	UpdatedBy    db.Col[*ProductPackTable, int32]
	Version      db.Col[*ProductPackTable, int16]
}

func (e ProductPackTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          126,
		Name:        "product_pack",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Min: 0, Max: 1}},
		Indexes: []db.Index{
			// The company's packs, delta-synced by the packs page and the sales screens.
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// ProductPackComponent is one slot of a pack: Quantity units of a product, or of any product in a
// category. Exactly one of ProductID / CategoryID is set.
type ProductPackComponent struct {
	ProductID int32 `cb:"1"`
	// CategoryID is a record of the shared list SharedListCategoryID.
	CategoryID int32 `cb:"2"`
	Quantity   int16 `cb:"3"`
}

// ValidateProductPack refuses a pack the sales engine could not apply: no name, a malformed slot, a
// repeated slot, an unknown price rule, or fewer than two units (a one-unit "pack" is a price change).
// It checks the shape only; the caller checks that the products and categories exist.
func ValidateProductPack(pack ProductPack) error {
	if len(strings.TrimSpace(pack.Name)) < 2 {
		return core.Err("El nombre del pack debe tener al menos 2 caracteres.")
	}
	if len(pack.Components) == 0 {
		return core.Err("El pack debe tener al menos un componente.")
	}
	totalUnits := 0
	seenProductIDs, seenCategoryIDs := map[int32]bool{}, map[int32]bool{}
	for componentIndex, component := range pack.Components {
		position := componentIndex + 1
		if (component.ProductID > 0) == (component.CategoryID > 0) {
			return core.Err(fmt.Sprintf("El componente %v del pack debe ser un producto o una categoría.", position))
		}
		if component.Quantity < 1 {
			return core.Err(fmt.Sprintf("El componente %v del pack debe tener una cantidad mayor a cero.", position))
		}
		// A repeated slot is the same slot with a larger quantity: one row keeps the pack readable.
		if component.ProductID > 0 {
			if seenProductIDs[component.ProductID] {
				return core.Err(fmt.Sprintf("El producto %v está repetido en el pack; sume su cantidad.", component.ProductID))
			}
			seenProductIDs[component.ProductID] = true
		} else {
			if seenCategoryIDs[component.CategoryID] {
				return core.Err(fmt.Sprintf("La categoría %v está repetida en el pack; sume su cantidad.", component.CategoryID))
			}
			seenCategoryIDs[component.CategoryID] = true
		}
		totalUnits += int(component.Quantity)
	}
	if totalUnits < 2 {
		return core.Err("El pack debe sumar al menos 2 unidades.")
	}

	switch pack.PriceType {
	case PackPriceFixed, PackPriceAmountOff:
		if pack.PriceValue <= 0 {
			return core.Err("El monto del pack debe ser mayor a cero.")
		}
	case PackPricePercentOff:
		if pack.PriceValue <= 0 || pack.PriceValue >= 10000 {
			return core.Err("El porcentaje de descuento debe estar entre 0% y 100%.")
		}
	default:
		return core.Err("El tipo de precio del pack no es válido.")
	}
	if pack.RoundingMode < PackRoundingNone || pack.RoundingMode > PackRoundingDown {
		return core.Err("El redondeo del pack no es válido.")
	}
	return nil
}

package production

import (
	business "app/business/types"
	"app/core"
	"app/db"
	"app/production/types"
	"encoding/json"
	"fmt"
	"strings"
)

func GetProductPacks(req *core.HandlerArgs) core.HandlerResponse {
	packs := []types.ProductPack{}
	query := db.Query(&packs)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), int8(1))
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los packs:", err)
	}
	return req.MakeResponse(packs)
}

// PostProductPack creates or updates one pack; ss = 0 deletes it. Sales already priced by it keep
// their line prices.
func PostProductPack(req *core.HandlerArgs) core.HandlerResponse {
	pack := types.ProductPack{}
	if err := json.Unmarshal([]byte(*req.Body), &pack); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	pack.Name = strings.TrimSpace(pack.Name)
	if pack.Status != 0 {
		pack.Status = 1
		// A fixed price is already the chosen total: there is nothing to round.
		if pack.PriceType == types.PackPriceFixed {
			pack.RoundingMode = types.PackRoundingNone
		}
		if err := types.ValidateProductPack(pack); err != nil {
			return req.MakeErr(err)
		}
		if err := validatePackComponentsExist(companyID, pack.Components); err != nil {
			return req.MakeErr(err)
		}
	}
	pack.CompanyID, pack.Updated, pack.UpdatedBy = companyID, core.SUnixTime(), req.User.ID

	records := []types.ProductPack{pack}
	if pack.ID <= 0 {
		if pack.Status == 0 {
			return req.MakeErr("No se puede eliminar un pack que no se ha guardado.")
		}
		records[0].ID, records[0].Created, records[0].CreatedBy = 0, pack.Updated, req.User.ID
		if err := db.Insert(&records); err != nil {
			return req.MakeErr("Error al guardar el pack:", err)
		}
		return req.MakeResponse(records[0])
	}
	stored := []types.ProductPack{}
	query := db.Query(&stored)
	query.Select(query.ID).CompanyID.Equals(companyID).ID.Equals(pack.ID)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener el pack:", err)
	}
	if len(stored) == 0 {
		return req.MakeErr("El pack no existe.")
	}
	table := db.TableOf[types.ProductPack]()
	if err := db.Update(&records, table.Name, table.Components, table.PriceType, table.PriceValue,
		table.RoundingMode, table.Status, table.Updated, table.UpdatedBy); err != nil {
		return req.MakeErr("Error al guardar el pack:", err)
	}
	return req.MakeResponse(records[0])
}

// validatePackComponentsExist refuses a slot naming a product or category the company does not
// have active: the sales engine could never fill it.
func validatePackComponentsExist(companyID int32, components []types.ProductPackComponent) error {
	productIDs, categoryIDs := []int32{}, []int32{}
	for _, component := range components {
		if component.ProductID > 0 {
			productIDs = append(productIDs, component.ProductID)
		} else {
			categoryIDs = append(categoryIDs, component.CategoryID)
		}
	}

	productsByID, err := readProductsByID(companyID, productIDs, func(query *types.ProductTable) []db.Coln {
		return db.Cols(query.ID, query.Status)
	})
	if err != nil {
		return err
	}
	for _, productID := range productIDs {
		if productsByID[productID].Status <= 0 {
			return core.Err(fmt.Sprintf("El producto %v del pack no existe o no está activo.", productID))
		}
	}

	if len(categoryIDs) == 0 {
		return nil
	}
	categories := []business.SharedListRecord{}
	query := db.Query(&categories)
	query.Select(query.ID, query.ListID, query.Status).CompanyID.Equals(companyID).
		ID.In(core.MakeSliceInclude(categoryIDs).Values...)
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener las categorías del pack:", err)
	}
	activeCategoryIDs := map[int32]bool{}
	for _, category := range categories {
		if category.ListID == business.SharedListCategoryID && category.Status > 0 {
			activeCategoryIDs[category.ID] = true
		}
	}
	for _, categoryID := range categoryIDs {
		if !activeCategoryIDs[categoryID] {
			return core.Err(fmt.Sprintf("La categoría %v del pack no existe o no está activa.", categoryID))
		}
	}
	return nil
}

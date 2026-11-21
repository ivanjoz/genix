package dining

import (
	business "app/business/types"
	"app/core"
	"app/db"
	"app/dining/types"
	production "app/production/types"
	"encoding/json"
	"fmt"
	"strings"
)

// GetDiningMenus is the delta of the company's active cartas; a deleted one arrives as a removed ID.
func GetDiningMenus(req *core.HandlerArgs) core.HandlerResponse {
	menus := []types.DiningMenu{}
	query := db.Query(&menus)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor("records"), int8(1))
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las cartas:", err)
	}
	return req.MakeResponse(map[string]any{"records": &menus})
}

// PostDiningMenu creates, updates or deletes (ss = 0) one carta (docs/DINING_MENU_PLAN.md §1.3).
func PostDiningMenu(req *core.HandlerArgs) core.HandlerResponse {
	menu := types.DiningMenu{}
	if err := json.Unmarshal([]byte(*req.Body), &menu); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	menu.CompanyID = companyID
	menu.UpdatedBy = req.User.ID
	menu.Updated = core.SUnixTime()

	err := db.RunTransaction(func(transaction *db.Transaction) error {
		var stored *types.DiningMenu
		if menu.ID > 0 {
			storedMenus := []types.DiningMenu{{CompanyID: companyID, ID: menu.ID}}
			if err := db.LockByKeys(transaction, &storedMenus); err != nil {
				return core.Err("Error al obtener la carta:", err)
			}
			if len(storedMenus) == 0 || storedMenus[0].Status == 0 {
				return core.Err("La carta no existe.")
			}
			stored = &storedMenus[0]
		} else if menu.Status == 0 {
			return core.Err("No se puede eliminar una carta que no se ha guardado.")
		}

		if menu.Status != 0 {
			menu.Status = 1
			types.NormalizeDiningMenu(&menu)
			if err := validateMenuReferences(companyID, menu); err != nil {
				return err
			}
		} else {
			// A delete keeps the carta as it was: only its status changes.
			if err := validateMenuNotAssigned(companyID, menu.ID); err != nil {
				return err
			}
			deletedMenu := *stored
			deletedMenu.Status, deletedMenu.Updated, deletedMenu.UpdatedBy = 0, menu.Updated, menu.UpdatedBy
			menu = deletedMenu
		}

		menus := []types.DiningMenu{menu}
		if stored == nil {
			if err := db.InsertIn(transaction, &menus); err != nil {
				return core.Err("Error al registrar la carta:", err)
			}
		} else {
			menuTable := db.TableOf[types.DiningMenu]()
			if err := db.UpdateIn(transaction, &menus,
				menuTable.Name, menuTable.Theme, menuTable.Blocks, menuTable.Status, menuTable.Updated, menuTable.UpdatedBy,
			); err != nil {
				return core.Err("Error al actualizar la carta:", err)
			}
		}
		menu = menus[0]
		return nil
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(menu)
}

// validateMenuReferences reads the categories and products the carta names and runs the pure
// validation over them.
func validateMenuReferences(companyID int32, menu types.DiningMenu) error {
	categoryIDs, productIDs := types.MenuReferencedIDs(menu)

	activeCategoryIDs := map[int32]bool{}
	if len(categoryIDs) > 0 {
		categories := []business.SharedListRecord{}
		query := db.Query(&categories)
		query.Select(query.ID, query.ListID, query.Status).CompanyID.Equals(companyID).
			ID.In(core.MakeSliceInclude(categoryIDs).Values...)
		if err := query.Exec(); err != nil {
			return core.Err("Error al obtener las categorías de la carta:", err)
		}
		for _, category := range categories {
			if category.ListID == business.SharedListCategoryID && category.Status > 0 {
				activeCategoryIDs[category.ID] = true
			}
		}
	}

	activeProductIDs := map[int32]bool{}
	if len(productIDs) > 0 {
		products := []production.Product{}
		query := db.Query(&products)
		query.Select(query.ID, query.Status).CompanyID.Equals(companyID).
			ID.In(core.MakeSliceInclude(productIDs).Values...)
		if err := query.Exec(); err != nil {
			return core.Err("Error al obtener los productos de la carta:", err)
		}
		for _, product := range products {
			if product.Status == production.ProductStatusActive {
				activeProductIDs[product.ID] = true
			}
		}
	}
	return types.ValidateDiningMenu(menu, activeCategoryIDs, activeProductIDs)
}

// validateMenuNotAssigned refuses deleting a carta that an active layout shows, naming the layouts.
func validateMenuNotAssigned(companyID int32, menuID int32) error {
	layouts := []types.DiningLayout{}
	query := db.Query(&layouts)
	query.Select(query.ID, query.Name, query.MenuID, query.Status).CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener los salones:", err)
	}
	assignedNames := []string{}
	for _, layout := range layouts {
		if layout.Status != 0 && layout.MenuID == menuID {
			assignedNames = append(assignedNames, fmt.Sprintf(`"%s"`, layout.Name))
		}
	}
	if len(assignedNames) > 0 {
		return core.Err("La carta está en uso por los salones " + strings.Join(assignedNames, ", ") + ": cambie su carta antes de eliminarla.")
	}
	return nil
}

// validateLayoutMenu checks a layout's carta is 0 (the default) or an active carta of the company.
func validateLayoutMenu(companyID int32, menuID int32) error {
	if menuID == 0 {
		return nil
	}
	menus := []types.DiningMenu{}
	query := db.Query(&menus)
	query.Select(query.ID, query.Status).CompanyID.Equals(companyID).ID.Equals(menuID)
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener la carta del salón:", err)
	}
	if len(menus) == 0 || menus[0].Status == 0 {
		return core.Err("La carta del salón no existe o fue eliminada.")
	}
	return nil
}

package dining

import (
	"app/core"
	"app/db"
	diningModule "app/dining"
	diningTypes "app/dining/types"
	"app/seeds/runner"
)

const cartaName = "Carta principal"

// ensureDiningCarta saves the company's seeded carta (docs/DINING_MENU_PLAN.md §5) unless one with
// that name exists, and returns its ID: a title, the three categories in their own styles and the
// dishes no other block lists.
func ensureDiningCarta(menu diningMenu) (int32, error) {
	storedMenus := []diningTypes.DiningMenu{}
	query := db.Query(&storedMenus)
	query.Select().CompanyID.Equals(runner.CompanyID).Status.Equals(1)
	if err := query.Exec(); err != nil {
		return 0, core.Err("no se pudieron leer las cartas:", err)
	}
	for _, storedMenu := range storedMenus {
		if storedMenu.Name == cartaName {
			return storedMenu.ID, nil
		}
	}

	carta := diningTypes.DiningMenu{Name: cartaName, Status: 1, Blocks: []diningTypes.DiningMenuBlock{
		{Kind: diningTypes.MenuBlockTitle, Text: "Nuestra carta", Align: diningTypes.MenuAlignCenter},
		{Kind: diningTypes.MenuBlockText, Text: "Cocina criolla preparada al momento.", Align: diningTypes.MenuAlignCenter},
		{Kind: diningTypes.MenuBlockCategory, CategoryID: menu.categoryIDByName[categoryEntradas], Variant: diningTypes.MenuProductsCards},
		{Kind: diningTypes.MenuBlockDivider},
		{Kind: diningTypes.MenuBlockCategory, CategoryID: menu.categoryIDByName[categoryFondos], Variant: diningTypes.MenuProductsWithPhoto},
		{Kind: diningTypes.MenuBlockDivider},
		{Kind: diningTypes.MenuBlockSubtitle, Text: "Para cerrar", Align: diningTypes.MenuAlignLeft},
		{Kind: diningTypes.MenuBlockCategory, CategoryID: menu.categoryIDByName[categoryPostres], Variant: diningTypes.MenuProductsCompact},
		{Kind: diningTypes.MenuBlockAllProducts, Variant: diningTypes.MenuProductsCompact, RestOnly: true},
	}}
	savedCarta := diningTypes.DiningMenu{}
	if err := callAsDiningUser("POST.dining-menu", diningModule.PostDiningMenu, carta, &savedCarta); err != nil {
		return 0, err
	}
	core.Log("SeedDining:: carta guardada", savedCarta.ID, savedCarta.Name, "bloques", len(savedCarta.Blocks))
	return savedCarta.ID, nil
}

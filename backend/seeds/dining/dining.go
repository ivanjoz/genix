// Package dining seeds a restaurant into the demo company: a menu of 20 platos de fondo, 4 entradas
// and 3 postres (made products), the packs that price them as menus and combos, two dining rooms
// with their prep stations, and the table service of yesterday and today, charged through the same
// handlers the floor-service page uses.
//
// The dining menu lists every active product the web store sells, so the retail catalog is hidden
// from the web store (WebHidden) to leave the menu clean. It keeps its own clock, like crm: two
// service days do not fit the day loop.
package dining

import (
	"fmt"
	"slices"

	business "app/business"
	businessTypes "app/business/types"
	"app/core"
	"app/db"
	production "app/production"
	productionTypes "app/production/types"
	"app/seeds/runner"
)

var Module = runner.Module{Name: "dining", Setup: Setup}

const (
	categoryEntradas = "Entradas"
	categoryFondos   = "Platos de fondo"
	categoryPostres  = "Postres"
	// dishWebExtraStock is a made dish's "Sobreventa permitida": what the kitchen can serve until the
	// daily production close makes the margin whole again.
	dishWebExtraStock = int32(200)
	hideBatchSize     = 500
)

// menuDish is one dish of the menu; Price in cents.
type menuDish struct {
	Name     string
	Category string
	Price    int32
}

// The menu: platos de fondo between S/ 15 and S/ 25, entradas at S/ 5, postres at S/ 4.
var menuDishes = []menuDish{
	{"Lomo saltado", categoryFondos, 2500},
	{"Ceviche clásico", categoryFondos, 2400},
	{"Arroz con mariscos", categoryFondos, 2500},
	{"Ají de gallina", categoryFondos, 1800},
	{"Seco de res con frejoles", categoryFondos, 2200},
	{"Tallarines verdes con bistec", categoryFondos, 2100},
	{"Pollo a la plancha con ensalada", categoryFondos, 1600},
	{"Arroz chaufa de pollo", categoryFondos, 1700},
	{"Tacu tacu con lomo", categoryFondos, 2400},
	{"Carapulcra con sopa seca", categoryFondos, 1900},
	{"Chicharrón de pescado", categoryFondos, 2300},
	{"Pescado a lo macho", categoryFondos, 2500},
	{"Estofado de pollo", categoryFondos, 1600},
	{"Olluquito con charqui", categoryFondos, 1800},
	{"Cau cau con arroz", categoryFondos, 1500},
	{"Adobo arequipeño", categoryFondos, 2000},
	{"Sudado de pescado", categoryFondos, 2200},
	{"Tallarín saltado de carne", categoryFondos, 2000},
	{"Milanesa de pollo con papas", categoryFondos, 1700},
	{"Arroz con pato", categoryFondos, 2500},
	{"Causa limeña", categoryEntradas, 500},
	{"Papa a la huancaína", categoryEntradas, 500},
	{"Ocopa arequipeña", categoryEntradas, 500},
	{"Sopa criolla de la casa", categoryEntradas, 500},
	{"Mazamorra morada", categoryPostres, 400},
	{"Arroz con leche de la casa", categoryPostres, 400},
	{"Suspiro a la limeña", categoryPostres, 400},
}

// menuPackSlot is one slot of a seeded pack: a dish by name, or a whole category.
type menuPackSlot struct {
	DishName string
	Category string
	Quantity int16
}

type menuPack struct {
	Name         string
	Slots        []menuPackSlot
	PriceType    int8
	PriceValue   int32
	RoundingMode int8
}

// The packs. The category menus weigh 2 per slot and the dish combos 5 per dish slot
// (sales.PackRelevance), so a combo wins the units it shares with a menu.
var menuPacks = []menuPack{
	{Name: "Menú completo", PriceType: productionTypes.PackPriceAmountOff, PriceValue: 1000, Slots: []menuPackSlot{
		{Category: categoryEntradas, Quantity: 1}, {Category: categoryFondos, Quantity: 1}, {Category: categoryPostres, Quantity: 1}}},
	{Name: "Menú ejecutivo", PriceType: productionTypes.PackPriceAmountOff, PriceValue: 500, Slots: []menuPackSlot{
		{Category: categoryEntradas, Quantity: 1}, {Category: categoryFondos, Quantity: 1}}},
	{Name: "Dúo marino", PriceType: productionTypes.PackPriceFixed, PriceValue: 2500, Slots: []menuPackSlot{
		{DishName: "Ceviche clásico", Quantity: 1}, {DishName: "Causa limeña", Quantity: 1}}},
	{Name: "Lomo con suspiro", PriceType: productionTypes.PackPriceAmountOff, PriceValue: 400, Slots: []menuPackSlot{
		{DishName: "Lomo saltado", Quantity: 1}, {DishName: "Suspiro a la limeña", Quantity: 1}}},
	{Name: "Criollo completo", PriceType: productionTypes.PackPriceFixed, PriceValue: 2000, Slots: []menuPackSlot{
		{DishName: "Papa a la huancaína", Quantity: 1}, {DishName: "Ají de gallina", Quantity: 1}, {DishName: "Mazamorra morada", Quantity: 1}}},
	{Name: "Arroz con pato + postre", PriceType: productionTypes.PackPricePercentOff, PriceValue: 1500,
		RoundingMode: productionTypes.PackRoundingDown, Slots: []menuPackSlot{
			{DishName: "Arroz con pato", Quantity: 1}, {Category: categoryPostres, Quantity: 1}}},
}

// diningMenu is the seeded menu as the service simulation needs it.
type diningMenu struct {
	categoryIDByName map[string]int32
	dishByName       map[string]productionTypes.Product
	dishesByCategory map[string][]productionTypes.Product
}

// diningUser is the run's admin, granted the table service and its "Cobrar mesa" sub-access: the
// fareward gate fills SubAccesos on a real request, a seed has to set them itself.
func diningUser() core.UsuarioToken {
	return core.UsuarioToken{CompanyID: runner.CompanyID, ID: runner.UserID, SubAccesos: map[int32]uint16{61: 0xFFFF, 62: 0xFFFF}}
}

func callAsDiningUser(route string, handler func(*core.HandlerArgs) core.HandlerResponse, payload any, target any) error {
	return runner.CallAs(diningUser(), route, handler, nil, payload, target)
}

// Setup writes the menu, the packs and the hidden retail catalog on every run (each step only fills
// what is missing), then the rooms and the two service days once: a company that already has the
// "Terraza" room is left as it is.
func Setup(run *runner.Run) ([]runner.DayStep, error) {
	core.SetHistoricalUnix(run.RealNow.AddDate(0, 0, -2).Unix())
	menu, err := ensureDiningMenu(run)
	if err != nil {
		return nil, core.Err("menú del restaurante:", err)
	}
	if err := hideRetailFromWebStore(menu); err != nil {
		return nil, core.Err("ocultar el catálogo minorista:", err)
	}
	if err := ensureMenuPacks(menu); err != nil {
		return nil, core.Err("packs del menú:", err)
	}
	layouts, isNew, err := ensureDiningLayouts(run, menu)
	if err != nil {
		return nil, core.Err("salones:", err)
	}
	if !isNew {
		core.Log("SeedDining:: la empresa ya tiene la Terraza, se omite el servicio de mesas")
		return nil, nil
	}
	if err := runTableService(run, menu, layouts); err != nil {
		return nil, core.Err("servicio de mesas:", err)
	}
	return nil, nil
}

// ensureDiningMenu creates the three categories and the dishes the company does not have yet. A
// dish name already taken by a product outside the menu's categories is refused: the menu would
// pick up a retail product.
func ensureDiningMenu(run *runner.Run) (diningMenu, error) {
	menu := diningMenu{categoryIDByName: map[string]int32{}, dishByName: map[string]productionTypes.Product{},
		dishesByCategory: map[string][]productionTypes.Product{}}

	categoryRecords := []businessTypes.SharedListRecord{}
	categoryQuery := db.Query(&categoryRecords)
	categoryQuery.Select(categoryQuery.ID, categoryQuery.NameHash, categoryQuery.Status).
		CompanyID.Equals(runner.CompanyID).ListID.Equals(businessTypes.SharedListCategoryID)
	if err := categoryQuery.Exec(); err != nil {
		return menu, core.Err("no se pudieron leer las categorías:", err)
	}
	missingCategories := []businessTypes.SharedListRecord{}
	for _, categoryName := range []string{categoryEntradas, categoryFondos, categoryPostres} {
		category := businessTypes.SharedListRecord{ListID: businessTypes.SharedListCategoryID, Name: categoryName, Status: 1}
		category.SelfParse()
		storedIndex := slices.IndexFunc(categoryRecords, func(record businessTypes.SharedListRecord) bool {
			return record.NameHash == category.NameHash && record.Status > 0
		})
		if storedIndex >= 0 {
			menu.categoryIDByName[categoryName] = categoryRecords[storedIndex].ID
			continue
		}
		category.ID = int32(-(len(missingCategories) + 1))
		missingCategories = append(missingCategories, category)
	}
	if len(missingCategories) > 0 {
		idMappings := []businessTypes.NewIDToID{}
		if err := callAsDiningUser("POST.shared-lists", business.PostSharedLists, missingCategories, &idMappings); err != nil {
			return menu, err
		}
		for _, category := range missingCategories {
			mappingIndex := slices.IndexFunc(idMappings, func(mapping businessTypes.NewIDToID) bool { return mapping.TempID == category.ID })
			if mappingIndex < 0 {
				return menu, core.Err("POST.shared-lists no devolvió el ID de la categoría", category.Name)
			}
			menu.categoryIDByName[category.Name] = idMappings[mappingIndex].ID
		}
	}

	nameHashes := []int32{}
	for _, dish := range menuDishes {
		nameHashes = append(nameHashes, productNameHash(dish.Name))
	}
	storedProducts := []productionTypes.Product{}
	productQuery := db.Query(&storedProducts)
	productQuery.Select(productQuery.ID, productQuery.Name, productQuery.NameHash, productQuery.CategoryIDs, productQuery.Status).
		CompanyID.Equals(runner.CompanyID).NameHash.In(nameHashes...)
	if err := productQuery.Exec(); err != nil {
		return menu, core.Err("no se pudieron leer los platos:", err)
	}
	newDishes := []productionTypes.Product{}
	for _, dish := range menuDishes {
		categoryID := menu.categoryIDByName[dish.Category]
		storedIndex := slices.IndexFunc(storedProducts, func(product productionTypes.Product) bool {
			return product.NameHash == productNameHash(dish.Name) && product.Status == productionTypes.ProductStatusActive
		})
		if storedIndex >= 0 {
			if !slices.Contains(storedProducts[storedIndex].CategoryIDs, categoryID) {
				return menu, core.Err(fmt.Sprintf(`ya existe un producto "%s" fuera de la categoría "%s"`, dish.Name, dish.Category))
			}
			continue
		}
		// A made dish sells from its margin, costed at a standard 35% of its price.
		newDishes = append(newDishes, productionTypes.Product{
			ID: int32(-(len(newDishes) + 1)), Name: dish.Name, Price: dish.Price, FinalPrice: dish.Price,
			CategoryIDs: []int32{categoryID}, Origin: productionTypes.ProductOriginMade,
			WebExtraStock: dishWebExtraStock, ProductionStandardCost: dish.Price * 35 / 118,
		})
	}
	if len(newDishes) > 0 {
		if err := callAsDiningUser("POST.products", production.PostProducts, newDishes, nil); err != nil {
			return menu, err
		}
		core.Log("SeedDining:: platos creados", len(newDishes))
	}

	// Read back what the simulation needs: the IDs, prices and categories as stored.
	dishes := []productionTypes.Product{}
	dishQuery := db.Query(&dishes)
	dishQuery.Select(dishQuery.ID, dishQuery.Name, dishQuery.NameHash, dishQuery.FinalPrice, dishQuery.CategoryIDs, dishQuery.Status).
		CompanyID.Equals(runner.CompanyID).NameHash.In(nameHashes...)
	if err := dishQuery.Exec(); err != nil {
		return menu, core.Err("no se pudieron leer los platos:", err)
	}
	for _, dish := range menuDishes {
		storedIndex := slices.IndexFunc(dishes, func(product productionTypes.Product) bool {
			return product.NameHash == productNameHash(dish.Name) && product.Status == productionTypes.ProductStatusActive
		})
		if storedIndex < 0 {
			return menu, core.Err(fmt.Sprintf(`el plato "%s" no quedó registrado`, dish.Name))
		}
		menu.dishByName[dish.Name] = dishes[storedIndex]
		menu.dishesByCategory[dish.Category] = append(menu.dishesByCategory[dish.Category], dishes[storedIndex])
	}
	return menu, nil
}

// hideRetailFromWebStore marks every other active product WebHidden, through the products form's
// own handler: the dining menu lists what the web store sells, and only the dishes belong there.
func hideRetailFromWebStore(menu diningMenu) error {
	menuProductIDs := map[int32]bool{}
	for _, dish := range menu.dishByName {
		menuProductIDs[dish.ID] = true
	}
	products := []productionTypes.Product{}
	query := db.Query(&products)
	query.Select().CompanyID.Equals(runner.CompanyID).Status.Equals(productionTypes.ProductStatusActive)
	if err := query.Exec(); err != nil {
		return core.Err("no se pudieron leer los productos:", err)
	}
	visibleRetail := slices.DeleteFunc(products, func(product productionTypes.Product) bool {
		return product.WebHidden || menuProductIDs[product.ID]
	})
	for batchStart := 0; batchStart < len(visibleRetail); batchStart += hideBatchSize {
		batch := visibleRetail[batchStart:min(batchStart+hideBatchSize, len(visibleRetail))]
		for index := range batch {
			batch[index].WebHidden = true
		}
		if err := callAsDiningUser("POST.products", production.PostProducts, batch, nil); err != nil {
			return err
		}
		core.Log("SeedDining:: productos ocultos de la tienda web", batchStart+len(batch), "de", len(visibleRetail))
	}
	return nil
}

// ensureMenuPacks creates the packs the company does not have yet, matched by name.
func ensureMenuPacks(menu diningMenu) error {
	storedPacks := []productionTypes.ProductPack{}
	query := db.Query(&storedPacks)
	query.Select(query.ID, query.Name, query.Status).CompanyID.Equals(runner.CompanyID).Status.Equals(1)
	if err := query.Exec(); err != nil {
		return core.Err("no se pudieron leer los packs:", err)
	}
	for _, seedPack := range menuPacks {
		if slices.ContainsFunc(storedPacks, func(pack productionTypes.ProductPack) bool { return pack.Name == seedPack.Name }) {
			continue
		}
		pack := productionTypes.ProductPack{Name: seedPack.Name, PriceType: seedPack.PriceType, PriceValue: seedPack.PriceValue,
			RoundingMode: seedPack.RoundingMode, Status: 1}
		for _, slot := range seedPack.Slots {
			pack.Components = append(pack.Components, productionTypes.ProductPackComponent{
				ProductID: menu.dishByName[slot.DishName].ID, CategoryID: menu.categoryIDByName[slot.Category], Quantity: slot.Quantity,
			})
		}
		if err := callAsDiningUser("POST.product-pack", production.PostProductPack, pack, nil); err != nil {
			return err
		}
		core.Log("SeedDining:: pack creado", pack.Name)
	}
	return nil
}

// productNameHash is the hash PostProducts dedupes names on, from the record's own SelfParse.
func productNameHash(name string) int32 {
	product := productionTypes.Product{Name: name}
	product.SelfParse()
	return product.NameHash
}

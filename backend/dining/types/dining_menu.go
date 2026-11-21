package types

import "app/db"

// DiningMenu is a carta: the menu a layout's customers see when they scan a table's QR
// (docs/DINING_MENU_PLAN.md §1). It is edited and saved as one document, so its blocks are a blob.
type DiningMenu struct {
	db.TableStruct[DiningMenuTable, DiningMenu]
	CompanyID int32           `cb:"1"`
	ID        int32           `cb:"2"`
	Name      string          `cb:"3"`
	Theme     DiningMenuTheme `cb:"4"`
	// Blocks are in display order.
	Blocks    []DiningMenuBlock `cb:"5"`
	Status    int8              `json:"ss" cb:"ss,6"`
	Updated   int32             `json:"upd" cb:"upd,7"`
	UpdatedBy int32             `cb:"8"`
	Version   int16             `json:"ver" cb:"ver,9"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,10"`
}

type DiningMenuTable struct {
	db.TableStruct[DiningMenuTable, DiningMenu]
	CompanyID db.Col[*DiningMenuTable, int32]
	ID        db.Col[*DiningMenuTable, int32]
	Name      db.Col[*DiningMenuTable, string]
	Theme     db.Col[*DiningMenuTable, DiningMenuTheme]
	Blocks    db.Col[*DiningMenuTable, []DiningMenuBlock]
	Status    db.Col[*DiningMenuTable, int8]
	Updated   db.Col[*DiningMenuTable, int32]
	UpdatedBy db.Col[*DiningMenuTable, int32]
	Version   db.Col[*DiningMenuTable, int16]
}

func (e DiningMenuTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          127,
		Name:        "dining_menu",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Min: 0, Max: 1}},
		Indexes: []db.Index{
			// The company's cartas, delta-synced by the builder and the layout settings.
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// The kinds of DiningMenuBlock.
const (
	MenuBlockAllProducts = int8(1) // every restaurant product, grouped by category
	MenuBlockCategory    = int8(2) // one category's products
	MenuBlockProducts    = int8(3) // hand-picked products
	MenuBlockTitle       = int8(4)
	MenuBlockSubtitle    = int8(5)
	MenuBlockText        = int8(6)
	MenuBlockImage       = int8(7)
	MenuBlockDivider     = int8(8)
)

// How a product block lists its products.
const (
	MenuProductsCompact   = int8(1) // name ··· price
	MenuProductsWithPhoto = int8(2) // thumbnail, name, description, price
	MenuProductsCards     = int8(3) // 2-column photo cards
)

// The fonts a carta's theme can pick.
const (
	MenuFontSans    = int8(1)
	MenuFontSerif   = int8(2)
	MenuFontRounded = int8(3)
)

// The alignments of a text block.
const (
	MenuAlignLeft   = int8(1)
	MenuAlignCenter = int8(2)
)

// DiningMenuTheme is the carta-wide look.
type DiningMenuTheme struct {
	AccentColor     string `cb:"1"` // "#b45309": titles, prices, separators
	BackgroundColor string `cb:"2"` // the text color is derived from it by contrast
	Font            int8   `cb:"3"`
}

// DiningMenuBlock is one block of the carta. One struct for every kind: each kind reads only the
// fields the plan's table names for it (§1.1); the rest stay empty.
type DiningMenuBlock struct {
	Kind       int8    `cb:"1"`
	Text       string  `cb:"2"`
	CategoryID int32   `cb:"3"`
	ProductIDs []int32 `cb:"4"`
	ImageURL   string  `cb:"5"`
	Variant    int8    `cb:"6"`
	Align      int8    `cb:"7"`
	// RestOnly (all-products block): leave out the products another block already lists.
	RestOnly bool `cb:"8"`
}

// DefaultMenuTheme is the look of a carta saved with no theme, and of the default carta.
func DefaultMenuTheme() DiningMenuTheme {
	return DiningMenuTheme{AccentColor: "#b45309", BackgroundColor: "#fffbeb", Font: MenuFontSans}
}

// DefaultMenuBlocks is the carta of a layout with no carta chosen, and the start of a new one.
func DefaultMenuBlocks() []DiningMenuBlock {
	return []DiningMenuBlock{{Kind: MenuBlockAllProducts, Variant: MenuProductsWithPhoto}}
}

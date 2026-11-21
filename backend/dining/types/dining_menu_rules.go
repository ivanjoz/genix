package types

import (
	"app/core"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// The limits a saved carta must respect (docs/DINING_MENU_PLAN.md §1.3).
const (
	MaxMenuNameLength     = 80
	MaxMenuBlocks         = 200
	MaxMenuHeadingLength  = 120
	MaxMenuTextLength     = 1000
	MaxMenuCaptionLength  = 200
	MaxMenuBlockProducts  = 100
	MaxMenuImageURLLength = 500
)

var hexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// NormalizeDiningMenu trims the free texts, fills a missing theme with the default and clears the
// fields a block's kind does not use, so the stored blob holds only what renders.
func NormalizeDiningMenu(menu *DiningMenu) {
	menu.Name = strings.TrimSpace(menu.Name)
	if menu.Theme == (DiningMenuTheme{}) {
		menu.Theme = DefaultMenuTheme()
	}
	for index, block := range menu.Blocks {
		kept := DiningMenuBlock{Kind: block.Kind}
		text := strings.TrimSpace(block.Text)
		switch block.Kind {
		case MenuBlockAllProducts:
			kept.Variant, kept.RestOnly = block.Variant, block.RestOnly
		case MenuBlockCategory:
			kept.CategoryID, kept.Text, kept.Variant = block.CategoryID, text, block.Variant
		case MenuBlockProducts:
			kept.ProductIDs, kept.Variant = block.ProductIDs, block.Variant
		case MenuBlockTitle, MenuBlockSubtitle, MenuBlockText:
			kept.Text, kept.Align = text, block.Align
		case MenuBlockImage:
			kept.ImageURL, kept.Text = strings.TrimSpace(block.ImageURL), text
		}
		menu.Blocks[index] = kept
	}
}

// ValidateDiningMenu checks a normalized carta. activeCategoryIDs and activeProductIDs are the
// company's active categories and products among the ones the carta names.
func ValidateDiningMenu(menu DiningMenu, activeCategoryIDs map[int32]bool, activeProductIDs map[int32]bool) error {
	if menu.Name == "" || len(menu.Name) > MaxMenuNameLength {
		return core.Err(fmt.Sprintf("El nombre de la carta es obligatorio y admite hasta %v caracteres.", MaxMenuNameLength))
	}
	if !hexColorPattern.MatchString(menu.Theme.AccentColor) || !hexColorPattern.MatchString(menu.Theme.BackgroundColor) {
		return core.Err("Los colores de la carta deben tener el formato #rrggbb.")
	}
	if !slices.Contains([]int8{MenuFontSans, MenuFontSerif, MenuFontRounded}, menu.Theme.Font) {
		return core.Err("La fuente de la carta no es válida.")
	}
	if len(menu.Blocks) > MaxMenuBlocks {
		return core.Err(fmt.Sprintf("Una carta admite hasta %v bloques.", MaxMenuBlocks))
	}
	for index, block := range menu.Blocks {
		if err := validateMenuBlock(block, activeCategoryIDs, activeProductIDs); err != nil {
			return core.Err(fmt.Sprintf("Bloque %v:", index+1), err)
		}
	}
	return nil
}

func validateMenuBlock(block DiningMenuBlock, activeCategoryIDs map[int32]bool, activeProductIDs map[int32]bool) error {
	isKnownVariant := slices.Contains([]int8{MenuProductsCompact, MenuProductsWithPhoto, MenuProductsCards}, block.Variant)
	switch block.Kind {
	case MenuBlockAllProducts:
		if !isKnownVariant {
			return core.Err("el estilo de los productos no es válido.")
		}
	case MenuBlockCategory:
		if !isKnownVariant {
			return core.Err("el estilo de los productos no es válido.")
		}
		if !activeCategoryIDs[block.CategoryID] {
			return core.Err("la categoría no existe o no está activa.")
		}
		if len(block.Text) > MaxMenuHeadingLength {
			return core.Err(fmt.Sprintf("el encabezado admite hasta %v caracteres.", MaxMenuHeadingLength))
		}
	case MenuBlockProducts:
		if !isKnownVariant {
			return core.Err("el estilo de los productos no es válido.")
		}
		if len(block.ProductIDs) == 0 || len(block.ProductIDs) > MaxMenuBlockProducts {
			return core.Err(fmt.Sprintf("debe tener de 1 a %v productos.", MaxMenuBlockProducts))
		}
		seen := map[int32]bool{}
		for _, productID := range block.ProductIDs {
			if seen[productID] {
				return core.Err(fmt.Sprintf("el producto %v está repetido.", productID))
			}
			seen[productID] = true
			if !activeProductIDs[productID] {
				return core.Err(fmt.Sprintf("el producto %v no existe o no está activo.", productID))
			}
		}
	case MenuBlockTitle, MenuBlockSubtitle, MenuBlockText:
		maxLength := MaxMenuHeadingLength
		if block.Kind == MenuBlockText {
			maxLength = MaxMenuTextLength
		}
		if block.Text == "" || len(block.Text) > maxLength {
			return core.Err(fmt.Sprintf("el texto es obligatorio y admite hasta %v caracteres.", maxLength))
		}
		if block.Align != MenuAlignLeft && block.Align != MenuAlignCenter {
			return core.Err("la alineación no es válida.")
		}
	case MenuBlockImage:
		if block.ImageURL == "" || len(block.ImageURL) > MaxMenuImageURLLength {
			return core.Err("la imagen es obligatoria.")
		}
		if len(block.Text) > MaxMenuCaptionLength {
			return core.Err(fmt.Sprintf("el pie de imagen admite hasta %v caracteres.", MaxMenuCaptionLength))
		}
	case MenuBlockDivider:
	default:
		return core.Err("el tipo de bloque no es válido.")
	}
	return nil
}

// MenuReferencedIDs are the category and product IDs a carta's blocks name, for the handler to
// check against the catalog.
func MenuReferencedIDs(menu DiningMenu) (categoryIDs []int32, productIDs []int32) {
	for _, block := range menu.Blocks {
		if block.Kind == MenuBlockCategory && block.CategoryID > 0 {
			categoryIDs = append(categoryIDs, block.CategoryID)
		}
		productIDs = append(productIDs, block.ProductIDs...)
	}
	return categoryIDs, productIDs
}

package production

import (
	"app/production/types"
	"reflect"
	"slices"
	"testing"
)

// The public by-IDs route answers anyone who knows a company id, so every Product field must be a
// deliberate choice. Adding a field fails this test until it lands in one of these two lists.
var publicProductFieldNames = []string{
	"ID", "Name", "Description", "CategoryIDs", "BrandID",
	"Price", "Discount", "FinalPrice", "CurrencyID",
	"SbuQuantity", "SbuUnit", "SbuPrice", "SbuDiscount", "SbuFinalPrice",
	"Presentations", "ImageMain", "ImageIDs", "ImageDescriptions",
	"Status", "Updated",
}

var privateProductFieldNames = []string{
	"CompanyID", "TempID", "ContentHTML", "Params", "UnitID", "Weight", "Volume", "SKU", "NameHash",
	"DepreciationMonths", "Properties", "Stock", "ReservedStock", "StockStatus", "NameUpdated",
	"UpdatedBy", "Created", "CreatedBy", "CategoriesWithStock", "Version", "Metadata", "BrandName",
	"AccountCode", "Origin", "WebExtraStock", "WebHidden",
	"ProductionYieldPortions", "ProductionSupplies", "ProductionExtraCostPerUnit", "ProductionStandardCost",
	"ProductionLabor", "TracksLots", "ShelfLifeDays", "QualityHoldOnEntry",
}

func TestPublicProductFieldsClassifyEveryField(t *testing.T) {
	productType := reflect.TypeOf(types.Product{})
	fullProduct := reflect.New(productType).Elem()
	for fieldIndex := 0; fieldIndex < productType.NumField(); fieldIndex++ {
		field := productType.Field(fieldIndex)
		if field.Anonymous {
			continue // db.TableStruct, the ORM's embedded marker
		}
		isPublic := slices.Contains(publicProductFieldNames, field.Name)
		if isPublic == slices.Contains(privateProductFieldNames, field.Name) {
			t.Fatalf("Product.%s must be listed exactly once, as public or as private", field.Name)
		}
		setNonZero(t, fullProduct.Field(fieldIndex), field.Name)
	}

	projected := reflect.ValueOf(publicProductFields(fullProduct.Interface().(types.Product)))
	for fieldIndex := 0; fieldIndex < productType.NumField(); fieldIndex++ {
		field := productType.Field(fieldIndex)
		if field.Anonymous {
			continue
		}
		isCopied := !projected.Field(fieldIndex).IsZero()
		if isPublic := slices.Contains(publicProductFieldNames, field.Name); isCopied != isPublic {
			t.Errorf("Product.%s: public=%v but copied=%v by publicProductFields", field.Name, isPublic, isCopied)
		}
	}
}

// setNonZero gives a field any non-zero value of its kind, so the projection's copy is observable.
func setNonZero(t *testing.T, value reflect.Value, fieldName string) {
	switch value.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value.SetInt(1)
	case reflect.Float32, reflect.Float64:
		value.SetFloat(1)
	case reflect.String:
		value.SetString("x")
	case reflect.Bool:
		value.SetBool(true)
	case reflect.Slice:
		value.Set(reflect.MakeSlice(value.Type(), 1, 1))
	case reflect.Pointer:
		value.Set(reflect.New(value.Type().Elem()))
	default:
		t.Fatalf("Product.%s: unhandled kind %v", fieldName, value.Kind())
	}
}

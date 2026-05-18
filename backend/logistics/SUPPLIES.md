I need to crate a new table called supply_materiales

the struct is SupplyMaterial

I need the column:

CompanyID
Name
Description
Brand
Price
CurrencyID
SKU
ProviderSupply // similar to ProductSupplyProviderRow

// Propiedades generales
Status    int8  `json:"ss,omitempty"`
Updated   int32 `json:"upd,omitempty"`
UpdatedBy int32 `json:",omitempty"`
Created   int32 `json:",omitempty"`
CreatedBy int32 `json:",omitempty"`

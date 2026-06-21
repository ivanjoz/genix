package types

import (
	"app/db"
)

// TextLine mirrors the frontend ITextLine (renderer-types.ts): one editable line
// of rich text with its own Tailwind classes.
type TextLine struct {
	Text string `json:"text,omitempty" cbor:"1,keyasint,omitempty"`
	Css  string `json:"css,omitempty" cbor:"2,keyasint,omitempty"`
	Tag  string `json:"tag,omitempty" cbor:"3,keyasint,omitempty"`
}

// GalleryImage mirrors the frontend IGalleryImagen.
type GalleryImage struct {
	Image       string `json:"image,omitempty" cbor:"1,keyasint,omitempty"`
	Title       string `json:"title,omitempty" cbor:"2,keyasint,omitempty"`
	Description string `json:"description,omitempty" cbor:"3,keyasint,omitempty"`
}

// AstNode mirrors the frontend ComponentAST (renderer-types.ts): one node of a
// parsed HTML section tree. The json tags keep the frontend's camelCase casing so
// the tree round-trips unchanged; cbor keyasint gives compact storage. `Props`
// stays a string-keyed map of coerced primitive component props (the frontend's
// Record<string, any>) — its values are primitives, so it decodes safely.
type AstNode struct {
	TagName    string            `json:"tagName,omitempty" cbor:"1,keyasint,omitempty"`
	Css        string            `json:"css,omitempty" cbor:"2,keyasint,omitempty"`
	Style      string            `json:"style,omitempty" cbor:"3,keyasint,omitempty"`
	Text       string            `json:"text,omitempty" cbor:"4,keyasint,omitempty"`
	Children   []AstNode         `json:"children,omitempty" cbor:"5,keyasint,omitempty"`
	Role       string            `json:"role,omitempty" cbor:"6,keyasint,omitempty"`
	Props      map[string]any    `json:"props,omitempty" cbor:"7,keyasint,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty" cbor:"8,keyasint,omitempty"`
}

// ContentFields mirrors the frontend StandardContent (section-types.ts): the flat
// content schema for component sections. The catch-all `[key]: any` is dropped —
// only these known fields are persisted.
type ContentFields struct {
	Title                string         `json:"title,omitempty" cbor:"1,keyasint,omitempty"`
	SubTitle             string         `json:"subTitle,omitempty" cbor:"2,keyasint,omitempty"`
	Description          string         `json:"description,omitempty" cbor:"3,keyasint,omitempty"`
	TextLeft             string         `json:"textLeft,omitempty" cbor:"4,keyasint,omitempty"`
	TextCenter           string         `json:"textCenter,omitempty" cbor:"5,keyasint,omitempty"`
	TextRight            string         `json:"textRight,omitempty" cbor:"6,keyasint,omitempty"`
	TextLines            []TextLine     `json:"textLines,omitempty" cbor:"7,keyasint,omitempty"`
	Image                string         `json:"image,omitempty" cbor:"8,keyasint,omitempty"`
	SecondaryImagen      string         `json:"secondaryImagen,omitempty" cbor:"9,keyasint,omitempty"`
	IconImagen           string         `json:"iconImagen,omitempty" cbor:"10,keyasint,omitempty"`
	BgImage              string         `json:"bgImage,omitempty" cbor:"11,keyasint,omitempty"`
	VideoUrl             string         `json:"videoUrl,omitempty" cbor:"12,keyasint,omitempty"`
	ProductIDs           []int32        `json:"productIDs,omitempty" cbor:"13,keyasint,omitempty"`
	CategoryIDs          []int32        `json:"categoryIDs,omitempty" cbor:"14,keyasint,omitempty"`
	BrandIDs             []int32        `json:"brandIDs,omitempty" cbor:"15,keyasint,omitempty"`
	Gallery              []GalleryImage `json:"gallery,omitempty" cbor:"16,keyasint,omitempty"`
	Limit                int32          `json:"limit,omitempty" cbor:"17,keyasint,omitempty"`
	PrimaryActionLabel   string         `json:"primaryActionLabel,omitempty" cbor:"18,keyasint,omitempty"`
	PrimaryActionHref    string         `json:"primaryActionHref,omitempty" cbor:"19,keyasint,omitempty"`
	SecondaryActionLabel string         `json:"secondaryActionLabel,omitempty" cbor:"20,keyasint,omitempty"`
	SecondaryActionHref  string         `json:"secondaryActionHref,omitempty" cbor:"21,keyasint,omitempty"`
}

// SectionContent mirrors the persisted fields of the frontend SectionData
// (renderer/section-types.ts). Top-level fields are PascalCase to match the
// frontend rename; nested frontend-owned trees (Ast/Content) are strict typed
// structs. The ORM stores the whole struct as CBOR (keyasint for compact keys).
type SectionContent struct {
	Type       string            `json:",omitempty" cbor:"1,keyasint,omitempty"`
	Ast        []AstNode         `json:",omitempty" cbor:"2,keyasint,omitempty"`
	Content    *ContentFields    `json:",omitempty" cbor:"3,keyasint,omitempty"`
	Css        map[string]string `json:",omitempty" cbor:"4,keyasint,omitempty"`
	Attributes map[string]any    `json:",omitempty" cbor:"5,keyasint,omitempty"`
	// PageCss is the section's pre-generated runtime Tailwind CSS (the UnoCSS
	// output for its tokens, wrapped in @layer ec-runtime). The builder generates
	// it on save so the storefront concatenates the stored stylesheets instead of
	// running the UnoCSS engine at view time. It rides in the content blob so it is
	// covered by the section hash.
	PageCss string `json:",omitempty" cbor:"6,keyasint,omitempty"`
	// Svgs deduplicates the inline SVG bodies of icons picked in the builder, keyed by
	// sprite id `icon--<set>-<name>`. The frontend renders one <symbol> per entry and each
	// Icon AST node references it via <use href="#id">, so each body is stored exactly once.
	Svgs map[string]string `json:",omitempty" cbor:"7,keyasint,omitempty"`
	// Palette is the page's growable color list (hex colors, referenced 1-based as
	// var(--color-N)). Page-global, so it rides on section 1 only — same convention as
	// the whole-page Css. The builder grows it as the agent introduces new colors.
	Palette []string `json:",omitempty" cbor:"8,keyasint,omitempty"`
	// CustomCss is the agent-authored raw CSS for this section, already scoped to
	// page-unique `.x{n}` classes by the builder. It rides in the content blob (so
	// it round-trips into the editor) and is also folded into the whole-page Css
	// column on save so the storefront serves it without regenerating.
	CustomCss string `json:",omitempty" cbor:"9,keyasint,omitempty"`
}

// EcommercePageContent stores one builder section, addressed by its page and its
// 1-based position (SectionID). Dedup is by Hash (FNV-1a 64 of the section JSON),
// computed server-side so unchanged sections are skipped on save. Removed
// positions are soft-deleted (Status=0) since the ORM has no hard delete.
type EcommercePageContent struct {
	db.TableStruct[EcommercePageContentTable, EcommercePageContent]
	CompanyID int32          `json:",omitempty"`
	PageID    int16          `json:",omitempty"`
	SectionID int16          `json:",omitempty"`
	Route     string         `json:",omitempty"`
	Content   SectionContent `json:",omitempty"`
	// Css holds the whole-page pre-generated runtime Tailwind CSS (the UnoCSS output
	// for every section's tokens). It is stored only on section 1 so the storefront
	// serves a single stylesheet from a plain column — no CBOR decode of Content
	// needed. The builder ships it inside section 1's SectionContent.PageCss on save;
	// the handler moves it here and clears PageCss to avoid storing two copies.
	Css       string         `json:",omitempty"`
	Hash      int64          `json:",omitempty"`
	Status    int8           `json:"ss,omitempty"`
	Updated   int32          `json:"upd,omitempty"`
	UpdatedBy int32          `json:",omitempty"`
}

type EcommercePageContentTable struct {
	db.TableStruct[EcommercePageContentTable, EcommercePageContent]
	CompanyID db.Col[EcommercePageContentTable, int32]
	PageID    db.Col[EcommercePageContentTable, int16]
	SectionID db.Col[EcommercePageContentTable, int16]
	Route     db.Col[EcommercePageContentTable, string]
	Content   db.Col[EcommercePageContentTable, SectionContent]
	Css       db.Col[EcommercePageContentTable, string]
	Hash      db.Col[EcommercePageContentTable, int64]
	Status    db.Col[EcommercePageContentTable, int8]
	Updated   db.Col[EcommercePageContentTable, int32]
	UpdatedBy db.Col[EcommercePageContentTable, int32]
}

func (e EcommercePageContentTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "ecommerce_page_content",
		Partition: e.CompanyID,
		// (CompanyID, PageID, SectionID) is the PK. SectionID is the section's
		// 1-based position on the page, recomputed on every save.
		Keys: []db.Coln{e.PageID, e.SectionID},
	}
}

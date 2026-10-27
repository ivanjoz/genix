package types

import (
	"app/db"
)

// TextLine mirrors the frontend ITextLine (renderer-types.ts): one editable line
// of rich text with its own Tailwind classes.
type TextLine struct {
	Text string `json:"text,omitempty" cb:"text,1"`
	Css  string `json:"css,omitempty" cb:"css,2"`
	Tag  string `json:"tag,omitempty" cb:"tag,3"`
}

// GalleryImage mirrors the frontend IGalleryImagen.
type GalleryImage struct {
	Image       string `json:"image,omitempty" cb:"image,1"`
	Title       string `json:"title,omitempty" cb:"title,2"`
	Description string `json:"description,omitempty" cb:"description,3"`
}

// AstNode mirrors the frontend ComponentAST (renderer-types.ts): one node of a
// parsed HTML section tree. The json tags keep the frontend's camelCase casing so
// the tree round-trips unchanged; colbin gives compact storage. `Props`
// stays a string-keyed map of coerced primitive component props (the frontend's
// Record<string, any>) — its values are primitives, so it decodes safely.
type AstNode struct {
	TagName    string            `json:"tagName,omitempty" cb:"tagName,1"`
	Css        string            `json:"css,omitempty" cb:"css,2"`
	Style      string            `json:"style,omitempty" cb:"style,3"`
	Text       string            `json:"text,omitempty" cb:"text,4"`
	Children   []AstNode         `json:"children,omitempty" cb:"children,5"`
	Role       string            `json:"role,omitempty" cb:"role,6"`
	Props      map[string]any    `json:"props,omitempty" cb:"props,7"`
	Attributes map[string]string `json:"attributes,omitempty" cb:"attributes,8"`
}

// ContentFields mirrors the frontend StandardContent (section-types.ts): the flat
// content schema for component sections. The catch-all `[key]: any` is dropped —
// only these known fields are persisted.
type ContentFields struct {
	Title                string         `json:"title,omitempty" cb:"title,1"`
	SubTitle             string         `json:"subTitle,omitempty" cb:"subTitle,2"`
	Description          string         `json:"description,omitempty" cb:"description,3"`
	TextLeft             string         `json:"textLeft,omitempty" cb:"textLeft,4"`
	TextCenter           string         `json:"textCenter,omitempty" cb:"textCenter,5"`
	TextRight            string         `json:"textRight,omitempty" cb:"textRight,6"`
	TextLines            []TextLine     `json:"textLines,omitempty" cb:"textLines,7"`
	Image                string         `json:"image,omitempty" cb:"image,8"`
	SecondaryImagen      string         `json:"secondaryImagen,omitempty" cb:"secondaryImagen,9"`
	IconImagen           string         `json:"iconImagen,omitempty" cb:"iconImagen,10"`
	BgImage              string         `json:"bgImage,omitempty" cb:"bgImage,11"`
	VideoUrl             string         `json:"videoUrl,omitempty" cb:"videoUrl,12"`
	ProductIDs           []int32        `json:"productIDs,omitempty" cb:"productIDs,13"`
	CategoryIDs          []int32        `json:"categoryIDs,omitempty" cb:"categoryIDs,14"`
	BrandIDs             []int32        `json:"brandIDs,omitempty" cb:"brandIDs,15"`
	Gallery              []GalleryImage `json:"gallery,omitempty" cb:"gallery,16"`
	Limit                int32          `json:"limit,omitempty" cb:"limit,17"`
	PrimaryActionLabel   string         `json:"primaryActionLabel,omitempty" cb:"primaryActionLabel,18"`
	PrimaryActionHref    string         `json:"primaryActionHref,omitempty" cb:"primaryActionHref,19"`
	SecondaryActionLabel string         `json:"secondaryActionLabel,omitempty" cb:"secondaryActionLabel,20"`
	SecondaryActionHref  string         `json:"secondaryActionHref,omitempty" cb:"secondaryActionHref,21"`
}

// SectionContent mirrors the persisted fields of the frontend SectionData
// (renderer/section-types.ts). Top-level fields are PascalCase to match the
// frontend rename; nested frontend-owned trees (Ast/Content) are strict typed
// structs. The ORM stores the whole struct as a colbin blob.
type SectionContent struct {
	Type       string            `json:",omitempty" cb:"Type,1"`
	Ast        []AstNode         `json:",omitempty" cb:"Ast,2"`
	Content    *ContentFields    `json:",omitempty" cb:"Content,3"`
	Css        map[string]string `json:",omitempty" cb:"Css,4"`
	Attributes map[string]any    `json:",omitempty" cb:"Attributes,5"`
	// PageCss is the section's pre-generated runtime Tailwind CSS (the UnoCSS
	// output for its tokens, wrapped in @layer ec-runtime). The builder generates
	// it on save so the storefront concatenates the stored stylesheets instead of
	// running the UnoCSS engine at view time. It rides in the content blob so it is
	// covered by the section hash.
	PageCss string `json:",omitempty" cb:"PageCss,6"`
	// Svgs deduplicates the inline SVG bodies of icons picked in the builder, keyed by
	// sprite id `icon--<set>-<name>`. The frontend renders one <symbol> per entry and each
	// Icon AST node references it via <use href="#id">, so each body is stored exactly once.
	Svgs map[string]string `json:",omitempty" cb:"Svgs,7"`
	// Palette is the page's growable color list (hex colors, referenced 1-based as
	// var(--color-N)). Page-global, so it rides on section 1 only — same convention as
	// the whole-page Css. The builder grows it as the agent introduces new colors.
	Palette []string `json:",omitempty" cb:"Palette,8"`
	// CustomCss is the agent-authored raw CSS for this section, already scoped to
	// page-unique `.x{n}` classes by the builder. It rides in the content blob (so
	// it round-trips into the editor) and is also folded into the whole-page Css
	// column on save so the storefront serves it without regenerating.
	CustomCss string `json:",omitempty" cb:"CustomCss,9"`
}

// EcommercePageContent stores one builder section, addressed by its page and its
// 1-based position (SectionID). Dedup is by Hash (FNV-1a 64 of the section JSON),
// computed server-side so unchanged sections are skipped on save. Removed
// positions are soft-deleted (Status=0) since the ORM has no hard delete.
type EcommercePageContent struct {
	db.TableStruct[EcommercePageContentTable, EcommercePageContent]
	CompanyID int32          `json:",omitempty" cb:"CompanyID,1"`
	PageID    int16          `json:",omitempty" cb:"PageID,2"`
	SectionID int16          `json:",omitempty" cb:"SectionID,3"`
	Route     string         `json:",omitempty" cb:"Route,4"`
	Content   SectionContent `json:",omitempty" cb:"Content,5"`
	// Css holds the whole-page pre-generated runtime Tailwind CSS (the UnoCSS output
	// for every section's tokens). It is stored only on section 1 so the storefront
	// serves a single stylesheet from a plain column — no colbin decode of Content
	// needed. The builder ships it inside section 1's SectionContent.PageCss on save;
	// the handler moves it here and clears PageCss to avoid storing two copies.
	Css       string `json:",omitempty" cb:"Css,6"`
	Hash      int64  `json:",omitempty" cb:"Hash,7"`
	Status    int8   `json:"ss,omitempty" cb:"ss,8"`
	Updated   int32  `json:"upd,omitempty" cb:"upd,9"`
	UpdatedBy int32  `json:",omitempty" cb:"UpdatedBy,10"`
}

type EcommercePageContentTable struct {
	db.TableStruct[EcommercePageContentTable, EcommercePageContent]
	CompanyID db.Col[*EcommercePageContentTable, int32]
	PageID    db.Col[*EcommercePageContentTable, int16]
	SectionID db.Col[*EcommercePageContentTable, int16]
	Route     db.Col[*EcommercePageContentTable, string]
	Content   db.Col[*EcommercePageContentTable, SectionContent]
	Css       db.Col[*EcommercePageContentTable, string]
	Hash      db.Col[*EcommercePageContentTable, int64]
	Status    db.Col[*EcommercePageContentTable, int8]
	Updated   db.Col[*EcommercePageContentTable, int32]
	UpdatedBy db.Col[*EcommercePageContentTable, int32]
}

func (e EcommercePageContentTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        12,
		Name:      "ecommerce_page_content",
		Partition: e.CompanyID,
		// (CompanyID, PageID, SectionID) is the PK. SectionID is the section's
		// 1-based position on the page, recomputed on every save.
		Keys: db.Cols(e.PageID, e.SectionID),
	}
}

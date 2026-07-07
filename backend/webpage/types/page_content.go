package types

import (
	"app/db"
)

// TextLine mirrors the frontend ITextLine (renderer-types.ts): one editable line
// of rich text with its own Tailwind classes.
type TextLine struct {
	Text string `json:"text,omitempty"`
	Css  string `json:"css,omitempty"`
	Tag  string `json:"tag,omitempty"`
}

// GalleryImage mirrors the frontend IGalleryImagen.
type GalleryImage struct {
	Image       string `json:"image,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// AstNode mirrors the frontend ComponentAST (renderer-types.ts): one node of a
// parsed HTML section tree. The json tags keep the frontend's camelCase casing so
// the tree round-trips unchanged; colbin gives compact storage. `Props`
// stays a string-keyed map of coerced primitive component props (the frontend's
// Record<string, any>) — its values are primitives, so it decodes safely.
type AstNode struct {
	TagName    string            `json:"tagName,omitempty"`
	Css        string            `json:"css,omitempty"`
	Style      string            `json:"style,omitempty"`
	Text       string            `json:"text,omitempty"`
	Children   []AstNode         `json:"children,omitempty"`
	Role       string            `json:"role,omitempty"`
	Props      map[string]any    `json:"props,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// ContentFields mirrors the frontend StandardContent (section-types.ts): the flat
// content schema for component sections. The catch-all `[key]: any` is dropped —
// only these known fields are persisted.
type ContentFields struct {
	Title                string         `json:"title,omitempty"`
	SubTitle             string         `json:"subTitle,omitempty"`
	Description          string         `json:"description,omitempty"`
	TextLeft             string         `json:"textLeft,omitempty"`
	TextCenter           string         `json:"textCenter,omitempty"`
	TextRight            string         `json:"textRight,omitempty"`
	TextLines            []TextLine     `json:"textLines,omitempty"`
	Image                string         `json:"image,omitempty"`
	SecondaryImagen      string         `json:"secondaryImagen,omitempty"`
	IconImagen           string         `json:"iconImagen,omitempty"`
	BgImage              string         `json:"bgImage,omitempty"`
	VideoUrl             string         `json:"videoUrl,omitempty"`
	ProductIDs           []int32        `json:"productIDs,omitempty"`
	CategoryIDs          []int32        `json:"categoryIDs,omitempty"`
	BrandIDs             []int32        `json:"brandIDs,omitempty"`
	Gallery              []GalleryImage `json:"gallery,omitempty"`
	Limit                int32          `json:"limit,omitempty"`
	PrimaryActionLabel   string         `json:"primaryActionLabel,omitempty"`
	PrimaryActionHref    string         `json:"primaryActionHref,omitempty"`
	SecondaryActionLabel string         `json:"secondaryActionLabel,omitempty"`
	SecondaryActionHref  string         `json:"secondaryActionHref,omitempty"`
}

// SectionContent mirrors the persisted fields of the frontend SectionData
// (renderer/section-types.ts). Top-level fields are PascalCase to match the
// frontend rename; nested frontend-owned trees (Ast/Content) are strict typed
// structs. The ORM stores the whole struct as a colbin blob.
type SectionContent struct {
	Type       string            `json:",omitempty"`
	Ast        []AstNode         `json:",omitempty"`
	Content    *ContentFields    `json:",omitempty"`
	Css        map[string]string `json:",omitempty"`
	Attributes map[string]any    `json:",omitempty"`
	// PageCss is the section's pre-generated runtime Tailwind CSS (the UnoCSS
	// output for its tokens, wrapped in @layer ec-runtime). The builder generates
	// it on save so the storefront concatenates the stored stylesheets instead of
	// running the UnoCSS engine at view time. It rides in the content blob so it is
	// covered by the section hash.
	PageCss string `json:",omitempty"`
	// Svgs deduplicates the inline SVG bodies of icons picked in the builder, keyed by
	// sprite id `icon--<set>-<name>`. The frontend renders one <symbol> per entry and each
	// Icon AST node references it via <use href="#id">, so each body is stored exactly once.
	Svgs map[string]string `json:",omitempty"`
	// Palette is the page's growable color list (hex colors, referenced 1-based as
	// var(--color-N)). Page-global, so it rides on section 1 only — same convention as
	// the whole-page Css. The builder grows it as the agent introduces new colors.
	Palette []string `json:",omitempty"`
	// CustomCss is the agent-authored raw CSS for this section, already scoped to
	// page-unique `.x{n}` classes by the builder. It rides in the content blob (so
	// it round-trips into the editor) and is also folded into the whole-page Css
	// column on save so the storefront serves it without regenerating.
	CustomCss string `json:",omitempty"`
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
	Css       string `json:",omitempty"`
	Hash      int64  `json:",omitempty"`
	Status    int8   `json:"ss,omitempty"`
	Updated   int32  `json:"upd,omitempty"`
	UpdatedBy int32  `json:",omitempty"`
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

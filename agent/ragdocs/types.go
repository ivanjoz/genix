package ragdocs

// Frontmatter is the machine-readable route identity at the top of DOCUMENTATION.md.
type Frontmatter struct {
	Schema     int    `yaml:"schema"`
	PageID     string `yaml:"page_id"`
	Route      string `yaml:"route"`
	Title      string `yaml:"title"`
	Status     string `yaml:"status"`
	Visibility string `yaml:"visibility"`
}

// EvidenceFile links retrievable claims to the exact source bytes reviewed by the writer.
type EvidenceFile struct {
	Path     string   `yaml:"path"`
	Role     string   `yaml:"role"`
	Hash     string   `yaml:"hash"`
	Supports []string `yaml:"supports"`
}

type EvidenceManifest struct {
	Schema        int            `yaml:"schema"`
	HashAlgorithm string         `yaml:"hash_algorithm"`
	Files         []EvidenceFile `yaml:"files"`
}

// Section is one independently retrievable DOC-ID unit before size-based splitting.
type Section struct {
	ID       string
	Title    string
	Type     string
	Markdown string
}

// Document is a validated source document ready for deterministic chunk construction.
type Document struct {
	RepositoryPath    string
	Frontmatter       Frontmatter
	Evidence          EvidenceManifest
	Sections          []Section
	FileHash          string
	DocumentationHash string
	SourceHashDigest  string
}

// Chunk is the exact unit stored and independently retrieved from Qdrant.
type Chunk struct {
	PointID       string
	PointKey      string
	SectionID     string
	SectionTitle  string
	SectionType   string
	PartIndex     int
	PartCount     int
	Content       string
	ContextHeader string
	EmbeddingText string
	ContentHash   string
	Keywords      []string
}

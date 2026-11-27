// export-functions prints the data function catalog as JSON, the input of genix-local-ai's functions
// index: `go run ./cmd/export-functions > functions.json`. Generated, never hand-edited.
package main

import (
	"encoding/json"
	"log"
	"os"

	"genix-agent/datafunctions"
)

// exportedFunction is one function as `search` finds it.
type exportedFunction struct {
	Name        string   `json:"name"`
	Domain      string   `json:"domain"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Signature   string   `json:"signature"`
	Examples    []string `json:"examples"`
	Access      string   `json:"access"`
}

func main() {
	exported := make([]exportedFunction, len(datafunctions.Functions))
	for i, function := range datafunctions.Functions {
		exported[i] = exportedFunction{Name: function.Name, Domain: function.Domain, Title: function.Title, Description: function.Description,
			Signature: function.Signature(), Examples: function.Examples, Access: function.Access}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(exported); err != nil {
		log.Fatal(err)
	}
}

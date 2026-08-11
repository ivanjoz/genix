package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"app/agent/embedding"
	"app/agent/knowledge"
	"app/core"
)

var spanishExampleQuestions = []string{
	"¿Dónde puedo crear una caja o una cuenta bancaria?",
	"¿Cómo registro un ingreso o egreso manual en caja?",
	"¿Qué ocurre si hay una diferencia cuando hago el arqueo de caja?",
	"¿Qué datos necesito para crear una orden de compra?",
	"¿Puedo pagarle al proveedor desde una orden de compra y qué necesito?",
	"¿Se puede editar o cancelar una orden de compra después de confirmarla?",
}

type searchOutput struct {
	Question string                          `json:"question"`
	Results  []knowledge.DocumentationResult `json:"results"`
}

func main() {
	if err := run(); err != nil {
		log.Printf("[agent.knowledge] documentation_search_failed error=%v", err)
		os.Exit(1)
	}
}

func run() error {
	question := flag.String("question", "", "Spanish or mixed-language user question")
	runExamples := flag.Bool("examples", false, "run the built-in Spanish evaluation questions")
	jsonOutput := flag.Bool("json", false, "print complete structured JSON results")
	resultLimit := flag.Uint64("limit", 5, "maximum fused results per question")
	candidateLimit := flag.Uint64("candidates", 25, "dense and BM25 candidates to prefetch")
	route := flag.String("route", "", "optional exact route filter")
	module := flag.String("module", "", "optional exact module filter")
	visibility := flag.String("visibility", "", "optional exact visibility filter")
	qdrantHost := flag.String("qdrant-host", "", "optional qdrant.host override without modifying config.toml")
	flag.Parse()

	questions, err := selectedQuestions(*question, *runExamples)
	if err != nil {
		return err
	}
	core.PopulateVariables()
	if strings.TrimSpace(*qdrantHost) != "" {
		core.Env.QDRANT_HOST = strings.TrimSpace(*qdrantHost)
	}
	qdrantConfig, err := knowledge.ConfigFromEnv()
	if err != nil {
		return err
	}
	qdrantStore, err := knowledge.NewStore(qdrantConfig)
	if err != nil {
		return err
	}
	defer qdrantStore.Close()
	if err := qdrantStore.ValidateExistingCollection(context.Background()); err != nil {
		return err
	}
	embeddingClient, err := embedding.NewClientFromEnv()
	if err != nil {
		return err
	}
	retriever, err := knowledge.NewRetriever(embeddingClient, qdrantStore)
	if err != nil {
		return err
	}

	outputs := make([]searchOutput, 0, len(questions))
	for questionIndex, selectedQuestion := range questions {
		results, err := retriever.SearchDocumentation(context.Background(), selectedQuestion, knowledge.SearchOptions{
			CandidateLimit: *candidateLimit,
			ResultLimit:    *resultLimit,
			Route:          *route,
			Module:         *module,
			Visibility:     *visibility,
		})
		if err != nil {
			return fmt.Errorf("search question %d: %w", questionIndex+1, err)
		}
		outputs = append(outputs, searchOutput{Question: selectedQuestion, Results: results})
	}
	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(outputs)
	}
	printReadableResults(outputs)
	return nil
}

func selectedQuestions(question string, examples bool) ([]string, error) {
	question = strings.TrimSpace(question)
	if examples && question != "" {
		return nil, errors.New("use either -question or -examples, not both")
	}
	if examples {
		return spanishExampleQuestions, nil
	}
	if question == "" {
		return nil, errors.New("provide -question or use -examples")
	}
	return []string{question}, nil
}

func printReadableResults(outputs []searchOutput) {
	for _, output := range outputs {
		fmt.Printf("\nPREGUNTA: %s\n", output.Question)
		if len(output.Results) == 0 {
			fmt.Println("  Sin resultados documentados.")
			continue
		}
		for resultIndex, result := range output.Results {
			preview := strings.Join(strings.Fields(result.Content), " ")
			previewRunes := []rune(preview)
			if len(previewRunes) > 260 {
				preview = string(previewRunes[:260]) + "…"
			}
			fmt.Printf("  %d. score=%.4f route=%s citation=%s\n     %s\n     %s\n",
				resultIndex+1, result.Score, result.Route, result.CitationID, result.SectionTitle, preview)
		}
	}
}

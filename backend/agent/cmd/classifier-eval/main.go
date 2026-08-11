package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"app/agent/llm"
	"app/agent/routing"
	"app/core"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "classifier evaluation failed:", err)
		os.Exit(1)
	}
}

func run() error {
	question := flag.String("question", "", "user question to classify")
	surfaceKind := flag.String("surface", string(routing.SurfaceUnknown), "frontend surface kind")
	route := flag.String("route", "", "current SPA route")
	appLanguage := flag.String("language", string(routing.LanguageSpanish), "app language: es or en")
	selectedSectionID := flag.String("selected-section", "", "optional selected builder section ID")
	previousUserMessage := flag.String("previous-user", "", "optional previous completed user message")
	previousAssistantMessage := flag.String("previous-assistant", "", "assistant reply paired with -previous-user")
	previousRoute := flag.String("previous-route", "", "route associated with the previous completed turn")
	flag.Parse()
	if strings.TrimSpace(*question) == "" {
		return errors.New("-question is required")
	}

	core.PopulateVariables()
	chatClient, err := llm.NewClientForProvider(core.Env.CLASSIFIER_PROVIDER, core.Env.CLASSIFIER_MODEL_ID)
	if err != nil {
		return err
	}
	classifier, err := routing.NewClassifier(chatClient, core.Env.CLASSIFIER_MODEL_ID)
	if err != nil {
		return err
	}
	surface := routing.SurfaceContext{
		Kind: routing.SurfaceKind(*surfaceKind), Route: strings.TrimSpace(*route),
		HasSelectedSection: strings.TrimSpace(*selectedSectionID) != "", SelectedSectionID: strings.TrimSpace(*selectedSectionID),
	}
	if surface.Kind == routing.SurfaceWebpageEditor {
		surface.PageID = strings.TrimPrefix(surface.Route, "/webpage-builder/")
		surface.AvailableContexts = []routing.BuilderContextScope{routing.BuilderScopeFullPage}
		if surface.HasSelectedSection {
			surface.SelectedSectionType = "HtmlSection"
			surface.AvailableContexts = append(surface.AvailableContexts, routing.BuilderScopeSelectedSection)
		}
	}
	input := routing.ClassifierInput{
		Schema: routing.SchemaVersion, CurrentMessage: strings.TrimSpace(*question), Surface: surface,
		Route: strings.TrimSpace(*route), Capabilities: routing.CapabilitySnapshot(), AppLanguage: routing.Language(*appLanguage),
	}
	if strings.TrimSpace(*previousUserMessage) != "" || strings.TrimSpace(*previousAssistantMessage) != "" {
		if strings.TrimSpace(*previousUserMessage) == "" || strings.TrimSpace(*previousAssistantMessage) == "" {
			return errors.New("-previous-user and -previous-assistant must be provided together")
		}
		input.CompletedTurns = []routing.CompletedTurn{{
			Offset: -1, UserMessage: strings.TrimSpace(*previousUserMessage),
			AssistantMessage: strings.TrimSpace(*previousAssistantMessage), Route: strings.TrimSpace(*previousRoute),
		}}
	}
	verdict, err := classifier.Classify(context.Background(), input)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(verdict)
}

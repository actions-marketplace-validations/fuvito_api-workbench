package main

import (
	_ "embed"
	"strings"
)

//go:embed help/user-guide.md
var userGuideMarkdown string

//go:embed help/format-spec.md
var formatSpecMarkdown string

//go:embed help/example-scenario.md
var exampleScenarioMarkdown string

//go:embed help/ai-prompt.md
var aiPromptMarkdown string

// whatWeCheckMarkdown is the catalog of what Clear Signal checks and does not check. The file is also
// the document the repository links to, and tests keep it in step with the code.
//
//go:embed help/what-we-check.md
var whatWeCheckMarkdown string

type HelpService struct{}

func NewHelpService() *HelpService { return &HelpService{} }

func (h *HelpService) GetUserGuide() string { return userGuideMarkdown }

func (h *HelpService) GetFormatSpec() string { return formatSpecMarkdown }

func (h *HelpService) GetExampleScenario() string { return exampleScenarioMarkdown }

// GetWhatWeCheck returns the catalog of checks, as shown in the Help panel's What We Check tab.
func (h *HelpService) GetWhatWeCheck() string { return whatWeCheckMarkdown }

// GetAIPromptInstructions returns the AI prompt document as shown in the Help panel.
func (h *HelpService) GetAIPromptInstructions() string { return aiPromptMarkdown }

// GetAIPrompt returns the complete prompt to paste into an AI assistant: the instructions
// plus the format specification and the example scenario they refer to.
func (h *HelpService) GetAIPrompt() string {
	return buildAIPrompt(aiPromptMarkdown, formatSpecMarkdown, exampleScenarioMarkdown)
}

// buildAIPrompt joins the three documents. The human-facing introduction of the
// instructions (everything before the first horizontal rule) is left out, because
// it describes the UI rather than telling the assistant what to do.
func buildAIPrompt(instructions, spec, example string) string {
	lf := func(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }
	instructions, spec, example = lf(instructions), lf(spec), lf(example)
	if _, rest, found := strings.Cut(instructions, "\n---\n"); found {
		instructions = rest
	}
	var b strings.Builder
	b.WriteString(strings.TrimSpace(instructions))
	b.WriteString("\n\n---\n\n# Reference 1 — Format Specification\n\n")
	b.WriteString(strings.TrimSpace(spec))
	b.WriteString("\n\n---\n\n# Reference 2 — Example Scenario\n\n")
	b.WriteString(strings.TrimSpace(example))
	b.WriteString("\n")
	return b.String()
}

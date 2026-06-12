package enrich

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// LangChainProvider implements Provider over any OpenAI-compatible endpoint via
// langchaingo. The endpoint, credential, and model are injected at construction;
// the model is asked for a JSON object matching the Enrichment contract.
type LangChainProvider struct {
	llm llms.Model
}

// NewLangChainProvider builds a provider against an OpenAI-compatible endpoint.
// baseURL points at the gateway/provider (e.g. a LiteLLM endpoint), apiKey is the
// bearer credential, model is the model id to call. No provider is hard-coded —
// any OpenAI-compatible backend works.
func NewLangChainProvider(baseURL, apiKey, model string) (*LangChainProvider, error) {
	llm, err := openai.New(
		openai.WithBaseURL(baseURL),
		openai.WithToken(apiKey),
		openai.WithModel(model),
	)
	if err != nil {
		return nil, fmt.Errorf("enrich: build llm client: %w", err)
	}
	return &LangChainProvider{llm: llm}, nil
}

// Enrich asks the model for a structured Enrichment for the job and parses the JSON
// response. It does not validate the result — the caller validates before persisting.
func (p *LangChainProvider) Enrich(ctx context.Context, job JobInput) (Enrichment, error) {
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt(job)),
	}
	resp, err := p.llm.GenerateContent(ctx, messages, llms.WithJSONMode())
	if err != nil {
		return Enrichment{}, fmt.Errorf("enrich: generate: %w", err)
	}
	if len(resp.Choices) == 0 {
		return Enrichment{}, fmt.Errorf("enrich: model returned no choices")
	}
	return parseEnrichment(resp.Choices[0].Content)
}

// parseEnrichment unmarshals a model's JSON response into an Enrichment, tolerating
// a markdown code fence some models add despite JSON mode.
func parseEnrichment(raw string) (Enrichment, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)

	var e Enrichment
	if err := json.Unmarshal([]byte(s), &e); err != nil {
		return Enrichment{}, fmt.Errorf("enrich: parse response: %w", err)
	}
	return e, nil
}

// systemPrompt instructs the model to emit only stated fields and to draw enum
// values from the controlled vocabularies — the same lists Validate enforces, so
// the prompt and the validator can never drift.
var systemPrompt = buildSystemPrompt()

func buildSystemPrompt() string {
	var b strings.Builder
	b.WriteString("You extract structured facts from an IT job posting and return ONLY a JSON object.\n")
	b.WriteString("Include a key only when the posting clearly states it; omit anything not stated. Never guess.\n")
	b.WriteString("Enum fields MUST use exactly one of the allowed values below.\n\n")
	b.WriteString("Allowed enum values:\n")

	enum := func(field string, vals []string) {
		fmt.Fprintf(&b, "- %s: %s\n", field, strings.Join(vals, ", "))
	}
	enum("work_mode", WorkModeValues)
	enum("regions (array)", RegionValues)
	enum("employment_type", EmploymentTypeValues)
	enum("relocation", RelocationValues)
	enum("salary_period", SalaryPeriodValues)
	enum("seniority", SeniorityValues)
	enum("english_level", EnglishLevelValues)
	enum("education_level", EducationLevelValues)
	enum("category", CategoryValues)
	enum("domains (array)", DomainValues)
	enum("company_type", CompanyTypeValues)
	enum("company_size", CompanySizeValues)

	b.WriteString("\nOther keys (omit when unstated): ")
	b.WriteString("visa_sponsorship (boolean), countries (array of ISO 3166-1 alpha-2), ")
	b.WriteString("cities (array of strings), timezone_note (string), ")
	b.WriteString("salary_min (int), salary_max (int), salary_currency (ISO 4217), ")
	b.WriteString("experience_years_min (non-negative int), ")
	b.WriteString("skills (array of lowercase tokens, e.g. go, postgresql), ")
	b.WriteString("posting_language (ISO 639-1, e.g. en, uk, ru).\n")

	b.WriteString("\nregions is the remote role's reach (only when work_mode is remote; omit otherwise): ")
	b.WriteString("use 'global' ONLY when the posting explicitly says the role is open worldwide / ")
	b.WriteString("anywhere / from any country; otherwise list the region(s) or country code(s) ")
	b.WriteString("the role is open to, from the allowed values. Omit when unstated (unknown is not global).\n")
	return b.String()
}

func userPrompt(job JobInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Title: %s\n", job.Title)
	fmt.Fprintf(&b, "Company: %s\n", job.Company)
	fmt.Fprintf(&b, "Location: %s\n", job.Location)
	// Source-provided remote hint (from the ATS API or the location text) — a
	// prior for the model, not a guarantee of scope.
	fmt.Fprintf(&b, "Remote flag: %t\n", job.Remote)
	fmt.Fprintf(&b, "Description:\n%s\n", job.Description)
	return b.String()
}

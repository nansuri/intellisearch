package services

import (
	"strings"

	"intellisearch/internal/repositories"
)

// ApiDocsService reads the seeded Mini Apps API documentation from the
// database and shapes it for the Studio: grouped sections for rendering plus a
// flat markdown export (including the AI ask reference) for download.
type ApiDocsService struct {
	repo *repositories.ApiDocRepository
}

func NewApiDocsService(repo *repositories.ApiDocRepository) *ApiDocsService {
	return &ApiDocsService{repo: repo}
}

// DocEntry is one documented API entry grouped under a section.
type DocEntry struct {
	Title    string `json:"title"`
	Method   string `json:"method,omitempty"`
	Path     string `json:"path,omitempty"`
	Markdown string `json:"markdown"`
}

// DocSection is a group of related documented entries (e.g. "AI (Ask)").
type DocSection struct {
	Section string     `json:"section"`
	Entries []DocEntry `json:"entries"`
}

// Groups returns the documentation grouped by section in storage order.
func (s *ApiDocsService) Groups() ([]DocSection, error) {
	docs, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	var sections []DocSection
	var current *DocSection
	for _, doc := range docs {
		if current == nil || current.Section != doc.Section {
			sections = append(sections, DocSection{Section: doc.Section})
			current = &sections[len(sections)-1]
		}
		current.Entries = append(current.Entries, DocEntry{Title: doc.Title, Method: doc.Method, Path: doc.Path, Markdown: doc.Markdown})
	}
	if sections == nil {
		// Never return a null array to the frontend when the table is empty.
		sections = []DocSection{}
	}
	return sections, nil
}

// Markdown builds the full Mini Apps API reference as a single markdown
// document — this is the "download the AI API as markdown" export. It is a
// static composition of the stored rows, so it always matches the rendered API
// list.
func (s *ApiDocsService) Markdown() (string, error) {
	docs, err := s.repo.List()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("# Mini Apps API\n\n")
	b.WriteString("This site lets you build mini apps from plain HTML + CSS + JS and run them in a sandboxed, same-origin iframe. Your app code can call the platform API below, including the AI question pipeline (which reuses the shared AI handler, queue, rate limits, and per-user daily quota).\n\n---\n\n")
	currentSection := ""
	for _, doc := range docs {
		if doc.Section != currentSection {
			b.WriteString("\n## " + doc.Section + "\n\n")
			currentSection = doc.Section
		}
		heading := doc.Title
		if doc.Method != "" && doc.Path != "" {
			heading = doc.Method + " `" + doc.Path + "`"
		}
		b.WriteString("### " + heading + "\n\n")
		b.WriteString(doc.Markdown + "\n\n")
	}
	return b.String(), nil
}
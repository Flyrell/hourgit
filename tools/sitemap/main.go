package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const baseURL = "https://hourgit.com"

// priorities maps URL paths to their sitemap priority.
var priorities = map[string]string{
	"/":                                 "1.0",
	"/docs/":                            "0.8",
	"/docs/installation":                "0.8",
	"/docs/quick-start":                 "0.8",
	"/docs/configuration":               "0.7",
	"/docs/data-storage":                "0.6",
	"/docs/commands/time-tracking":      "0.7",
	"/docs/commands/project-management": "0.7",
	"/docs/commands/schedule":           "0.6",
	"/docs/commands/defaults":           "0.6",
	"/docs/commands/shell-completions":  "0.5",
	"/docs/commands/utility":            "0.5",
}

type urlEntry struct {
	XMLName    xml.Name `xml:"url"`
	Loc        string   `xml:"loc"`
	LastMod    string   `xml:"lastmod"`
	ChangeFreq string   `xml:"changefreq"`
	Priority   string   `xml:"priority"`
}

type urlSet struct {
	XMLName xml.Name   `xml:"urlset"`
	XMLNS   string     `xml:"xmlns,attr"`
	URLs    []urlEntry `xml:"url"`
}

func main() {
	docsDir := flag.String("docs", "web/docs", "path to docs directory")
	outPath := flag.String("out", "web/sitemap.xml", "output path for sitemap.xml")
	flag.Parse()

	sidebarPath := filepath.Join(*docsDir, "_sidebar.md")
	sidebarData, err := os.ReadFile(sidebarPath)
	if err != nil {
		fatal("reading sidebar: %v", err)
	}

	paths := parseSidebar(string(sidebarData))
	today := time.Now().Format("2006-01-02")

	var urls []urlEntry

	// Homepage entry
	urls = append(urls, makeEntry("/", today))

	// Docs entries from sidebar
	for _, p := range paths {
		urlPath := mdToURLPath(p)
		urls = append(urls, makeEntry("/docs/"+urlPath, today))
	}

	sitemap := urlSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	output, err := xml.MarshalIndent(sitemap, "", "  ")
	if err != nil {
		fatal("marshalling XML: %v", err)
	}

	content := xml.Header + string(output) + "\n"

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fatal("creating output directory: %v", err)
	}

	if err := os.WriteFile(*outPath, []byte(content), 0o644); err != nil {
		fatal("writing %s: %v", *outPath, err)
	}

	fmt.Printf("  sitemap: %d URLs written to %s\n", len(urls), *outPath)
}

// parseSidebar extracts markdown file paths from _sidebar.md, skipping back links.
func parseSidebar(content string) []string {
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	var paths []string

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed == "-" {
			continue
		}

		// Skip back link
		if strings.Contains(trimmed, "←") {
			continue
		}

		// Only indented links are nav items
		if !strings.HasPrefix(line, "  ") {
			continue
		}

		if m := linkRe.FindStringSubmatch(trimmed); m != nil {
			paths = append(paths, m[2])
		}
	}

	return paths
}

// mdToURLPath converts a .md sidebar path to a URL path segment.
// README.md → "" (empty, meaning directory index)
// foo.md → "foo"
// commands/foo.md → "commands/foo"
func mdToURLPath(mdPath string) string {
	dir := filepath.Dir(mdPath)
	base := filepath.Base(mdPath)

	if strings.EqualFold(base, "README.md") {
		if dir == "." {
			return ""
		}
		return dir + "/"
	}

	name := strings.TrimSuffix(base, ".md")
	if dir == "." {
		return name
	}
	return dir + "/" + name
}

// makeEntry creates a sitemap URL entry with priority and changefreq from the map.
func makeEntry(path, lastmod string) urlEntry {
	prio, ok := priorities[path]
	if !ok {
		prio = "0.5"
	}

	changefreq := "monthly"
	if prio == "1.0" || prio == "0.8" {
		changefreq = "weekly"
	}

	return urlEntry{
		Loc:        baseURL + path,
		LastMod:    lastmod,
		ChangeFreq: changefreq,
		Priority:   prio,
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "sitemap: "+format+"\n", args...)
	os.Exit(1)
}

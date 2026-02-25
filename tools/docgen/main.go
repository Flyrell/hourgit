package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// NavGroup is a sidebar navigation group with a label and items.
type NavGroup struct {
	Label string
	Items []NavItem
}

// NavItem is a single sidebar navigation link.
type NavItem struct {
	Title string
	Path  string // original .md path from sidebar
}

// PageData is the template data for rendering a docs page.
type PageData struct {
	Title   string
	Sidebar template.HTML
	Content template.HTML
	CSSPath string
	RootPath string
}

func main() {
	docsDir := flag.String("docs", "web/docs", "path to docs directory")
	tmplPath := flag.String("template", "", "path to template (default: <docs>/_template.html)")
	flag.Parse()

	if *tmplPath == "" {
		*tmplPath = filepath.Join(*docsDir, "_template.html")
	}

	// Parse sidebar
	sidebarPath := filepath.Join(*docsDir, "_sidebar.md")
	sidebarData, err := os.ReadFile(sidebarPath)
	if err != nil {
		fatal("reading sidebar: %v", err)
	}
	groups, backLink := parseSidebar(string(sidebarData))

	// Read template
	tmplData, err := os.ReadFile(*tmplPath)
	if err != nil {
		fatal("reading template: %v", err)
	}
	tmpl, err := template.New("page").Parse(string(tmplData))
	if err != nil {
		fatal("parsing template: %v", err)
	}

	// Set up goldmark
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Linkify,
			extension.Strikethrough,
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
				highlighting.WithFormatOptions(),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	// Collect all pages from sidebar
	var pages []NavItem
	for _, g := range groups {
		pages = append(pages, g.Items...)
	}

	// Generate each page
	for _, page := range pages {
		mdPath := filepath.Join(*docsDir, page.Path)
		mdData, err := os.ReadFile(mdPath)
		if err != nil {
			fatal("reading %s: %v", mdPath, err)
		}

		// Extract title from first heading
		title := extractTitle(string(mdData))
		if title == "" {
			title = page.Title
		}

		// Render markdown to HTML
		var contentBuf bytes.Buffer
		if err := md.Convert(mdData, &contentBuf); err != nil {
			fatal("converting %s: %v", page.Path, err)
		}

		// Rewrite .md links to .html in rendered content
		contentHTML := rewriteLinks(contentBuf.String())

		// Determine output path
		outPath := mdToHTMLPath(page.Path)
		outFile := filepath.Join(*docsDir, outPath)

		// Determine relative paths for CSS and root
		cssPath, rootPath := relativePaths(outPath)

		// Render sidebar HTML
		sidebarHTML := renderSidebar(groups, backLink, page.Path, rootPath)

		// Render full page
		data := PageData{
			Title:   title,
			Sidebar: template.HTML(sidebarHTML),
			Content: template.HTML(contentHTML),
			CSSPath: cssPath,
			RootPath: rootPath,
		}

		var pageBuf bytes.Buffer
		if err := tmpl.Execute(&pageBuf, data); err != nil {
			fatal("executing template for %s: %v", page.Path, err)
		}

		// Ensure output directory exists
		if err := os.MkdirAll(filepath.Dir(outFile), 0o755); err != nil {
			fatal("creating directory for %s: %v", outFile, err)
		}

		if err := os.WriteFile(outFile, pageBuf.Bytes(), 0o644); err != nil {
			fatal("writing %s: %v", outFile, err)
		}

		fmt.Printf("  generated %s\n", outPath)
	}

	fmt.Printf("\n  %d pages generated\n", len(pages))
}

// parseSidebar parses the _sidebar.md format into nav groups and extracts the back link.
func parseSidebar(content string) ([]NavGroup, string) {
	var groups []NavGroup
	var backLink string

	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	boldRe := regexp.MustCompile(`\*\*([^*]+)\*\*`)

	var currentGroup *NavGroup

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed == "-" {
			continue
		}

		// Detect indentation level (group label vs nav item)
		isIndented := strings.HasPrefix(line, "  ")

		// Check for back link (contains ←)
		if strings.Contains(trimmed, "←") {
			if m := linkRe.FindStringSubmatch(trimmed); m != nil {
				backLink = m[2]
			}
			continue
		}

		// Check for bold group label
		if m := boldRe.FindStringSubmatch(trimmed); m != nil && !isIndented {
			if currentGroup != nil {
				groups = append(groups, *currentGroup)
			}
			currentGroup = &NavGroup{Label: m[1]}
			continue
		}

		// Check for nav item link
		if m := linkRe.FindStringSubmatch(trimmed); m != nil && isIndented {
			if currentGroup != nil {
				currentGroup.Items = append(currentGroup.Items, NavItem{
					Title: m[1],
					Path:  m[2],
				})
			}
			continue
		}
	}

	if currentGroup != nil {
		groups = append(groups, *currentGroup)
	}

	return groups, backLink
}

// extractTitle extracts the text from the first # heading in markdown.
func extractTitle(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimPrefix(trimmed, "# ")
		}
	}
	return ""
}

// mdToHTMLPath converts a .md path to the corresponding .html output path.
// Special case: README.md → index.html
func mdToHTMLPath(mdPath string) string {
	dir := filepath.Dir(mdPath)
	base := filepath.Base(mdPath)

	var htmlName string
	if strings.EqualFold(base, "README.md") {
		htmlName = "index.html"
	} else {
		htmlName = strings.TrimSuffix(base, ".md") + ".html"
	}

	if dir == "." {
		return htmlName
	}
	return filepath.Join(dir, htmlName)
}

// relativePaths computes CSSPath and RootPath for a given output file path.
// Pages in subdirectories (e.g. commands/foo.html) need "../" prefix.
func relativePaths(outPath string) (cssPath, rootPath string) {
	dir := filepath.Dir(outPath)
	if dir == "." {
		return "_docs.css", ""
	}
	// Count directory depth
	depth := strings.Count(filepath.ToSlash(dir), "/") + 1
	prefix := strings.Repeat("../", depth)
	return prefix + "_docs.css", prefix
}

// mdToLinkPath converts a .md path to the URL path used in href attributes.
// Links omit the .html extension (server resolves automatically).
// Special case: README.md maps to the directory root (empty basename).
func mdToLinkPath(mdPath string) string {
	htmlPath := mdToHTMLPath(mdPath)
	// index.html → "" (root of directory)
	if filepath.Base(htmlPath) == "index.html" {
		dir := filepath.Dir(htmlPath)
		if dir == "." {
			return ""
		}
		return dir + "/"
	}
	return strings.TrimSuffix(htmlPath, ".html")
}

// rewriteLinks replaces .md href references with extensionless paths in rendered HTML.
// Also handles README.md → directory root.
var linkHrefRe = regexp.MustCompile(`href="([^"]*\.md)(#[^"]*)?`)

func rewriteLinks(htmlContent string) string {
	return linkHrefRe.ReplaceAllStringFunc(htmlContent, func(match string) string {
		m := linkHrefRe.FindStringSubmatch(match)
		mdRef := m[1]
		fragment := ""
		if len(m) > 2 {
			fragment = m[2]
		}

		linkRef := mdToLinkPath(mdRef)

		return `href="` + linkRef + fragment
	})
}

// renderSidebar generates the sidebar nav HTML.
func renderSidebar(groups []NavGroup, backLink, currentPath, rootPath string) string {
	var b strings.Builder

	// Back link
	if backLink != "" {
		href := rootPath + backLink
		// Normalize ../ for back link
		if strings.HasPrefix(backLink, "../") {
			href = rootPath + backLink
		}
		b.WriteString(fmt.Sprintf(`<a href="%s" class="back-link">← Back to hourgit.com</a>`+"\n", href))
	}

	b.WriteString(`<nav class="sidebar-nav">` + "\n")

	for _, group := range groups {
		b.WriteString(fmt.Sprintf(`  <div class="nav-group-label">%s</div>`+"\n", group.Label))
		for _, item := range group.Items {
			linkPath := mdToLinkPath(item.Path)
			href := rootPath + linkPath

			activeClass := ""
			if item.Path == currentPath {
				activeClass = " active"
			}

			b.WriteString(fmt.Sprintf(`  <a href="%s" class="nav-link%s">%s</a>`+"\n", href, activeClass, item.Title))
		}
	}

	b.WriteString("</nav>\n")
	return b.String()
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "docgen: "+format+"\n", args...)
	os.Exit(1)
}

package pkg

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"path"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chroma_html "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/chrishrb/go-grip/defaults"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

var blockquotes = []string{"Note", "Tip", "Important", "Warning", "Caution", "BlockQuote"}

type Parser struct {
	theme string
}

func NewParser(theme string) *Parser {
	return &Parser{
		theme: theme,
	}
}

func (m Parser) MdToHTML(bytes []byte) []byte {
	// Preprocess markdown to add blank lines before lists for GFM-style behavior
	bytes = preprocessMarkdown(bytes)

	extensions := parser.NoIntraEmphasis | parser.Tables | parser.FencedCode |
		parser.Autolink | parser.Strikethrough | parser.SpaceHeadings | parser.HeadingIDs |
		parser.BackslashLineBreak | parser.MathJax | parser.OrderedListStart |
		parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(bytes)

	htmlFlags := html.CommonFlags
	opts := html.RendererOptions{Flags: htmlFlags, RenderNodeHook: m.renderHook}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}

func (m Parser) renderHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	switch node.(type) {
	case *ast.BlockQuote:
		return renderHookBlockQuote()
	case *ast.Paragraph:
		return renderHookParagraph(w, node, entering)
	case *ast.Text:
		return renderHookText(w, node)
	case *ast.ListItem:
		return renderHookListItem(w, node, entering)
	case *ast.CodeBlock:
		return renderHookCodeBlock(w, node, m.theme)
	}

	return ast.GoToNext, false
}

func renderHookCodeBlock(w io.Writer, node ast.Node, theme string) (ast.WalkStatus, bool) {
	block := node.(*ast.CodeBlock)

	if string(block.Info) == "mermaid" {
		m, err := renderMermaid(string(block.Literal), theme)
		if err != nil {
			log.Println("Error:", err)
		}
		fmt.Fprint(w, m)
		return ast.GoToNext, true
	}

	var lexer chroma.Lexer
	if block.Info == nil {
		lexer = lexers.Analyse(string(block.Literal))
	} else {
		lexer = lexers.Get(string(block.Info))
	}
	// ensure lexer is never nil
	if lexer == nil {
		lexer = lexers.Get("plaintext")
	}

	iterator, _ := lexer.Tokenise(nil, string(block.Literal))
	formatter := chroma_html.New(chroma_html.WithClasses(true))
	err := formatter.Format(w, styles.Fallback, iterator)
	if err != nil {
		log.Println("Error:", err)
	}
	return ast.GoToNext, true
}

func renderHookBlockQuote() (ast.WalkStatus, bool) {
	return ast.GoToNext, true
}

func renderHookParagraph(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	paragraph := node.(*ast.Paragraph)

	_, ok := paragraph.GetParent().(*ast.BlockQuote)
	if !ok {
		return ast.GoToNext, false
	}

	t, ok := (paragraph.GetChildren()[0]).(*ast.Text)
	if !ok {
		return ast.GoToNext, false
	}

	// Get the text content of the blockquote
	content := string(t.Literal)

	var alert string
	for _, b := range blockquotes {
		if strings.HasPrefix(content, fmt.Sprintf("[!%s]", strings.ToUpper(b))) {
			alert = strings.ToLower(b)
		}
	}

	if alert == "" {
		return ast.GoToNext, false
	}

	// Set the message type based on the content of the blockquote
	var err error
	if entering {
		var s string
		s, _ = createBlockquoteStart(alert)
		_, err = io.WriteString(w, s)
	} else {
		_, err = io.WriteString(w, "</div>")
	}
	if err != nil {
		log.Println("Error:", err)
	}

	return ast.GoToNext, true
}

func renderHookText(w io.Writer, node ast.Node) (ast.WalkStatus, bool) {
	block := node.(*ast.Text)

	r := regexp.MustCompile(`(:\S+:)`)
	withEmoji := r.ReplaceAllStringFunc(string(block.Literal), func(s string) string {
		val, ok := EmojiMap[s]
		if !ok {
			return s
		}

		if strings.HasPrefix(val, "/") {
			return fmt.Sprintf(`<img class="emoji" title="%s" alt="%s" src="%s" height="20" width="20" align="absmiddle">`, s, s, val)
		}

		return val
	})

	paragraph, ok := block.GetParent().(*ast.Paragraph)
	if !ok {
		_, err := io.WriteString(w, withEmoji)
		if err != nil {
			log.Println("Error:", err)
		}
		return ast.GoToNext, true
	}

	_, ok = paragraph.GetParent().(*ast.BlockQuote)
	if ok {
		// Remove prefixes
		for _, b := range blockquotes {
			content, found := strings.CutPrefix(withEmoji, fmt.Sprintf("[!%s]", strings.ToUpper(b)))
			if found {
				_, err := io.WriteString(w, content)
				if err != nil {
					log.Println("Error:", err)
				}
				return ast.GoToNext, true
			}
		}
	}

	_, ok = paragraph.GetParent().(*ast.ListItem)
	if ok {
		content, found := strings.CutPrefix(withEmoji, "[ ]")
		content = `<input type="checkbox" disabled class="task-list-item-checkbox"> ` + content
		if found {
			_, err := io.WriteString(w, content)
			if err != nil {
				log.Println("Error:", err)
			}
			return ast.GoToNext, true
		}

		content, found = strings.CutPrefix(withEmoji, "[x]")
		content = `<input type="checkbox" disabled class="task-list-item-checkbox" checked> ` + content
		if found {
			_, err := io.WriteString(w, content)
			if err != nil {
				log.Println("Error:", err)
			}
		}
	}

	_, err := io.WriteString(w, withEmoji)
	if err != nil {
		log.Println("Error:", err)
	}
	return ast.GoToNext, true
}

func renderHookListItem(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	block := node.(*ast.ListItem)

	paragraph, ok := (block.GetChildren()[0]).(*ast.Paragraph)
	if !ok {
		return ast.GoToNext, false
	}

	t, ok := (paragraph.GetChildren()[0]).(*ast.Text)
	if !ok {
		return ast.GoToNext, false
	}

	if !(strings.HasPrefix(string(t.Literal), "[ ]") || strings.HasPrefix(string(t.Literal), "[x]")) {
		return ast.GoToNext, false
	}

	if entering {
		_, err := io.WriteString(w, "<li class=\"task-list-item\">")
		if err != nil {
			log.Println("Error:", err)
		}
	} else {
		_, err := io.WriteString(w, "</li>")
		if err != nil {
			log.Println("Error:", err)
		}
	}

	return ast.GoToNext, true
}

func createBlockquoteStart(alert string) (string, error) {
	lp := path.Join("templates/alert", fmt.Sprintf("%s.html", alert))
	tmpl, err := template.ParseFS(defaults.Templates, lp)
	if err != nil {
		return "", err
	}
	var tpl bytes.Buffer
	if err := tmpl.Execute(&tpl, alert); err != nil {
		return "", err
	}
	return tpl.String(), nil
}

type mermaid struct {
	Content string
	Theme   string
}

func renderMermaid(content string, theme string) (string, error) {
	m := mermaid{
		Content: content,
		Theme:   theme,
	}
	lp := path.Join("templates/mermaid/mermaid.html")
	tmpl, err := template.ParseFS(defaults.Templates, lp)
	if err != nil {
		return "", err
	}
	var tpl bytes.Buffer
	if err := tmpl.Execute(&tpl, m); err != nil {
		return "", err
	}
	return tpl.String(), nil
}

// preprocessMarkdown adds blank lines before lists that don't have one
// to ensure they are rendered as lists (GFM-style behavior)
func preprocessMarkdown(md []byte) []byte {
	lines := bytes.Split(md, []byte("\n"))
	var result [][]byte
	inCodeBlock := false

	for i, line := range lines {
		// Track fenced code blocks to avoid modifying them
		trimmed := bytes.TrimSpace(line)
		if bytes.HasPrefix(trimmed, []byte("```")) || bytes.HasPrefix(trimmed, []byte("~~~")) {
			inCodeBlock = !inCodeBlock
			result = append(result, line)
			continue
		}

		// Skip if we're inside a code block
		if inCodeBlock {
			result = append(result, line)
			continue
		}

		// Check if we need to add a blank line before this line
		if i > 0 {
			prevLine := lines[i-1]
			prevTrimmed := bytes.TrimSpace(prevLine)

			// If previous line is not empty and current line starts a list
			if len(prevTrimmed) > 0 && isListStart(trimmed) {
				// Only add blank line if previous line is not already a list item
				if !isListStart(prevTrimmed) {
					result = append(result, []byte(""))
				}
			}
		}

		result = append(result, line)
	}

	return bytes.Join(result, []byte("\n"))
}

// isListStart checks if a line starts a list item (ordered or unordered)
func isListStart(line []byte) bool {
	if len(line) == 0 {
		return false
	}

	// Remove blockquote prefixes to check for lists inside blockquotes
	for bytes.HasPrefix(line, []byte(">")) {
		line = bytes.TrimPrefix(line, []byte(">"))
		line = bytes.TrimLeft(line, " \t")
	}

	// If we removed everything, it was just blockquote markers
	if len(line) == 0 {
		return false
	}

	// Check for unordered list markers: -, *, + followed by space
	if len(line) >= 2 {
		if (line[0] == '-' || line[0] == '*' || line[0] == '+') && line[1] == ' ' {
			return true
		}
	}

	// Check for ordered list: digits followed by . or ) and space
	// Pattern: "1. " or "1) " or "123. " etc.
	digitCount := 0
	for i := 0; i < len(line); i++ {
		if line[i] >= '0' && line[i] <= '9' {
			digitCount++
			continue
		}
		// After digits, expect . or ) followed by space
		if digitCount > 0 && (line[i] == '.' || line[i] == ')') {
			if i+1 < len(line) && line[i+1] == ' ' {
				return true
			}
		}
		break
	}

	return false
}

package docs

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/billgraziano/mssqlh/v2"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pkg/errors"
	"github.com/scalesql/isitsql/internal/mssql"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var ErrNoDocsFolder = errors.New("docs folder not found")

// Document holds a parsed Markdown file.
type Document struct {
	Path string
	HTML template.HTML
}

// Get returns an array of documents and any parsing errors.
// It makes a best effort to parse anything it can.
func Get(domain string, names []string) ([]Document, []error, error) {
	docs := make([]Document, 0)
	problems := make([]error, 0)

	docList, err := getDocList(domain, names)
	if err != nil {
		return []Document{}, problems, errors.Wrap(err, "getdoclist")
	}
	for _, name := range docList {
		doc, exists, err := getDoc(name)
		if err != nil {
			problems = append(problems, err)
		}
		if exists {
			docs = append(docs, doc)
		}
	}
	return docs, problems, nil
}

func getDocList(domain string, names []string) ([]string, error) {
	// clean up any protocol prefix (tcp, np, etc.)
	for i := range names {
		_, names[i] = mssqlh.StripProtocol(names[i])
	}
	docsDir, exists, err := getDocsDir()
	if err != nil {
		return []string{}, errors.Wrap(err, "getdocsdir")
	}
	if !exists {
		return []string{}, ErrNoDocsFolder
	}
	docList := make([]string, 0)
	relativePaths := getRelativePaths(names)
	for _, pth := range relativePaths {
		docList = append(docList, filepath.Join(docsDir, pth))
		docList = append(docList, filepath.Join(docsDir, domain, pth))
	}

	return docList, nil
}

func getRelativePaths(names []string) []string {
	list := make([]string, 0)
	// host, instance := mssql.SplitServerName(serverName)
	// // friendly, fqdn.md, computer.md, computer__instance.md, computer/mssqlserver.md, computer/instance.md
	// list = append(list, friendly+".md")
	// list = append(list, fqdn+".md")
	// list = append(list, host+".md")
	// if instance != "" {
	// 	list = append(list, fmt.Sprintf("%s__%s.md", host, instance))
	// 	list = append(list, filepath.Join(host, instance+".md"))
	// } else {
	// 	list = append(list, fmt.Sprintf("%s__mssqlserver.md", host))
	// 	list = append(list, filepath.Join(host, "mssqlserver.md"))
	// }
	// for i := range list {
	// 	list[i] = strings.ToLower(list[i])
	// }
	// friendly, fqdn.md, computer.md, computer__instance.md, computer/mssqlserver.md, computer/instance.md
	// name.md, name__instance.md, name/mssqlserver.md, name/instance.md
	for _, name := range names {
		host, instance := mssql.SplitServerName(name)
		list = append(list, host+".md")
		if instance != "" {
			list = append(list, fmt.Sprintf("%s__%s.md", host, instance))
			list = append(list, filepath.Join(host, instance+".md"))
		} else {
			list = append(list, fmt.Sprintf("%s__mssqlserver.md", host))
			list = append(list, filepath.Join(host, "mssqlserver.md"))
		}
	}
	for i := range list {
		list[i] = strings.ToLower(list[i])
	}

	// only unique values
	uniques := make([]string, 0, len(list))
	existing := make(map[string]bool)
	for _, str := range list {
		exists := existing[str]
		if exists {
			continue
		}
		existing[str] = true
		uniques = append(uniques, str)
	}
	return uniques
}

func getDoc(name string) (Document, bool, error) {
	body, err := os.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		return Document{}, false, nil
	}
	if err != nil {
		return Document{}, false, errors.Wrap(err, "os.readfile")
	}
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.TaskList,
			extension.DefinitionList,
			extension.Footnote,
			extension.Typographer,
			extension.Linkify,
			extension.Strikethrough,
			extension.CJK,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	var buf bytes.Buffer
	err = md.Convert(body, &buf)

	if err != nil {
		return Document{}, true, errors.Wrap(err, "goldmark.convert")
	}
	bm := bluemonday.UGCPolicy()
	bb := bm.SanitizeBytes(buf.Bytes())
	doc := Document{name, template.HTML(string(bb))}
	return doc, true, nil
}

func getDocsDir() (string, bool, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", false, errors.Wrap(err, "os.executable")
	}
	exeDir := filepath.Dir(exe)
	docsDir := filepath.Join(exeDir, "docs")
	_, err = os.Stat(docsDir)
	if errors.Is(err, os.ErrNotExist) {
		return docsDir, false, nil
	}
	if err != nil {
		return docsDir, false, errors.Wrap(err, "os.stat")
	}
	return docsDir, true, nil
}

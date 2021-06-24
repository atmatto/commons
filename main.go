package main

// Handle the command and execute functions based on the config.

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type options struct {
	input       string
	output      string
	headPath    string
	headerPath  string
	footerPath  string
	titlePrefix string
	titleSuffix string
}

var opt options

type attribute struct {
	attr     string
	hasValue bool
	value    string
}

type Meta struct {
	title       string
	description string
	public      bool
	includes    []string // paths
}

type Document struct {
	path string // relative to input directory
}

var MetaCache = make(map[string]*Meta) // [Document.path] Meta
var writeEmptyTodos = false
var neverWritePublicTodos = false
var localInfoHeader = true // insert a link to the public-branch version of current document in the header
var fileTypes = map[string]string{
	".htm":  "htm",
	".html": "attr",
	".txt":  "attr",
}

/* TODO: replaced
// A rudimentary test, to check if current file is public.
// Path must be an absolute path.
func testPublic(d Document) bool {
	return !strings.Contains(path, filepath.Join(opt.output+"/local/"))
}*/

func getMetaCache(name string) *Meta {

	if mc, ok := MetaCache[name]; ok {
		return mc
	} else {
		file, err := os.Open(name)
		doErr(err)
		defer file.Close()

		getMeta(file)

		if MetaCache[name] == nil {
			log.Panicln("Error: Failed to access file meta cache.")
		}

		return MetaCache[name]
	}
}

// Input: from - absolute, to - relative, local - is the document where the link will be included local?
func getDocLink(from string, to string, local bool) (path string, title string, description string, success bool) {
	toAbs := filepath.Join(filepath.Dir(from), to)
	if filepath.Ext(toAbs) == ".html" {
		toAbs = toAbs[:len(toAbs)-1]
	}
	fm := getMetaCache(toAbs)
	if !local && !fm.public {
		return "", "", "", false
	}
	path = to
	title = strings.ReplaceAll(strings.ReplaceAll(fm.title, opt.titleSuffix, ""), opt.titlePrefix, "")
	description = fm.description
	success = true
	return
}

func getLink(from string, to string) (string, error) {
	if filepath.Ext(to) == ".htm" {
		to = to + "l"
	}
	path, err := filepath.Rel(filepath.Dir(from), to)
	if err != nil {
		fmt.Println("Warning: failed to create link:", err)
		return "", err
	}
	name := filepath.Base(to)[:len(filepath.Base(to))-len(filepath.Ext(filepath.Base(to)))]
	return "<a href=\"" + path + "\">" + name + "</a>", nil
}

func writeTodos(file *os.File, isPublic bool) {
	if (!writeEmptyTodos && len(todos) == 0) || (neverWritePublicTodos && isPublic) {
		return
	}

	_, err := file.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	if err != nil {
		log.Fatalln("Error writing to file:", err)
	}
	// HEAD
	if opt.headPath != "" {
		head, err := os.Open(opt.headPath)
		doErr(err)
		defer head.Close()
		copyFile(head, file)
	}
	_, err = file.WriteString("\t<title>" + opt.titlePrefix + "to-do" + opt.titleSuffix + "</title>")
	if err != nil {
		log.Fatalln("Error writing to file:", err)
	}
	_, err = file.WriteString("</head>\n<body>\n")
	if err != nil {
		log.Fatalln("Error writing to file:", err)
	}
	// HEADER
	if opt.headerPath != "" {
		header, err := os.Open(opt.headerPath)
		doErr(err)
		defer header.Close()
		copyFile(header, file)
	}
	// BODY
	_, err = file.WriteString("<p><em>Things to do.</em></p>\n<hr class=\"hrin\">\n")
	if err != nil {
		log.Fatalln("Error writing to file:", err)
	}
	for doc, todo := range todos {
		if isPublic && !todo.public || len(todo.texts) == 0 {
			continue
		}

		//_, err = file.WriteString("<h1>" + filepath.Base(doc)[:int(math.Min(float64(len(filepath.Base(doc))-len(filepath.Ext(filepath.Base(doc)))), float64(len(filepath.Base(doc)))))] + "</h1>\n<ul>")
		//_, err = file.WriteString("<hr><h1>" + filepath.Base(doc)[:len(filepath.Base(doc))-len(filepath.Ext(filepath.Base(doc)))] + "</h1>\n<ul>")
		heading, err := getLink(filepath.Join(opt.input, "todo.html"), doc)
		heading = strings.ReplaceAll(heading, "a href=", "a class=\"ax\" href=")
		if err != nil {
			_, err = file.WriteString("<hr><h1>-</h1>\n<ul>")
		} else {
			_, err = file.WriteString("<p class=\"blob blob-top\">" + heading + "</p>\n")
		}
		if err != nil {
			log.Fatalln("Error writing to file:", err)
		}
		for i, t := range todo.texts {
			if t == "" {
				continue
			}
			if i == len(todo.texts)-1 {
				_, err = file.WriteString("\t<div class=\"blob blob-bottom blob-stack\">" + t + "</div>\n")
			} else {
				_, err = file.WriteString("\t<div class=\"blob blob-middle blob-stack\">" + t + "</div>\n")
			}

			if err != nil {
				log.Fatalln("Error writing to file:", err)
			}
		}
		_, err = file.WriteString("")
		if err != nil {
			log.Fatalln("Error writing to file:", err)
		}
	}
	// FOOTER
	if opt.footerPath != "" {
		footer, err := os.Open(opt.footerPath)
		doErr(err)
		defer footer.Close()
		copyTemplate(footer, file)
	}
	_, err = file.WriteString("</body>\n</html>\n")
	if err != nil {
		log.Fatalln("Error writing to file:", err)
	}
}

func writeHtm(in *os.File, out *os.File, meta Meta) {
	_, err := out.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	if err != nil {
		log.Fatalln("Error writing to file:", err)
	}
	// HEAD
	if opt.headPath != "" {
		head, err := os.Open(opt.headPath)
		doErr(err)
		defer head.Close()
		copyTemplate(head, out)
	}
	_, err = out.WriteString("\t<title>" + meta.title + "</title>")
	if err != nil {
		log.Fatalln("Error writing to file:", err)
	}
	_, err = out.WriteString("</head>\n<body>\n")
	if err != nil {
		log.Fatalln("Error writing to file:", err)
	}
	// HEADER
	if opt.headerPath != "" {
		header, err := os.Open(opt.headerPath)
		doErr(err)
		defer header.Close()
		copyTemplate(header, out)
	}
	if localInfoHeader && !strings.Contains(out.Name(), filepath.Join(opt.output, "/public")) {
		if meta.public {
			_, err = out.WriteString("<p style=\"color: #777 !important; display: block; position: fixed; top: 15px; right: 30px;\"><a class=\"ax\" style=\"color: #777 !important;\" href=\"" + strings.Replace(out.Name(), "local", "public", 1) + "\">(public)</a></p>\n")
			if err != nil {
				log.Fatalln("Error writing to file:", err)
			}
		} else {
			_, err = out.WriteString("<p style=\"display: block; position: fixed; top: 15px; right: 30px; color: transparent; text-shadow: 0 0 0 #999;\">🔒</p>\n")
			if err != nil {
				log.Fatalln("Error writing to file:", err)
			}
		}
	}
	// BODY
	copyAttrFile(in, out)
	// FOOTER
	if opt.footerPath != "" {
		footer, err := os.Open(opt.footerPath)
		doErr(err)
		defer footer.Close()
		copyTemplate(footer, out)
	}
	_, err = out.WriteString("</body>\n</html>\n")
	if err != nil {
		log.Fatalln("Error writing to file:", err)
	}
}

func copyAttrFile(in *os.File, out *os.File) {
	scanner := bufio.NewScanner(in)
	localOnly := false
	for scanner.Scan() {
		if filepath.Ext(in.Name()) == ".htm" && strings.Contains(scanner.Text(), "<img src=") {
			log.Println("Warning: Use |Img()| attribute instead of <img> tag.")
		}
		attr, isAttr := getAttr(scanner.Text())
		if isAttr {
			if attr.attr == "LocalEnd" && !attr.hasValue {
				localOnly = false
			} else if !localOnly || (localOnly && strings.Contains(out.Name(), filepath.Join(opt.output+"/local/"))) {
				switch {
				case attr.attr == "LocalBegin" && !attr.hasValue:
					localOnly = true
				case attr.attr == "Img" && attr.hasValue:
					_, err := out.WriteString("<img src=\"" + attr.value + "\">\n")
					if err != nil {
						log.Panicln("Error writing file:", err)
					}
				case attr.attr == "Todo" && attr.hasValue:
					_, err := out.WriteString("<p>TODO: <em>" + attr.value + "</em></p>\n")
					if err != nil {
						log.Panicln("Error writing file:", err)
					}
				case attr.attr == "DocLinkA" && attr.hasValue:
					p, t, d, s := getDocLink(in.Name(), attr.value, strings.Contains(out.Name(), filepath.Join(opt.output+"/local/")))
					if s && d != "" && t != "" {
						_, err := out.WriteString("<a href=\"" + p + "\">" + t + "</a> – " + d + "\n")
						if err != nil {
							log.Panicln("Error writing file:", err)
						}
					} else if s && t != "" {
						_, err := out.WriteString("<a href=\"" + p + "\">" + t + "</a>\n")
						if err != nil {
							log.Panicln("Error writing file:", err)
						}
					}
				case attr.attr == "DocLinkDef" && attr.hasValue:
					p, t, d, s := getDocLink(in.Name(), attr.value, strings.Contains(out.Name(), filepath.Join(opt.output+"/local/")))
					if s && d != "" && t != "" {
						_, err := out.WriteString("<dt><a href=\"" + p + "\">" + t + "</a></dt>\n<dd>" + d + "</dd>\n")
						if err != nil {
							log.Panicln("Error writing file:", err)
						}
					} else if s && t != "" {
						_, err := out.WriteString("<dt><a href=\"" + p + "\">" + t + "</a></dt>\n")
						if err != nil {
							log.Panicln("Error writing file:", err)
						}
					}
				}
			}
		} else if !localOnly || (localOnly && strings.Contains(out.Name(), filepath.Join(opt.output+"/local/"))) {
			var err error
			if len(scanner.Text()) > 1 && scanner.Text()[:2] == "\\|" {
				_, err = out.WriteString(scanner.Text()[1:] + "\n")
			} else {
				_, err = out.WriteString(scanner.Text() + "\n")
			}
			if err != nil {
				log.Panicln("Error writing file:", err)
			}
		}
	}
}

func copyFile(in *os.File, out *os.File) {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		_, err := out.WriteString(scanner.Text() + "\n")
		if err != nil {
			log.Panicln("Error writing file:", err)
		}
	}
}

func copyTemplate(in *os.File, out *os.File) {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		t := scanner.Text()
		b := -1
		for i := 0; i < len(t); i++ {
			if i+1 < len(t) && t[i:i+2] == "{{" {
				b = i
			}
			if b != -1 && i+1 < len(t) && t[i:i+2] == "}}" {
				rel, err := filepath.Rel(filepath.Dir(out.Name()), opt.output)
				rel = filepath.Dir(rel)
				rel = filepath.Join(rel, t[b+2:i])
				if err != nil {
					log.Fatalln("Error (template): ", err)
				}
				t = t[:b] + rel + t[i+2:]
				b = -1
			}
		}
		_, err := out.WriteString(t + "\n")
		if err != nil {
			log.Panicln("Error writing file:", err)
		}
	}
}

// Get attribute at specified line.
// Returns attribute struct and a bool, true when attribute was found.
func getAttr(line string) (attribute, bool) {
	attr := attribute{}
	if len(line) == 0 || line[0] != '|' {
		return attr, false
	}
	attrstart := 1
	attrend := -1
	valstart := -1
	valend := -1

	i := 1
	p := 0
line:
	for {
		if i == len(line) || line[i] == '\n' {
			if attrend == -1 || (valstart != -1 && valend == -1) {
				return attr, false
			}
			break line
		}
		switch line[i] {
		case '|':
			if valstart != -1 && valend == -1 { // xx(x| - unclosed value parenthesis
				fmt.Println("Warning: Malformed attribute (unclosed value parenthesis):", line)
				return attr, false
			} else if i == 1 { // || - empty
				fmt.Println("Warning: Malformed attribute (empty attribute):", line)
				return attr, false
			} else { /* - valid */
				attrend = i
				break line
			}
		case '(':
			if i == 1 { // |( - unnamed
				fmt.Println("Warning: Malformed attribute (unknown type):", line)
				return attr, false
			} else { /* valid */
				p++
				if p == 1 {
					valstart = i + 1
				}
			}
		case ')':
			if valstart == -1 { /* |xx) - unopened value parenthesis */
				fmt.Println("Warning: Malformed attribute (unopened value parenthesis):", line)
				return attr, false
			} else { /* - valid */
				valend = i
			}
		}
		i++
	}
	if attrend == -1 || (valstart == -1 && valend != -1) || (valstart != -1 && valend == -1) {
		log.Panicln("Error: attribute parsing error (this should never happen):", line)
	}
	if valend != -1 && valend+1 != attrend {
		fmt.Println("Warning: Malformed attribute (there may not be any text after value parenthesis is closed):", line)
		return attr, false
	}

	if valstart != -1 {
		attr.hasValue = true
		attr.attr = line[attrstart : valstart-1]
		attr.value = line[valstart:valend]
	} else {
		attr.attr = line[attrstart:attrend]
	}
	return attr, true
}

func getMeta(file *os.File) Meta {
	if MetaCache[file.Name()] != nil {
		return *MetaCache[file.Name()]
	}
	scanner := bufio.NewScanner(file)
	fm := Meta{}
	if todos[file.Name()] == nil {
		todos[file.Name()] = &todoent{}
	}
	for scanner.Scan() {
		//fmt.Println(scanner.Text())
		attr, found := getAttr(scanner.Text())
		if !found || attr.attr == "" {
			continue
		}
		switch {
		case attr.attr == "OverrideTitle" && attr.hasValue:
			fm.title = attr.value
		case attr.attr == "Title" && attr.hasValue:
			fm.title = opt.titlePrefix + attr.value + opt.titleSuffix
		case attr.attr == "Description" && attr.hasValue:
			fm.description = attr.value
		case attr.attr == "Public" && !attr.hasValue:
			fm.public = true
		case (attr.attr == "Include" || attr.attr == "Img") && attr.hasValue:
			fm.includes = append(fm.includes, attr.value)
		case attr.attr == "Todo" && attr.hasValue:
			todos[file.Name()].texts = append(todos[file.Name()].texts, attr.value)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	if fm.title == "" {
		fm.title = filepath.Base(file.Name())
		fm.title = fm.title[:len(fm.title)-len(filepath.Ext(fm.title))]
		fm.title = opt.titlePrefix + fm.title + opt.titleSuffix
	}

	todos[file.Name()].public = fm.public
	MetaCache[file.Name()] = &fm
	return fm
}

func (d *Document) doAttributeDocument() {
	fmt.Println("File:", d.Abs(false, false))
	file, err := os.Open(d.Abs(false, false))
	doErr(err)
	defer file.Close()

	meta := getMeta(file)

	local, public := d.Abs(true, false), d.Abs(true, true)

	// Local
	err = os.MkdirAll(filepath.Dir(local), 0755)
	doErr(err)
	localf, err := os.OpenFile(local, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	doErr(err)
	defer localf.Close()
	file.Seek(0, 0)

	if d.FileType() == "htm" {
		writeHtm(file, localf, meta)
	} else {
		copyAttrFile(file, localf)
	}

	if meta.public { // Public
		err = os.MkdirAll(filepath.Dir(public), 0755)
		doErr(err)
		publicf, err := os.OpenFile(public, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		doErr(err)
		defer publicf.Close()
		file.Seek(0, 0)

		if d.FileType() == "htm" {
			writeHtm(file, publicf, meta)
		} else {
			copyAttrFile(file, publicf)
		}
	}

	/* Includes */
	in := 0
	for _, incl := range meta.includes {
		abs := filepath.Join(filepath.Dir(d.Abs(false, false)), incl)
		rel, err := filepath.Rel(opt.input, abs)
		doErr(err)
		doc := Document{path: rel}
		incloc, incpub := doc.Abs(true, false), doc.Abs(true, true)

		incf, err := os.Open(abs)
		doErr(err)
		defer incf.Close()

		err = os.MkdirAll(filepath.Dir(incloc), 0755)
		doErr(err)
		inclocf, err := os.OpenFile(incloc, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		doErr(err)
		defer inclocf.Close()
		incf.Seek(0, 0)

		_, err = io.Copy(inclocf, incf)
		doErr(err)

		if meta.public {
			err = os.MkdirAll(filepath.Dir(incpub), 0755)
			doErr(err)
			incpubf, err := os.OpenFile(incpub, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
			doErr(err)
			defer incpubf.Close()
			incf.Seek(0, 0)

			_, err = io.Copy(incpubf, incf)
			doErr(err)
		}
		in++
	}
	if in != 0 {
		fmt.Println("Included", in, "files.")
	}
}

func (d *Document) Abs(output bool, public bool) string { // TODO: check
	if !output && public {
		log.Panicln("Error: called Abs with argument output=false but public=true")
	}
	if !output { // input
		return filepath.Join(opt.input, d.path)
	} else if public {
		rel := filepath.Dir(d.path)
		file := filepath.Base(d.path)
		if filepath.Ext(file) == ".htm" {
			file = file + "l"
		}
		return filepath.Join(opt.output, "public", rel, file)
	} else { // local
		rel := filepath.Dir(d.path)
		file := filepath.Base(d.path)
		if filepath.Ext(file) == ".htm" {
			file = file + "l"
		}
		return filepath.Join(opt.output, "local", rel, file)
	}
}

func (d *Document) FileType() string {
	ext := filepath.Ext(d.path)
	if t, exists := fileTypes[ext]; !exists {
		return "other"
	} else {
		return t
	}
}

func walkDirectory(path string, d fs.DirEntry, e error) error {
	if !d.Type().IsRegular() {
		return nil
	}

	relativePath, err := filepath.Rel(opt.input, path)
	doErr(err)
	doc := Document{path: relativePath}

	fileType := doc.FileType()
	switch fileType {
	case "other":
		return nil
	case "htm", "attr":
		doc.doAttributeDocument()
		return nil
	}
	return nil
}

func PreWalkHook() {
	MetaCache[filepath.Join(opt.input, "/todo.htm")] = &Meta{title: "todo", description: "things to do", public: !neverWritePublicTodos}
}

func PostWalkHook() {
	doTodos()
}

type todoent struct {
	public bool
	texts  []string
}

var todos = make(map[string]*todoent) // [path] todoent

func doTodos() {
	d := Document{path: "todo.html"}
	localtodo, publictodo := d.Abs(true, false), d.Abs(true, true)
	err := os.MkdirAll(filepath.Dir(localtodo), 0755)
	doErr(err)
	localtodof, err := os.OpenFile(localtodo, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	doErr(err)
	defer localtodof.Close()

	err = os.MkdirAll(filepath.Dir(publictodo), 0755)
	doErr(err)
	publictodof, err := os.OpenFile(publictodo, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	doErr(err)
	defer publictodof.Close()

	writeTodos(localtodof, false)
	writeTodos(publictodof, true)
}

// doErr is a simple helper function to handle errors.
func doErr(err error) {
	if err != nil {
		log.Fatalln("Error:", err)
	}
}

// parseCommand parses command arguments and displays program usage information.
func parseCommand() []string {
	headPath := flag.String("head", "", "path to file used as html <head>")
	headerPath := flag.String("header", "", "path to file used as page header")
	footerPath := flag.String("footer", "", "path to file used as page footer")
	titlePrefix := flag.String("tprefix", "", "page title prefix")
	titleSuffix := flag.String("tsuffix", "", "page title suffix")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "(parameters) INPUTDIR OUTPUTDIR")
		fmt.Fprintf(os.Stderr, "\nParameters:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		flag.Usage()
		os.Exit(0)
	}

	var err error
	opt.input, err = filepath.Abs(args[0])
	doErr(err)
	opt.output, err = filepath.Abs(args[1])
	doErr(err)

	opt.headPath, err = filepath.Abs(*headPath)
	doErr(err)
	opt.headerPath, err = filepath.Abs(*headerPath)
	doErr(err)
	opt.footerPath, err = filepath.Abs(*footerPath)
	doErr(err)
	opt.titlePrefix = *titlePrefix
	opt.titleSuffix = *titleSuffix

	if opt.headPath == "" {
		fmt.Println("Warning: <head> file not specified (-head=FILE)")
	}
	return args
}

func main() {
	log.SetFlags(0)

	args := parseCommand()

	err := os.MkdirAll(args[0], 0755)
	if err != nil {
		log.Fatal("Error: Couldn't create input directory - ", err)
	}
	err = os.MkdirAll(args[1], 0755)
	if err != nil {
		log.Fatal("Error: Couldn't create output directory - ", err)
	}

	PreWalkHook()

	_ = filepath.WalkDir(args[0], walkDirectory)

	PostWalkHook()
}

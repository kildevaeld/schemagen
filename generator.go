package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Value represents a declared constant.
type Value struct {
	name string // The name of the constant.
	// The value is stored as a bit pattern alone. The boolean tells us
	// whether to interpret it as an int64 or a uint64; the only place
	// this matters is when sorting.
	// Much of the time the str field is all we need; it is printed
	// by Value.String.
	value  uint64 // Will be converted to int64 when needed.
	signed bool   // Whether the constant is a signed type.
	str    string // The string representation given by the "go/exact" package.
}

func (v *Value) String() string {
	return v.str
}

// isDirectory reports whether the named file is a directory.
func isDirectory(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		log.Fatal(err)
	}
	return info.IsDir()
}

// Generator holds the state of the analysis. Primarily used to buffer
// the output for format.Source.
type Generator struct {
	buf bytes.Buffer // Accumulated output.
	pkg *Package     // Package we are scanning.
}

func (g *Generator) Printf(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, format, args...)
}

// File holds a single parsed file and associated data.
type File struct {
	pkg  *Package  // Package to which this file belongs.
	file *ast.File // Parsed AST.
	// These fields are reset for each type being generated.
	typeName string    // Name of the constant type.
	values   []*Schema // Accumulator for constant values of that type.
}

type Package struct {
	dir      string
	name     string
	defs     map[*ast.Ident]types.Object
	files    []*File
	typesPkg *types.Package
}

// parsePackageDir parses the package residing in the directory.
func (g *Generator) parsePackageDir(directory string) {
	pkg, err := build.Default.ImportDir(directory, 0)
	if err != nil {
		log.Fatalf("cannot process directory %s: %s", directory, err)
	}
	var names []string
	names = append(names, pkg.GoFiles...)
	names = append(names, pkg.CgoFiles...)
	// TODO: Need to think about constants in test files. Maybe write type_string_test.go
	// in a separate pass? For later.
	// names = append(names, pkg.TestGoFiles...) // These are also in the "foo" package.
	names = append(names, pkg.SFiles...)
	names = prefixDirectory(directory, names)
	g.parsePackage(directory, names, nil)
	//fmt.Printf("PAGEVE %#v", g.pkg)
}

// parsePackageFiles parses the package occupying the named files.
func (g *Generator) parsePackageFiles(names []string) {
	g.parsePackage(".", names, nil)
}

// prefixDirectory places the directory name on the beginning of each name in the list.
func prefixDirectory(directory string, names []string) []string {
	if directory == "." {
		return names
	}
	ret := make([]string, len(names))
	for i, name := range names {
		ret[i] = filepath.Join(directory, name)
	}
	return ret
}

// parsePackage analyzes the single package constructed from the named files.
// If text is non-nil, it is a string to be used instead of the content of the file,
// to be used for testing. parsePackage exits if there is an error.
func (g *Generator) parsePackage(directory string, names []string, text interface{}) {
	var files []*File
	var astFiles []*ast.File
	g.pkg = new(Package)
	fs := token.NewFileSet()
	for _, name := range names {
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		parsedFile, err := parser.ParseFile(fs, name, text, 0)
		if err != nil {
			log.Fatalf("parsing package: %s: %s", name, err)
		}
		astFiles = append(astFiles, parsedFile)
		files = append(files, &File{
			file: parsedFile,
			pkg:  g.pkg,
		})
	}
	if len(astFiles) == 0 {
		log.Fatalf("%s: no buildable Go files", directory)
	}
	g.pkg.name = astFiles[0].Name.Name
	g.pkg.files = files
	g.pkg.dir = directory
	// Type check the package.
	g.pkg.check(fs, astFiles)
}

// check type-checks the package. The package must be OK to proceed.
func (pkg *Package) check(fs *token.FileSet, astFiles []*ast.File) {
	pkg.defs = make(map[*ast.Ident]types.Object)
	config := types.Config{Importer: importer.Default(), FakeImportC: true}
	info := &types.Info{
		Defs: pkg.defs,
	}
	typesPkg, err := config.Check(pkg.dir, fs, astFiles, info)
	if err != nil {
		log.Fatalf("checking package: %s", err)
	}
	pkg.typesPkg = typesPkg
}

// generate produces the String method for the named type.
func (g *Generator) generate(typeName string) {
	values := make([]*Schema, 0, 100)
	for _, file := range g.pkg.files {
		// Set the state for this run of the walker.
		file.typeName = typeName
		file.values = nil
		if file.file != nil {
			ast.Inspect(file.file, file.genDecl)
			values = append(values, file.values...)
		}
	}

	if len(values) == 0 {
		log.Fatalf("no values defined for type %s", typeName)
	}

}

func (g *Generator) schemas() []*Schema {
	var out []*Schema
	for _, file := range g.pkg.files {
		out = append(out, file.values...)
	}
	return out
}

func (self *File) handleField(schema *Schema, iden *ast.Ident, field *ast.Field) Property {

	t_name := ""
	format := ""
	switch iden.Name {
	case "string":
		t_name = "string"
	case "Time":
		t_name = "string"
		format = "date-time"
	case "int", "uint", "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64",
		"float64":
		t_name = "number"
	case "bool":
		t_name = "boolean"
	default:
		fmt.Printf("Name %s\n", iden.Name)
	}

	if t_name == "" {
		//iden := self.file.Scope.Lookup(iden.Name)
		/*if iden.Decl != nil {
			if s, ok := iden.Decl.(*ast.TypeSpec) {
				p = self.handleStruct(schema, s, node)
			}
		}*/
	}
	var p Property
	switch t_name {
	case "string":
		p = &StringProperty{
			Description: field.Comment.Text(),
			Format:      format,
		}
	case "number":
		p = &NumberProperty{
			Description: field.Comment.Text(),
		}

	}

	return p

}

func (self *File) handleStruct(schema Schema, typspec *ast.TypeSpec, node *ast.StructType) *ObjectProperty {
	if node.Fields == nil || node.Fields.NumFields() == 0 {
		return nil
	}

	//schema := Schema{}
	schema.Title = typspec.Name.Name
	schema.Description = typspec.Comment.Text()

	oprop := &ObjectProperty{
		Properties: make(map[string]Property),
	}

	for _, field := range node.Fields.List {
		var prop Property
		name := field.Names[0].Name
		switch f := field.Type.(type) {
		case *ast.StarExpr:
			sel := f.X.(*ast.SelectorExpr)
			if !sel.Sel.IsExported() {
				continue
			}

			//fmt.Printf("X = %#v\n", sel.X)
			prop = self.handleField(&schema, sel.Sel, field)
			if prop != nil {
				oprop.Required = append(oprop.Required, name)
			}
		case *ast.Ident:
			//fmt.Printf("%s", field.Tag)
			//

			prop = self.handleField(&schema, f, field)
		}

		if prop == nil {
			continue
		}

		oprop.Properties[name] = prop

	}

	return oprop
}

func (self *File) genDecl(node ast.Node) bool {
	decl, ok := node.(*ast.TypeSpec)

	if !ok {
		return true
	}

	var s *ast.StructType
	if s, ok = decl.Type.(*ast.StructType); !ok {
		return true
	}

	if s.Fields == nil || s.Fields.NumFields() == 0 {
		return true
	}

	schema := Schema{}
	schema.Title = decl.Name.Name
	schema.Description = decl.Comment.Text()

	/*for _, field := range s.Fields.List {
		var prop Property
		switch f := field.Type.(type) {
		case *ast.StarExpr:
			sel := f.X.(*ast.SelectorExpr)
			if !sel.Sel.IsExported() {
				continue
			}

			//fmt.Printf("X = %#v\n", sel.X)
			p = self.handleField(&schema, sel.Sel, field)
		case *ast.Ident:
			//fmt.Printf("%s", field.Tag)
			//

			p = self.handleField(&schema, f, field)
		}

		if p == nil {
			continue
		}

	}*/
	p := self.handleStruct(schema, decl, s)
	if p == nil {
		return true
	}
	schema.Root = p

	self.values = append(self.values, &schema)

	return false
}

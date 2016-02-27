package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/types"

	"github.com/kildevaeld/dict"
)

const (
	kPath = "./"
)

func main() {
	/*fset := token.NewFileSet()

	pkgs, e := parser.ParseDir(fset, kPath, nil, 0)
	if e != nil {
		log.Fatal(e)
		return
	}

	astf := make([]*ast.File, 0)
	for _, pkg := range pkgs {
		fmt.Printf("package %v\n", pkg.Name)
		for fn, f := range pkg.Files {
			fmt.Printf("file %v\n", fn)
			astf = append(astf, f)
		}
	}

	config := &types.Config{
		Error: func(e error) {
			fmt.Println(e)
		},
		Importer: importer.Default(),
	}
	info := types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	pkg, e := config.Check(kPath, fset, astf, &info)
	if e != nil {
		fmt.Println(e)
	}
	fmt.Printf("types.Config.Check got %v\n", pkg.String())

	for _, f := range astf {
		ast.Walk(&PrintASTVisitor{&info}, f)
	}*/

	g := Generator{}
	g.parsePackageFiles([]string{"test.go"})

	g.generate("TestStruct")
	g.generate("TestStruct2")

	for _, schema := range g.schemas() {
		b, _ := json.MarshalIndent(schema.ToMap(), "", "  ")
		fmt.Printf("JSON %s\n", b)
	}

}

func handleField(node ast.Node, infos *types.Info, field *ast.Field) dict.Map {
	m := dict.NewMap()
	switch t := field.Type.(type) {
	case *ast.StarExpr:
		sel := t.X.(*ast.SelectorExpr)
		fmt.Printf("%s\n", sel.Sel.Name)

		//fmt.Printf(" %v\n", infos.ObjectOf(sel.Sel))
		fmt.Printf(" %v\n", infos.TypeOf(sel.Sel))
	}

	return m
}

type StructASTVisitor struct {
	info     *types.Info
	Exported string
}

func (v *StructASTVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return v
	}
	//fmt.Printf("%s\n", reflect.TypeOf(node).String())

	switch t := node.(type) {
	case *ast.FieldList:
		fmt.Printf("fields %v\n", t.NumFields())
		for _, f := range t.List {
			//name := f.Names[0]
			//
			handleField(node, v.info, f)

		}
	}

	return v
}

type PrintASTVisitor struct {
	info *types.Info
}

func (v *PrintASTVisitor) Visit(node ast.Node) ast.Visitor {
	/*if node != nil {
		fmt.Printf("%s", reflect.TypeOf(node).String())
		switch node.(type) {
		case ast.Expr:
			t := v.info.TypeOf(node.(ast.Expr))
			if t != nil {
				fmt.Printf(" : %s", t.String())
			}
		}
		fmt.Println()
	}
	return v*/
	switch t := node.(type) {
	case *ast.TypeSpec:
		switch t.Type.(type) {
		case *ast.StructType:
			name := t.Name.Name
			fmt.Printf("Namen %s\n", name)

			return &StructASTVisitor{v.info, ""}
		}
	}
	return v
}

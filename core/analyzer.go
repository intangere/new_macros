package core

import (
	"golang.org/x/tools/go/packages"
	//"golang.org/x/tools/go/ast/astutil"
	"fmt"
	"go/token"
	"go/types"
	"go/ast"
	//"go/printer"
	"strings"
	"os"
	//"bytes"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/dstutil"
	"path/filepath"
	"regexp"
	"encoding/json"
	"github.com/dave/dst/decorator/resolver/gotypes"
	"github.com/dave/dst/decorator/resolver/simple"
)

const loadMode = packages.NeedName |
    packages.NeedFiles |
    packages.NeedCompiledGoFiles |
    packages.NeedImports |
    packages.NeedDeps |
    packages.NeedTypes |
    packages.NeedSyntax |
    packages.NeedTypesInfo

func test() string {
	return "kek"
}

type Scope struct {
        // contains all the defined variables for some block of code. could be a module as well
	Variables map[string]Variable
}

type Variable struct {
	Name string
	BasicType string
	Node dst.Node
}

type PackageScope struct {
        Scope
}

func AppendStmt(node *dst.FuncDecl, stmt dst.Stmt) {
	node.Body.List = append(node.Body.List, stmt)
	// shift comments automatically
	/*for _, comment := range node.Doc.List {
		// gangy
		fmt.Println("pos", pkg.Fset.Position(comment.Slash).Offset, pkg.Fset.Position(node.End()).Offset)
		fmt.Println("ganaglang", comment.Text)
		if pkg.Fset.Position(comment.Slash).Offset > pkg.Fset.Position(node.End()).Offset {
			fmt.Println("Called")
			comment.Slash++
		}
	}*/
}

/*func InjectBlocks(new_func_blocks map[dst.Node][]dst.Node) {
	fmt.Println("Injecting expanded macros..")

	dec := decorator.NewDecorator(pkg.Fset)

	for _, f := range pkg.Syntax {
		/*fmt.Println(f.Decls)
		started := false
		var start dst.Node
		index := 0*/

		// maybe this will fix comments
		/*astutil.Apply(f, func(cr *astutil.Cursor) bool {
			n := cr.Node()
			if n == nil {
				return true
			}

			expr := &dst.UnaryExpr{
						Op: token.NOT,
						X:  nil,
					}


			cr.InsertAfter(expr)
			return true
		}, nil)

		/*
			if _, ok := new_func_blocks[n]; ok {
				// this is the start of a block that has to be overwritten
				fmt.Println("Started expansion")
				started = true
				start = n
				index = 0
			}
			fmt.Println(index)
			if started {
				fmt.Println("Checking node equality")
				if n != new_func_blocks[start][index] {
					fmt.Println("Inserted node")
					//cr.InsertBefore(new_func_blocks[start][index])
					cr.InsertBefore(new_func_blocks[start][index])
				}

				if index == len(new_func_blocks) {
					fmt.Println("Expanded index", index, len(new_func_blocks))
					started = false
					index = 0
					fmt.Println("Expansion ended")
					fmt.Println(new_func_blocks)
				} else {
					index++
				}
			}

			return true
		}, nil)*/

		// time to perform super hackerman to fix comments.
		// the fact go fucked this ast parsing up so bad is disappointing af

		// Decorate the *dst.File to give us a *dst.File
/*		f, err := dec.DecorateFile(f)
		if err != nil {
			panic(err)
		}

		decorator.Print(f)
		//printer.Fprint(os.Stdout, pkg.Fset, f)
	}


*/

// r5edo this..
func invert_map(some_map map[string]ImportDescriptor) map[string]string {
	another_map := map[string]string{}
	for k,v := range some_map {
		another_map[v.Path] = k
	}
	return another_map
}

func Contains[T comparable](ts []T, n T) bool {
	for _, t := range ts {
		if t == n {
			return true
		}
	}
	return false
}

func Run(dec *decorator.Decorator, pkg AnnotatedPackage, skip_paths []string) {
//, pkg_path string, info *types.Info) {
	main_file := ""

	for idx, f := range pkg.Files {

		fmt.Println("Skip paths", skip_paths)
		fmt.Println("File", dec.Filenames[f])

		if annos, ok := FileAnnotationMap[f]; !ok || len(annos) == 0 || shouldSkipOutput(skip_paths, dec.Filenames[f]) {
			continue
		}

		//fmt.Println("known imports", pkg.ImportMap[pkg.Files[idx]])
		//fmt.Println("known imports", invert_map(pkg.ImportMap[pkg.Files[idx]]))
		//for n, z := range pkg.ImportMap[pkg.Files[idx]] {
		//	fmt.Println(n, z)
		//}

		r := decorator.NewRestorerWithImports(pkg.PkgPath, simple.New(invert_map(pkg.ImportMap[pkg.Files[idx]])))

		og_name := dec.Filenames[f]

		if strings.HasSuffix(og_name, "macro_generator.go") {
			continue
		}

		new_name := strings.Split(og_name, ".go")[0]
		new_name = new_name + "_generated.go"

		//defer os.Remove(new_name)

		file, err := os.Create(new_name)
		if err != nil {
			panic(err)
		}

		err = r.Fprint(file,f )
		if err != nil {
			panic(err)
		}
		//decorator.Fprint(file,f )
		// need to find the main file in order to execute it
	}

	if main_file != "" {
		fmt.Println("Running..")
	}

	/*for _, f := range pkg.Syntax {


		file_src := pkg.Fset.File(f.Pos())
		if strings.HasSuffix(file_src.Name(), "macro_generator.go") {
			continue
		}

                new_file := file_src.Name()[:len(file_src.Name())-3] +"_generated.go"

		var b bytes.Buffer
		printer.Fprint(&b, pkg.Fset, f)

		err := os.WriteFile(new_file, b.Bytes(), 0644)
		if err != nil {
			panic(err)
		}
	}*/
}

//var pkg *packages.Package

type AnnotatedPackage struct {
	Annotations map[dst.Node][]Annotation
	Funcs []dst.Node
	Consts []dst.Node
	Structs []dst.Node
	Info *types.Info
	Dec *decorator.Decorator
	Files []*dst.File
	PkgName string
	PkgPath string
	ImportMap map[*dst.File]map[string]ImportDescriptor // import name/alias -> imported path 
	NodeToFiles map[dst.Node]*dst.File
}

func ignoreFiles(ignore_files []string) []string {
        paths := []string{}
        err := filepath.Walk(".",
                func(path string, info os.FileInfo, err error) error {
                if err != nil {
                        return err
                }
                for _, pattern := range ignore_files {
                        if strings.HasPrefix(pattern, "r:") {
                                r, _ := regexp.Compile(pattern[2:])
                                if r.MatchString(path) {
                                        paths = append(paths, path)
                                }
                        } else {
				// ignore an exact file, i.e "test/test.go", "test.go"
                                if pattern == path {
                                        paths = append(paths, path)
                                } else if strings.HasPrefix(path, pattern) && pattern[len(pattern)-1] == '/' {
					// skip all files in a given directory i.e -ignore "test/"
					fmt.Println("Skipping file in ignored directory", path)
				}
                        }
                }

                //fmt.Println(path, info.Size())
                return nil
        })

        if err != nil {
                panic(err)
        }


        fmt.Println("Files to ignore", paths)

        return paths
}

func shouldSkipOutput(skip_patterns []string, file_path string) bool {
        for _, pattern := range skip_patterns {
                if strings.HasPrefix(pattern, "r:") {
                        r, _ := regexp.Compile(pattern[2:])
                        if r.MatchString(file_path) {
				return true
                        }
                } else {
			// ignore an exact file, i.e "test/test.go", "test.go"
                        if pattern == file_path {
				return true
                        } else if strings.HasPrefix(file_path, pattern) && pattern[len(pattern)-1] == '/' {
				fmt.Println("Skipping file output in ignored directory", file_path)
				return true
			}
                }
        }
	return false
}

var imported_packages map[string]AnnotatedPackage = map[string]AnnotatedPackage{}

func buildScope(f *dst.FuncDecl, dec *decorator.Decorator, info *types.Info) Scope {
	scope := Scope{
		Variables: map[string]Variable{},
	}

	for _, n := range f.Body.List {
		switch n.(type) {
		case *dst.AssignStmt:
			stmt := n.(*dst.AssignStmt).Lhs[0]
			fmt.Println("stmt", stmt)
			a, ok := dec.Ast.Nodes[stmt].(*ast.Ident)
			if !ok{
				break
			}
			if a.String() == "_" {
				break
			}
			if b, ok := info.Types[a]; ok {
				fmt.Println("Assign", b)
				scope.Variables[a.String()] = Variable{
					Name: a.String(),
					BasicType: b.Type.String(),
				}
			} else if b, ok := info.Defs[a]; ok {
				fmt.Println("Assign 1", b)
				scope.Variables[a.String()] = Variable{
					Name: a.String(),
					BasicType: b.Type().String(),
				}
			} else {
				fmt.Println("Not found")
			}
		}
	}
	fmt.Println("s", scope)
	return scope
}

func buildTypedFuncParams(f *dst.FuncDecl, dec *decorator.Decorator, info *types.Info) []Variable {
	vars := []Variable{}
	for _, param := range f.Type.Params.List {
		a, ok := dec.Ast.Nodes[param].(*ast.Field)
		if ok{
			fmt.Println("BIG PARAMA!!!", info.Types[a.Type].Type.String())
			vars = append(vars, Variable{
				Name: a.Names[0].String(),
				BasicType: info.Types[a.Type].Type.String(),
			})
		} else {
			panic("k")
		}
	}
	return vars
}

func buildTypedFuncReturns(f *dst.FuncDecl, dec *decorator.Decorator, info *types.Info) []Variable {
	vars := []Variable{}
	if f.Type.Results != nil {
		for _, param := range f.Type.Results.List {
			a, ok := dec.Ast.Nodes[param].(*ast.Field)
			if ok{
				fmt.Println("BIG PARAMA!!!", info.Types[a.Type].Type.String())
				name := ""
				if len(a.Names) > 0 {
					name = a.Names[0].String()
				}
				vars = append(vars, Variable{
					Name: name,
					BasicType: info.Types[a.Type].Type.String(),
				})
			} else {
				panic("k")
			}
		}
	}
	return vars
}

var annotated_packages []AnnotatedPackage


type ImportDescriptor struct {
	Name string
	Alias bool
	Path string
}

//func Build(pkg_name string, ignore_files []string) (map[dst.Node][]Annotation, []dst.Node, *types.Info, *decorator.Decorator, []*dst.File) {
func Build(pkg_name string, ignore_files []string) []AnnotatedPackage {

        loadConfig := new(packages.Config)
        loadConfig.Mode = loadMode
        loadConfig.Fset = token.NewFileSet()

	ignorables := ignoreFiles(ignore_files)
	for _, file := range ignorables {
		os.Rename(file, file[:len(file)-3])
	}

	defer func() {
		for _, file := range ignorables {
			os.Rename(file[:len(file)-3], file)
		}
		fmt.Println("Restored ignored files")
	}()

        // can't use `package/main`, must be `package` :(
        raw_pkg_name := pkg_name

        pkgs, err := packages.Load(loadConfig, strings.Split(raw_pkg_name, ",")...)
        if err != nil {
                panic(err)
        }

        fmt.Println(pkgs)
        //fmt.Println(pkgs[0].PkgPath, "pkg name")

        /*if len(pkgs) > 1 {
                panic("bad pkg length")
        }*/

        if packages.PrintErrors(pkgs) > 0 {
                panic("Failed to load packages")
        }

        for _, p := range pkgs {
                fmt.Println("source file", p.Name) //, p.TypesInfo.Uses)
                fmt.Println(p.GoFiles)
                fmt.Println(p.PkgPath)
        }

	// we need to do like a map of everything being annotated by package i guess
	// like type AnnotatedPackage struct {
	//	Annotations:
	//	Files:
	//	Info:
	//}

	//unused_annotations := []Annotation{}

	//var current_func dst.Node

	// globally store a reference to the package
	//pkg = pkgs[0]

	//annotated_packages := []AnnotatedPackage{}

	for _, pkg := range pkgs {

		pkg_map := map[*dst.File]map[string]ImportDescriptor{}

	        info := pkg.TypesInfo
	        _ = info

		dec := decorator.NewDecoratorWithImports(pkg.Fset, pkg.PkgPath, gotypes.New(info.Uses))
		//dec := decorator.NewDecorator(pkg.Fset)
		files := []*dst.File{}

		annotations := make(map[dst.Node][]Annotation)
		funcs := []dst.Node{}
		consts := []dst.Node{}
		structs := []dst.Node{}

	        // this should always be length 1 since we load one package at a time.

		node_to_files := map[dst.Node]*dst.File{}

		// so we have access to each file in in the package
		for _, og_f := range pkg.Syntax {
			// walk the ast

			f, err := dec.DecorateFile(og_f)
			if err != nil {
				panic(err)
			}
			files = append(files, f)

			file_src := pkg.Fset.File(og_f.Pos())
	                src, err := os.ReadFile(file_src.Name())

	                if err != nil {
				panic(err)
	                }


			pkg_map[f] = map[string]ImportDescriptor{}
			for _, imported := range f.Imports {

				name := ""
				alias := false

				if imported.Name != nil {
					alias = true
					name = imported.Name.String()
				} else {
					mapped, ok := dec.Map.Ast.Nodes[imported]
					if !ok {
						panic("I did oopsie")
					}
					fmt.Println(imported.Name.String())
					obj, ok := info.Implicits[mapped]
					if !ok {

					}
					fmt.Println("yeet")
					name = obj.Name()
				}
				// this leaves \" in the strings. there is probably a good solution here [TODO]

				pkg_map[f][name] = ImportDescriptor{
					Path: strings.Replace(imported.Path.Value, `"`, "", -1),
					Name: name,
					Alias: alias,
				}
			}

			dstutil.Apply(f, func(cr *dstutil.Cursor) bool {
				n := cr.Node()

				if n == nil {
					return true
				}

				// we need to build up function blocks

				switch n.(type) {
					//case *dst.CommentGroup:
					//	comments := []string{}
					//	for _, r_comment := range n.(*dst.CommentGroup).List {
					//		comment := strings.TrimSpace(r_comment.Text)
					//		comments = append(comments, comment)
					//	}
					//	fmt.Println("Found comments", comments)
					//	unused_annotations = append(unused_annotations, extractAnnotations(comments)...)
					case *dst.FuncDecl:

						fmt.Println("Start", n.Decorations().Start)
						annos := extractAnnotations(n.Decorations().Start)
						if len(annos) > 0 {
							annotations[n] = annos

							if entry, ok := FileAnnotationMap[f]; ok {
								FileAnnotationMap[f] = append(entry, annos...)
							} else {
								FileAnnotationMap[f] = append([]Annotation{}, annos...)
							}
						}

						function := n.(*dst.FuncDecl)
						//if len(unused_annotations) > 0 {
						//	// consume annotations
						//	if current_func != nil {
						//		annotations[current_func] = unused_annotations
						//	} else {
						//		annotations[n] = unused_annotations
						//	}
						//	unused_annotations = []Annotation{}
						//}

						//if current_func != nil {
						//	fmt.Println("Block reset by func")
						//}

				                // extract the body
						ast_function := dec.Map.Ast.Nodes[function]
				                _type_start := pkg.Fset.Position(ast_function.Pos()).Offset
				                _type_end := pkg.Fset.Position(ast_function.End()).Offset
				                body := src[_type_start:_type_end]

						Func_descriptors[n] = FuncDescriptor{
							FuncBody: string(body),
							FuncName: function.Name.String(),
							Scope: buildScope(function, dec, info),
							Params: buildTypedFuncParams(function, dec, info),
							Returns: buildTypedFuncReturns(function, dec, info),
							PkgName: pkg.Name,
							PkgPath: pkg.PkgPath,
						}
						fmt.Println("Found function. Block started..", Func_descriptors[n].FuncName)
						fmt.Println("annos", annos)
						//if len(annos) > 0 {
						funcs = append(funcs, n)

						node_to_files[n] = f
						//}
					case *dst.GenDecl:
						fmt.Println("Found gen decl")
						annos := extractAnnotations(n.Decorations().Start)
						if len(annos) > 0 {
							annotations[n] = annos

							if entry, ok := FileAnnotationMap[f]; ok {
								FileAnnotationMap[f] = append(entry, annos...)
							} else {
								FileAnnotationMap[f] = append([]Annotation{}, annos...)
							}
						}

						decl := n.(*dst.GenDecl)
						_type := decl.Tok.String()

						if _type == "type" {
							// hacky way to determine struct
							for _, spec := range decl.Specs {
								if _, ok := spec.(*dst.TypeSpec); ok {
									if _, ok := spec.(*dst.TypeSpec).Type.(*dst.StructType); ok {
										structs = append(structs, n)
										Struct_descriptors[n] = StructDescriptor{
											PkgName: pkg.Name,
											PkgPath: pkg.PkgPath,
										}
										break
									}
								}
							}
						} else if _type == "const" {
							Const_descriptors[n] = ConstDescriptor{
								PkgName: pkg.Name,
								PkgPath: pkg.PkgPath,
							}
							consts = append(consts, n)
						}

						node_to_files[n] = f
				}

				/*if current_func != nil {
					if n.End() <= current_func.End() {
						if entry, ok := func_blocks[current_func]; ok {
							entry = append(entry, n)
							func_blocks[current_func] = entry
						} else {
							func_blocks[current_func] = []dst.Node{n}
						}
					} else {
						// reset block
						fmt.Println("block reset...")
						current_func = nil
					}
					//if len(unused_annotations) > 0 {
					//	annotations[current_func] = append(annotations[current_func], unused_annotations...)
					//	unused_annotations = []Annotation{}
					//}
				}*/

				//printer.Fprint(os.Stdout, pkgs[0].Fset, n)
				return true
			}, nil)
		}

		annotated_packages = append(annotated_packages, AnnotatedPackage{
			Annotations: annotations,
			Funcs: funcs,
			Consts: consts,
			Structs: structs,
			Info: info,
			Dec: dec,
			Files: files,
			PkgName: pkg.Name,
			PkgPath: pkg.PkgPath,
			ImportMap: pkg_map,
			NodeToFiles: node_to_files,
		})
	}

	// so now we have annotations, functions, and their blocks

	// now we have to build any macros
	//for n, nodes := range func_blocks {
	//	if entry, ok := annotations[n]; ok {
	//		output_nodes := buildMacro
	//	}
	//}
	//fmt.Println(func_blocks)

	//fmt.Println(annotations)
	// we used to return this directly for one package. how should we do it for multiple packages..?
	//return annotations, funcs, info, dec, files

	for _, pkg := range annotated_packages {
		imported_packages[pkg.PkgName] = pkg
		fmt.Println(pkg.PkgName, pkg.ImportMap)
		fmt.Println("consts", pkg.Consts)
	}


	return annotated_packages
}

func FindAnnotatedNode(annotation string) (dst.Node, bool) {
	for i := range annotated_packages {
		for n, annos := range annotated_packages[i].Annotations {
			for _, anno := range annos {
				for _, tag := range anno.Params {
					if tag[0] == annotation {
						return n, true
					}
				}
			}
		}
	}
	return nil, false
}

func Compile(code string) (stmts []*dst.ExprStmt) {

	code = "package main; func test() error { " + code + "}"

	fmt.Println("Compiling...")
	fmt.Println(code)

	f, err := decorator.Parse(code)
	if err != nil {
		panic(err)
	}
	node := f.Decls[0].(*dst.FuncDecl)
	for _, n := range node.Body.List {
		new_node := dst.Clone(n)
		stmts = append(stmts, new_node.(*dst.ExprStmt))
	}
	return stmts
}

var MACROS = map[string]func(dst.Node, *types.Info, ...any){}
var MACRO_ANNOTATIONS = map[string][]Annotation{}

func Inject(macro_name string, annotations_json string, f func(dst.Node, *types.Info, ...any)) {
	annos := []Annotation{}
	_ = json.Unmarshal([]byte(annotations_json), &annos)
	MACROS[macro_name] = f
	MACRO_ANNOTATIONS[macro_name] = annos
}

func GetMacro(macro_name string) func(dst.Node, *types.Info, ...any) {
	if f, ok := MACROS[macro_name]; ok {
		return f
	}
	panic("Macro does not exist")
}

func IsMacro(macro_name string) bool {
	fmt.Println("Macro names", MACROS)
	if _, ok := MACROS[macro_name]; ok {
		return true
	}
	return false
}

var ANNOTATIONS map[dst.Node][]Annotation
func GetAnnotations(node dst.Node) []Annotation {
	return ANNOTATIONS[node]
}

func HasWithNode(annotations [][]string) ([]string, bool) {
	fmt.Println("looking for with_node", annotations)
	for _, anno := range annotations {
		if anno[0] == ":with_node" {
			return anno[1:], true
		}
	}
	return nil, false
}

func HasWithPackageScope(annotations [][]string) bool {
	fmt.Println("looking for with_package_scope", annotations)
	for _, anno := range annotations {
		if anno[0] == ":with_package_scope" {
			return true
		}
	}
	return false
}

func HasWithScope(annotations [][]string) bool {
	fmt.Println("looking for with_scope", annotations)
	for _, anno := range annotations {
		if anno[0] == ":with_scope" {
			return true
		}
	}
	return false
}

func getAnnotationValue(node_info []string, name string) string {
	for idx, tag := range node_info {
		if tag == name {
			if idx+1 >= len(node_info) {
				panic("Value for tag `" + name + "` missing")
			}
			return node_info[idx+1]
		}
	}
	panic("Value for tag `" + name + "` missing")
}

func WithNode(node_info []string) dst.Node {
	pkg_name := getAnnotationValue(node_info, ":package")
	func_name := getAnnotationValue(node_info, ":func_name")

	// WithNode is restricted to run on packages for the imported LOCAL modules only.
	for imported_pkg_name, pkg := range imported_packages {
		// find a matching function node in this package
		if imported_pkg_name == pkg_name {
			fmt.Println("pkg found", pkg.Funcs)
			for idx, node := range pkg.Funcs {
				if f, ok := node.(*dst.FuncDecl); ok {
					if f.Name.Name == func_name {
						fmt.Println("with node found")
						return pkg.Funcs[idx]
					}
				}
			}
		}
	}
	panic("WithNode() could not find a suitable node!")
}

var IsLastMap map[string]dst.Node = map[string]dst.Node{}

func BuildMacros(funcs []dst.Node, consts []dst.Node, structs []dst.Node, annotations map[dst.Node][]Annotation, type_info *types.Info) {
	fmt.Println("Building macros")
	fmt.Println(funcs, consts, structs)
	fmt.Println("annos", annotations)

	ANNOTATIONS = annotations

	all_types := append(append(funcs, consts...), structs...)

	// map of which function macro occured last for each macro type so we can group macros
	for idx, _ := range all_types {
		start := all_types[idx]
		for _, annotation_set := range annotations[start] {
			for _, annotation := range annotation_set.Params {
				annotation_name := annotation[0]
				fmt.Println("checking anno", annotation_name)
				if IsMacro(annotation_name) {
					IsLastMap[annotation_name] = all_types[idx] //start
				}
			}
		}

	}

	for idx, _ := range all_types {
		start := all_types[idx]
		for _, annotation_set := range annotations[start] {
			for _, annotation := range annotation_set.Params {
				annotation_name := annotation[0]
				fmt.Println("checking anno", annotation_name)
				if IsMacro(annotation_name) {
					macro := GetMacro(annotation_name)

					others := []any{}
					fmt.Println("with", MACRO_ANNOTATIONS[annotation_name], annotation_name)
					fmt.Println("ans", MACRO_ANNOTATIONS)
					for _, anno := range MACRO_ANNOTATIONS[annotation_name] {
						if node_info, ok := HasWithNode(anno.Params); ok {
							node := WithNode(node_info)
							others = append(others, node)
						}
						/*if HasWithPackageScope(anno.Params) {
							// TODO
							others = append(others, PackageScope{})
						}
						if HasWithScope(anno.Params) {
							// TODO
							fmt.Println("name", annotation_name)
							if f, ok := Func_descriptors[start]; ok {
								fmt.Println("Added scope in!")
								others = append(others, f.Scope)
							} else {
								panic("Scope was not parsed for a macro ..? [bug]")
							}
						}*/
					}
					macro(start, type_info, others...)
					// to chain macros we use output blocks
					//if new_nodes, ok := new_func_blocks[start]; ok {
					//	outputs := macro(new_nodes, type_info)
					//	new_func_blocks[start] = outputs
					//} else {
					//	outputs := macro(nodes, type_info)
					//	new_func_blocks[start] = outputs
					//}
					fmt.Println("Built macro")
				}
			}
		}
	}
}


type Ctx struct {
	ExtraImports []string
	Macros []MacroDescriptor
	PkgName string
	IgnoreFiles []string
	SkipFiles []string
}

type MacroDescriptor struct {
	MacroName string
	FuncName string
	FuncDefinition string
	Imports []string
	Annotations []Annotation
	AnnotationsJson string
	FuncNode dst.Node
}

type FuncDescriptor struct {
	FuncBody string
	FuncName string
	Imports []string
	Scope Scope
	Params []Variable
	Returns []Variable
	PkgName string
	PkgPath string
}

type StructDescriptor struct {
	PkgName string
	PkgPath string
}

type ConstDescriptor struct {
	PkgName string
	PkgPath string
}

var Macro_descriptors []MacroDescriptor
var Func_descriptors  = map[dst.Node]FuncDescriptor{}
var Struct_descriptors  = map[dst.Node]StructDescriptor{}
var Const_descriptors  = map[dst.Node]ConstDescriptor{}

type Annotation struct {
        Params [][]string
}

var FileAnnotationMap map[*dst.File][]Annotation = map[*dst.File][]Annotation{}

func extractAnnotationsOld(comments []string) []Annotation {
	annotations := []Annotation{}
	for _, r_comment := range comments {
		var annotation *Annotation

		comment := strings.TrimSpace(r_comment)
		fmt.Println("Comment->", comment)
		comment = strings.Replace(comment, "// [", "//[", 1)

		//end_offset := int(raw_comment.End())
		if strings.HasPrefix(comment, "//[") &&	strings.HasSuffix(comment, "]") {
			if annotation == nil {
				annotation = &Annotation{}
				annotation.Params = [][]string{}
			}
			comment = comment[3:len(comment)-1]
			params := strings.Split(comment, ",")
			for _, param_pair := range params {
				// handles the case of :annotation(:annotation_param=seomthing&something..)
				if strings.Contains(param_pair, "(") {
					name, after, _ := strings.Cut(param_pair, "(")
					raw_vals, _, _ := strings.Cut(after, ")")
					vals := strings.Split(raw_vals, "=")
					fmt.Println("AM HERE")
					name = strings.TrimSpace(name)
					fmt.Println(name, vals)
					split_vals := []string{}
					for _, val := range vals {
						if strings.Contains(val, "&") {
							splits := strings.Split(val, "&")
							for _, v := range splits {
								split_vals = append(split_vals, strings.TrimSpace(v))
							}
						} else {
							split_vals = append(split_vals, strings.TrimSpace(val))
						}
					}
					annotation.Params = append(annotation.Params, append([]string{name}, split_vals...))
				} else {
					// handles :annotation=something
					if strings.Contains(param_pair, "=") {
						pair := strings.Split(param_pair, "=")
						name := strings.TrimSpace(pair[0])
						val := strings.TrimSpace(pair[1])
						annotation.Params = append(annotation.Params, []string{name, val})
					} else {
						// handles :annotation
						param := strings.TrimSpace(param_pair)
						annotation.Params = append(annotation.Params, []string{param})
					}
				}
			}
			if annotation != nil {
				annotations = append(annotations, *annotation)
			}
		}
	}
	return annotations
}

func extractAnnotations(comments []string) []Annotation {
	annotations := []Annotation{}
	for _, r_comment := range comments {
		var annotation *Annotation

		comment := strings.TrimSpace(r_comment)
		fmt.Println("Comment->", comment)
		comment = strings.Replace(comment, "// [", "//[", 1)

		//end_offset := int(raw_comment.End())
		if strings.HasPrefix(comment, "//[") &&	strings.HasSuffix(comment, "]") {
			sections := []string{}
			children := []bool{}
			section := []byte{}
			idx := 0
			open_paren := false

			comment = comment[3:len(comment)-1]

			fmt.Println("Handling", comment)
			for idx < len(comment) {
				if comment[idx] == ',' && !open_paren{
					if strings.TrimSpace(string(section)) != "" {
						sections = append(sections, strings.TrimSpace(string(section)))
						children = append(children, false)
					}
					section = []byte{}
					idx++
					continue
				}

				if comment[idx] == '(' {
					fmt.Println("opened")
					open_paren = true
					sections = append(sections, strings.TrimSpace(string(section)))
					section = []byte{}
				} else if comment[idx] == ')' {
					fmt.Println("closed")
					open_paren = false
					sections = append(sections, strings.TrimSpace(string(section)))
					children = append(children, true)
					section = []byte{}
				} else {
					section = append(section, comment[idx])
				}

				idx++
			}

			fmt.Println(section)

			if len(sections) == 0 || string(section) != sections[len(sections)-1] {
				sections = append(sections, string(section))
				children = append(children, false)
			}

			for _, sec := range sections {
				fmt.Println("Sec", sec)
			}

			if annotation == nil {
				annotation = &Annotation{}
				annotation.Params = [][]string{}
			}

			for idx, section := range sections {
				// handles the case of :annotation(:annotation_param=seomthing&something..)
				if strings.Contains(section, "(") {
					/*fmt.Println("parsing inner", param_pair)
					idx := strings.LastIndex(param_pair, ")")
					if idx < 0 {
						panic("Closing annotation parenthesis missing!")
					}
					param_pair = param_pair[:idx+1]
					inner_annotations := extractAnnotationsNew([]string{"//["+param_pair+"]"})
					//annos := [][]string{}
					//for _, anno := range inner_annotations {
					//	annos = append(annos, anno.Params...)
					//}
					//annotation.Params = append(annotation.Params, append([]string{name}, annos...))
					fmt.Println("Inner annos", inner_annotations)*/
					panic("Nested tag tuples are not supported at this time!")
				} else {
					for _, param_pair := range strings.Split(section, ",") {
						// handles :annotation=something
						if strings.Contains(param_pair, "=") {
							pair := strings.Split(param_pair, "=")
							name := strings.TrimSpace(pair[0])
							val := strings.TrimSpace(pair[1])
							if idx > 0 && children[idx-1] {
								annotation.Params[len(annotation.Params)-1] = append(annotation.Params[len(annotation.Params)-1], []string{name, val}...)
							} else {
								annotation.Params = append(annotation.Params, []string{name, val})
							}
						} else {
							// handles :annotation
							param := strings.TrimSpace(param_pair)
							if idx > 0 && children[idx-1] {
								annotation.Params[len(annotation.Params)-1] = append(annotation.Params[len(annotation.Params)-1], []string{param}...)
							} else {
								annotation.Params = append(annotation.Params, []string{param})
							}
						}
					}
				}
			}
			if annotation != nil {
				annotations = append(annotations, *annotation)
			}
		}
	}

	// merge annos together
	//for idx, anno := range annotations {
	//	if anno[0] == ":with_node" {
	//		annotations[idx].Params = append(annotations[idx].Params, annotations[idx
	//	}
	//}

	return annotations
}

/*func init() {
	fmt.Println(extractAnnotationsNew([]string{ "//[:macro=:some_macro, :with_node(:package=main, :func_name=main)]"}))
	fmt.Println(extractAnnotationsNew([]string{ "//[:role(kek,kek,1), :zoot]"}))
	panic("k")
}*/

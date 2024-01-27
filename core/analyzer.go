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
	"os/exec"
	"bufio"
	"errors"
	"strconv"
)

const loadMode = packages.NeedName |
    packages.NeedFiles |
    packages.NeedCompiledGoFiles |
    packages.NeedImports |
    packages.NeedDeps |
    packages.NeedTypes |
    packages.NeedSyntax |
    packages.NeedTypesInfo

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

var generated_files = []string{}

func IgnoreUnexpandedPaths() []string {
        paths := []string{}
        err := filepath.Walk(".",
                func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, "_generated.go") {
			maybe_original_file := path[:len(path)-len("_generated.go")] + ".go"
			if _, err := os.Stat(path); err == nil {
				os.Rename(maybe_original_file, maybe_original_file+".buildignore")
				paths = append(paths, maybe_original_file)
			}
		}
                //fmt.Println(path, info.Size())
                return nil
        })

        if err != nil {
                panic(err)
        }


        fmt.Println("Files to ignore during build", paths)

        return paths
}

func BuildOnly() {
	RunCommand([]string{"go", "build"})
}

const macro_dir = ".generated/"

func createMacroDir() {
	path := macro_dir
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func getFreeFileName(used_names []string, name string, nonce int) string {
	for Contains(used_names, name + strconv.Itoa(nonce)) {
		nonce += 1
	}
	return name + strconv.Itoa(nonce)
}

// rename to something else cause this saves the expanded file
func Run(dec *decorator.Decorator, pkg AnnotatedPackage, skip_paths []string) {
//, pkg_path string, info *types.Info) {
	main_file := ""

	createMacroDir()

	// have to add to the current file map!
	file_map := getFileMap()
	used_names := []string{}

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

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

		if strings.Contains(new_name, "/") {
			parts := strings.Split(new_name, "/")
			new_name = parts[len(parts)-1]
		}

		new_name = macro_dir + new_name //+ "_generated.go"

		if Contains(used_names, new_name) {
			new_name = getFreeFileName(used_names, new_name, 0)
		}

		new_name += "_generated.go"

		// old method
		//new_name := strings.Split(og_name, ".go")[0]
		//new_name = new_name + "_generated.go"

		//defer os.Remove(new_name)

		file, err := os.Create(new_name)
		if err != nil {
			panic(err)
		}

		defer file.Close()

		err = r.Fprint(file,f )
		if err != nil {
			panic(err)
		}

		og_name = strings.Split(og_name, cwd+"/")[1]
		file_map[og_name] = new_name

		generated_files = append(generated_files, new_name)
		//decorator.Fprint(file,f )
		// need to find the main file in order to execute it
	}

	if main_file != "" {
		fmt.Println("Running..")
	}


	file, err := os.Create(macro_dir + "map")
	if err != nil {
		panic(err)
	}

	defer file.Close()

	m, _ := json.Marshal(file_map)

	file.Write(m)

}

func RunCommand(args []string) error {

    cmd :=  exec.Command(args[0], args[1:]...)
    stdout, _ := cmd.StdoutPipe()
    stderr, _ := cmd.StderrPipe()
    cmd.Start()
    scanner := bufio.NewScanner(stdout)
    for scanner.Scan() {
        m := scanner.Text()
        fmt.Println(m)
    }
    scanner = bufio.NewScanner(stderr)
    for scanner.Scan() {
        m := scanner.Text()
        fmt.Println(m)
    }

    err := cmd.Wait()
    if err != nil {
	fmt.Println("Failed to execute command", args)
    }
    return err
}

func Clean() {
	fmt.Println("Cleaning up...")
	os.RemoveAll(macro_dir)
	//for _, file := range generated_files {
	//	fmt.Println("Removing generated file", file)
	//	os.Remove(file)
	//}
}

func getFileMap() map[string]string {
    file_map := map[string]string{}
    data, err := os.ReadFile(macro_dir+"map")
    if err != nil {
       fmt.Println("No generated macro files found.")
       return file_map
    }
    //ptr may not be needed
    _ = json.Unmarshal(data, &file_map)
    return file_map
}

func BuildOrRun(build bool, run bool, merge string) {
	var err error

	// this needs to be severely refactored
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	entry_name := "."
	for _, pkg := range Annotated_packages {
		if pkg.PkgName == "main" {
			entry_name = pkg.Dec.Filenames[pkg.Files[0]]
			entry_name = strings.Split(entry_name, cwd+"/")[1]
		}
	}

	if entry_name == "" {
		panic("Could not find an executable go file for package `main`")
	}

	fmt.Println("Executable build/run for", entry_name)

	merge_source := false

	switch merge {
		case "auto":
			merge_source = true
			break
		default:
			// this simply does not work because we need to pass in the subprocess stdin somehow :|
			fmt.Printf("Merge expanded macros into source (y/n): ")
			reader := bufio.NewReader(os.Stdin)
			char, _, err := reader.ReadRune()
			if err != nil {
				fmt.Println("Could not read if should merge or not. Defaulting to NO!")
			}
			if char == 'y' || char == 'Y' {
				merge_source = true
			}
	}

	for og_path, generated_path := range getFileMap() {
		fmt.Println("Swapping", og_path, "with generated macro file", generated_path)
		os.Rename(og_path, strings.Split(og_path, ".go")[0]+".original")
		fmt.Println(og_path,"->",strings.Split(og_path, ".go")[0]+".original")
		raw_path := og_path[:strings.LastIndex(og_path, "/")+1]
		generated_name := strings.Split(generated_path, macro_dir)[1]
		fmt.Println(generated_path,"->",raw_path+generated_name)
		os.Rename(generated_path, raw_path + generated_name)

		if og_path == entry_name {
			entry_name = raw_path + generated_name
			fmt.Println("Executable build/run uses generated file. New entry point:", entry_name)
		}
	}

	// undo the macro expansion to retain the original source code
	defer func() {
		// if build/run errors or not merging, undo the macro expansion to retain the original source code
		if err != nil || !merge_source {
			fmt.Println("Restoring original source code (merge is set to false)")
			for og_path, generated_path := range getFileMap() {
				fmt.Println("Restoring original file", og_path)
				// swap generated back first
				raw_path := og_path[:strings.LastIndex(og_path, "/")+1]
				generated_name := strings.Split(generated_path, macro_dir)[1]
				err := os.Rename(raw_path + generated_name, generated_path)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println(raw_path+generated_name,"->",generated_path)
				os.Rename(strings.Split(og_path, ".go")[0]+".original", og_path)
				fmt.Println(strings.Split(og_path, ".go")[0]+".original","->",og_path)
			}
		} else {
			// delete all of the original files
			// rename the generated files that have been moved in place of the original ones
			fmt.Println("Merging expanded macros into main source code")
			for og_path, generated_path := range getFileMap() {

				raw_path := og_path[:strings.LastIndex(og_path, "/")+1]
				generated_name := strings.Split(generated_path, macro_dir)[1]

				fmt.Println("renaming", raw_path+generated_name,"->",og_path)

				os.Rename(raw_path+generated_name, og_path)
				os.Remove(strings.Split(og_path, ".go")[0]+".original")
			}

		}
	}()

	if build {
		err = RunCommand([]string{"go", "build", entry_name})
	}

	if run {
		if build {
			fmt.Println("Running built executable:" + strings.Split(entry_name,".")[0])
			err = RunCommand([]string{strings.Split(entry_name,".")[0]})
		} else {
			err = RunCommand([]string{"go", "run", entry_name})
		}
	}
}

//var pkg *packages.Package

type AnnotatedPackage struct {
	Annotations map[dst.Node][]Annotation
	Funcs []dst.Node
	Consts []dst.Node
	Structs []dst.Node
	Vars []dst.Node
	Interfaces []dst.Node
	Info *types.Info
	Dec *decorator.Decorator
	Files []*dst.File
	PkgName string
	PkgPath string
	ImportMap map[*dst.File]map[string]ImportDescriptor // import name/alias -> imported path 
	NodeToFiles map[dst.Node]*dst.File
}


//This cannot work since imports are identified based on nodes usage of import identifiers
func AddImport(n dst.Node, name string, import_path string, has_alias bool) {
	for _, pkg := range imported_packages {
		if f, ok := pkg.NodeToFiles[n]; ok {
			pkg.ImportMap[f][import_path] = ImportDescriptor {
				Name: name,
				Alias: has_alias,
				Path: import_path,
			}
			var alias_ident *dst.Ident
			if has_alias {
				alias_ident = &dst.Ident{
					Name: name,
				}
			}

			f.Imports = append(f.Imports, &dst.ImportSpec{
				Name: alias_ident,
				Path: &dst.BasicLit{
					Kind: token.STRING,
					Value: import_path,
				},
			})
		}
	}
}

func IgnoreFiles(ignore_files []string) []string {
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

var Annotated_packages []AnnotatedPackage

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

	// only ignore macro generator here as the ignores are otherwise handled in main
	ignore_files = []string{"macro_generator.go"}
	ignorables := IgnoreFiles(ignore_files)
	for _, file := range ignorables {
		os.Rename(file, file[:len(file)-3])
	}

	defer func() {
		for _, file := range ignorables {
			os.Rename(file[:len(file)-3], file)
		}
		fmt.Println("Restored ignored files:", len(ignorables))
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

	for _, pkg := range pkgs {
        //if packages.PrintErrors(pkgs) > 0 {
		// ignore type errors only since our macros may implement interfaces and generate whole types.
		if len(pkg.Errors) > 0 {
			for _, err := range pkg.Errors {
				if err.Kind != packages.TypeError {
					packages.PrintErrors(pkgs)
					panic("Failed to load packages")
				}
			}
		}
        //        panic("Failed to load packages")
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
		vars := []dst.Node{}
		interfaces := []dst.Node{}

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
					obj, ok := info.Implicits[mapped]
					if !ok {
						panic("bug")
					}
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

				/// ignore generated nodes
				if len(n.Decorations().End) > 0 {
					for _, d := range n.Decorations().End {
						if strings.TrimSpace(d) == "/**generated**/" {
							fmt.Println("Ignoring generated node")
							//n = nil
							cr.Delete()
							break
						}
					}
				}

				// we need to build up function blocks

				switch n.(type) {
					case *dst.FuncDecl:

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

						funcs = append(funcs, n)

						if len(annos) > 0 {
							fmt.Println("Annotated Function:", pkg.Name+"."+Func_descriptors[n].FuncName, len(annos), "annotations")
						}

						node_to_files[n] = f
					case *dst.GenDecl:
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
									} else if _, ok := spec.(*dst.TypeSpec).Type.(*dst.InterfaceType); ok {
										interfaces = append(interfaces, n)
										Interface_descriptors[n] = InterfaceDescriptor{
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
						} else if _type == "var" {
							Var_descriptors[n] = VarDescriptor{
								PkgName: pkg.Name,
								PkgPath: pkg.PkgPath,
							}
							consts = append(consts, n)
						}


						if len(annos) > 0 {
							fmt.Println("Annotated Struct/Const/Var", len(annos), "annotations")
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

		Annotated_packages = append(Annotated_packages, AnnotatedPackage{
			Annotations: annotations,
			Funcs: funcs,
			Consts: consts,
			Structs: structs,
			Vars: vars,
			Interfaces: interfaces,
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

	for _, pkg := range Annotated_packages {
		imported_packages[pkg.PkgName] = pkg
	}


	return Annotated_packages
}

func FindAnnotatedNode(annotation string) (dst.Node, bool) {
	for i := range Annotated_packages {
		for n, annos := range Annotated_packages[i].Annotations {
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

type WithImport struct {
	Alias string
	Path string
}

func Compile(code string, imports ...WithImport) (stmts []dst.Stmt) { //*dst.ExprStmt) {

	new_code := "package main\n"

	for _, i := range imports {
		if strings.TrimSpace(i.Alias) != "" {
			new_code += "import " + i.Alias + " \"" + i.Path + "\"\n"
		} else {
			new_code += "import \"" + i.Path + "\"\n"
		}
	}

	new_code += "\n func test() error { " + code + "}"

	code = new_code

	fmt.Println("Compiling...")
	fmt.Println(code)

	f, err := decorator.Parse(code)
	if err != nil {
		panic(err)
	}

	// index of our func decl is len(imports) basically lol
	node := f.Decls[len(imports)].(*dst.FuncDecl)
	for _, n := range node.Body.List {
		new_node := dst.Clone(n)
		stmts = append(stmts, new_node.(dst.Stmt)) //.(*dst.ExprStmt))
	}
	return stmts
}

func CompileFunctions(code string) (stmts []dst.Decl) { //*dst.ExprStmt) {

	code = "package main;" + code

	fmt.Println("Compiling...")
	fmt.Println(code)

	f, err := decorator.Parse(code)
	if err != nil {
		panic(err)
	}
	return f.Decls
}

var MACROS = map[string]func(Node){} //dst.Node, *types.Info, ...any){}
var MACRO_ANNOTATIONS = map[string][]Annotation{}

func Inject(macro_name string, annotations_json string, f func(Node)) { //dst.Node, *types.Info, ...any)) {
	annos := []Annotation{}
	_ = json.Unmarshal([]byte(annotations_json), &annos)
	MACROS[macro_name] = f
	MACRO_ANNOTATIONS[macro_name] = annos
}

func GetMacro(macro_name string) func(Node) { //dst.Node, *types.Info, ...any) {
	if f, ok := MACROS[macro_name]; ok {
		return f
	}
	panic("Macro does not exist")
}

func IsMacro(macro_name string) bool {
	if _, ok := MACROS[macro_name]; ok {
		return true
	}
	return false
}

var ANNOTATIONS map[dst.Node][]Annotation = map[dst.Node][]Annotation{}

func GetAnnotations(node dst.Node) []Annotation {
	return ANNOTATIONS[node]
}

func HasWithNode(annotations [][]string) ([]string, bool) {
	for _, anno := range annotations {
		if anno[0] == ":with_node" {
			return anno[1:], true
		}
	}
	return nil, false
}

func HasWithPackageScope(annotations [][]string) bool {
	for _, anno := range annotations {
		if anno[0] == ":with_package_scope" {
			return true
		}
	}
	return false
}

func HasWithScope(annotations [][]string) bool {
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

var IsLastMap map[string]int = map[string]int{}

func BuildMacros(funcs []dst.Node, consts []dst.Node, structs []dst.Node, vars []dst.Node, annotations map[dst.Node][]Annotation, type_info *types.Info) {
	fmt.Println("Building macros")
	fmt.Println(funcs, consts, structs, vars)
	fmt.Println("annos", annotations)

	for k, v := range annotations {
		ANNOTATIONS[k] = v
	}

	//ANNOTATIONS = annotations

	all_types := append(append(append(funcs, consts...), structs...), vars...)

	// map of which function macro occured last for each macro type so we can group macros
	for idx, _ := range all_types {
		start := all_types[idx]
		called := map[string]bool{}
		for _, annotation_set := range annotations[start] {
			for _, annotation := range annotation_set.Params {
				annotation_name := annotation[0]
				if IsMacro(annotation_name) && !called[annotation_name] {
					// need to skip duplicate calls to the same macro still [TODO]
					IsLastMap[annotation_name] = IsLastMap[annotation_name] + 1
					called[annotation_name] = true
				} else if IsMacro(annotation_name) && called[annotation_name] {
					fmt.Println("Duplicate macro call found for", start)
					panic("Duplicate macro call")
				}
			}
		}

	}

	for idx, _ := range all_types {
		start := all_types[idx]
		for _, annotation_set := range annotations[start] {
			for _, annotation := range annotation_set.Params {
				annotation_name := annotation[0]
				if IsMacro(annotation_name) {
					macro := GetMacro(annotation_name)

					others := []any{}
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
					n := Node {
						Annotations: ANNOTATIONS[start],
						Types: type_info,
						Extra: others,
						Node: start,
					}
					macro(n) //start, type_info, others...)
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

type Node struct {
	Node dst.Node
	Annotations []Annotation
	Types *types.Info
	Extra []any
}

type Ctx struct {
	ExtraImports []string
	Macros []MacroDescriptor
	PkgName string
	IgnoreFiles []string
	SkipFiles []string
	KeepExpanded bool
	Build bool
	Run bool
	Merge string
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

type VarDescriptor struct {
	PkgName string
	PkgPath string
}

type InterfaceDescriptor struct {
	PkgName string
	PkgPath string
}

var Macro_descriptors []MacroDescriptor
var Func_descriptors  = map[dst.Node]FuncDescriptor{}
var Struct_descriptors  = map[dst.Node]StructDescriptor{}
var Const_descriptors  = map[dst.Node]ConstDescriptor{}
var Var_descriptors  = map[dst.Node]VarDescriptor{}
var Interface_descriptors = map[dst.Node]InterfaceDescriptor{}

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
		comment = strings.Replace(comment, "// [", "//[", 1)

		//end_offset := int(raw_comment.End())
		if strings.HasPrefix(comment, "//[") &&	strings.HasSuffix(comment, "]") {
			sections := []string{}
			children := []bool{}
			section := []byte{}
			idx := 0
			open_paren := false

			comment = comment[3:len(comment)-1]

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
					open_paren = true
					sections = append(sections, strings.TrimSpace(string(section)))
					section = []byte{}
				} else if comment[idx] == ')' {
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

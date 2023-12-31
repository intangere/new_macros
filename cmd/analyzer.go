package main

import (
	"fmt"
	// "go/printer"
	"golang.org/x/exp/slices"
	"flag"
	"text/template"
	"bytes"	
	"os"
	. "github.com/intangere/new_macros/core"
	"strings"
	"encoding/json"
	"bufio"
	//"go/token"

	"io/ioutil"

	"golang.org/x/tools/go/packages"

        "github.com/dave/dst/decorator"
//        "github.com/dave/dst/decorator/resolver/simple"
//        "github.com/dave/dst/decorator/resolver/goast"
)

/*
backing this up
		{{- range $_, $macro := .Macros }}
			{{$macro.FuncDefinition}}
		{{- end}}
*/

func asStrings(strs []string) []string {
	wrapped_strs := []string{}
	for _, str := range strs {
		wrapped_strs = append(wrapped_strs, `"` + str + `"`)
	}
	return wrapped_strs
}

func main() {

        pkg_name := flag.String("package", "", "The name of the package(s) to parse which should be parts of the module in the current directory i.e my_packge, my_package/something, or ./... to load all local packagesw")
	maybe_ignore_files := flag.String("ignore", "", "Files to ignore from being parsed by go. Regex pattern matching support via r: prefix")
	maybe_skip_files := flag.String("ignore_outputs", "", "Skip generating files that don't have their contents changed i.e macros that do not output code (this is a temporary fix). Regex pattern matching support via r: prefix")
	should_build := flag.Bool("build", true, "Build go executable after expanding macros")
	should_run := flag.Bool("run", false, "Run go program after expanding macros. do not use this. exitting from the running program will not call the required defers and will have to manually fix your source")
	keep_expanded := flag.Bool("keep_generated", false, "Keep the generated files")
	clean := flag.Bool("clean", false, "Remove all generated files. This removes every file matching *_generated.go recursively")
	build_only := flag.Bool("build_only", false, "Build the current source code using the original source code or with the already expanded macros from -keep_expanded")
        flag.Parse()

	if *clean {
		fmt.Println("Cleaning generated files")
		os.RemoveAll(".generated/")
		return
	}

	// ignore generate files and anything we generate!
	ignore_files := []string{}

	if !*build_only{
		ignore_files = append(ignore_files, []string{"r:(.*)_generated.go$", "macro_generator.go"}...)
	}

	skip_files := []string{}

	if *maybe_ignore_files != "" {
		ignore_files = append(ignore_files, strings.Split(*maybe_ignore_files, ",")...)
	}

	if *maybe_skip_files != "" {
		skip_files = append(skip_files, strings.Split(*maybe_skip_files, ",")...)
	}

	// ignore all files.
	// this must take place here for several reasons
	// if its in Build() or per package and you run other code that works on those packages
	// the files won't be ignored anymore and it gets messy.
        ignorables := IgnoreFiles(ignore_files)
        for _, file := range ignorables {
                os.Rename(file, file[:len(file)-3]+".original")
        }

        defer func() {   
                for _, file := range ignorables {
                        os.Rename(file[:len(file)-3]+".original", file)
                }
                fmt.Println("Restored ignored files")
        }()

	if *build_only {
		/*paths := IgnoreUnexpandedPaths()
		defer func() {
			for _, file := range paths {
				os.Rename(file+".buildignore", file)
				fmt.Println("Restored ignore build file", file)
			}
		}()
		BuildOnly()*/
		BuildOrRun(true, false)
		return
	}

	//annotations, _, _, _, _ := Build(*pkg_name, ignore_files)
	annotated_packages := Build(*pkg_name, ignore_files)

	// create a build script for the package

	used := map[string]struct{}{
                    "github.com/intangere/new_macros/core" : struct{}{},
                    "github.com/intangere/new_macros/helpers"  : struct{}{},
                    "go/types"   : struct{}{},
                    "go/token" : struct{}{},
                    "github.com/dave/dst" : struct{}{},
	}

	imports_required := map[string]string{
                    "github.com/intangere/new_macros/core" : "core",
                    "github.com/intangere/new_macros/helpers"  : "helpers",
                    "go/types"   : "types",
                    "go/token" : "token",
                    "github.com/dave/dst" : "dst",
	}

	tmpl_data := `package main

		import(
                    "github.com/intangere/new_macros/core"
                    "github.com/intangere/new_macros/helpers"
		    "go/types"
		    "go/token"
                    "github.com/dave/dst"
                    {{- range $_, $val := .ExtraImports }}
                         "{{$val}}"
                    {{- end }}
			"some testing"
	   	    #INJECTPKGSHERE#
                )

		{{- range $_, $macro := .Macros }}
			{{$macro.FuncDefinition}}
		{{- end}}

		// inject macro definitions here
		// and their imports
		func main() {
			{{ if not .KeepExpanded }}
				defer core.Clean()
			{{- end }}
			{{- range $_, $macro := .Macros }}
			core.Inject("{{.MacroName}}", {{.AnnotationsJson}}, {{.FuncName}})
			{{- end}}
			{{ $wrapped_ignores := AsStrings .IgnoreFiles }}
			{{ $wrapped_skips := AsStrings .SkipFiles }}
			annotated_packages := core.Build("{{.PkgName}}", []string{ {{ StringsJoin $wrapped_ignores "," }} })
                        for _, pkg := range annotated_packages {
				core.BuildMacros(pkg.Funcs, pkg.Consts, pkg.Structs, pkg.Vars, pkg.Annotations, pkg.Info)
			}
			// now we need to inject/overwite the generated nodes back into the ast
			//core.InjectBlocks(new_func_blocks)
			for _, pkg := range annotated_packages {
				if len(pkg.Annotations) > 0 {
					core.Run(pkg.Dec, pkg, []string { {{ StringsJoin $wrapped_skips "," }} })
				}
			}
			{{ if or .Build .Run }}
				core.BuildOrRun({{.Build}}, {{.Run}})
			{{- end }}
		}
	`

        tmpl, err := template.New("test").Funcs(template.FuncMap{"StringsJoin": strings.Join, "AsStrings": asStrings}).Parse(tmpl_data)

        if err != nil {
                panic(err)
        }

        var b bytes.Buffer

	imports := []string{}
	codes := []string{}

	// build macro descriptors

	for _, pkg := range annotated_packages {
		//for start, annotation_set := range annotations {
		for start, annotation_set := range pkg.Annotations {
			for _, annotation_sub_set := range annotation_set {
				for _, annotation := range annotation_sub_set.Params {
					if annotation[0] == ":macro" {
						// save the last iterated node for a macro. we know this is the last occurance on the last loop
						// a little hacker man
						anno_json, _ := json.Marshal(annotation_set)
						anno_json, _ = json.Marshal(string(anno_json))
						Macro_descriptors = append(Macro_descriptors, MacroDescriptor{
							//scary stuff lmfao
							FuncNode: start,
							FuncDefinition: strings.Replace(Func_descriptors[start].FuncBody, Func_descriptors[start].FuncName, Func_descriptors[start].FuncName+"__macro", -1), // this is heavily broken and will replace atrbitrary text
							FuncName: Func_descriptors[start].FuncName + "__macro",
							MacroName: annotation[1],
							Annotations: annotation_set,
							AnnotationsJson: string(anno_json),
						})
					}
				}
			}
		}
	}

	for _, md := range Macro_descriptors {
		for _, imp := range md.Imports {
			if !slices.Contains(imports, imp) {
				imports = append(imports, imp)
			}
		}
		if !slices.Contains(codes, md.FuncDefinition) {
			codes = append(codes, md.FuncDefinition)
		}
	}

	//fmt.Println(Macro_descriptors)

	// we need to parse the template using dst
	// restore it using a map composed of all the file imports from all the macros


        err = tmpl.Execute(&b, Ctx{
		ExtraImports: imports,
		Macros: Macro_descriptors,
		PkgName: *pkg_name,
		IgnoreFiles: ignore_files,
		SkipFiles: skip_files,
		KeepExpanded: *keep_expanded,
		Run: *should_run,
		Build: *should_build,
	})

        if err != nil {
                panic(err)
        }

	dir, err := ioutil.TempDir(".", "prefix")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	err = os.WriteFile(dir + "/go.mod", []byte("module root"), 0644)
        if err != nil {
                panic(err)
	}

	//dec := decorator.NewDecorator(token.NewFileSet()) //WithImports(token.NewFileSet(), "main", goast.New())

	raw := b.Bytes()
	extra_imports := ""

	import_map := []map[string]string{
		imports_required,
	}
	nonce := 0

	if len(Macro_descriptors) > 0 {
		for _, macro := range Macro_descriptors {
			found := false
			for _, pkg := range annotated_packages {
				if f1, ok := pkg.NodeToFiles[macro.FuncNode]; ok {
					imps := pkg.ImportMap[f1]
					sub_import_map := map[string]string{}
					for k,d := range imps {
						v := d.Path
						if !d.Alias {
							if _, ok := used[v]; !ok {
								extra_imports += `"`+v+"\"\n"
								used[v]=struct{}{}
							}
						} else {
							if _, ok := used[k+ `"`+v+`"`]; !ok {
								extra_imports += k + ` "`+v+"\"\n"
								used[k+ `"`+v+`"`]=struct{}{}
							}
						}

						//if k1, ok := import_map[v]; ok {
						//	if k1 != k {
						//		//panic("Conflicting import aliases for import `"+ v +"`. Imported as both " + k1 + " and " + k)
						//		import_map[v+fmt.Sprintf("-[conflict%d]", nonce)] = k
						//	}
						//} else {
						sub_import_map[v] = k
						//}
					}
					import_map = append(import_map, sub_import_map)
					found = true
				}
			}
			if !found {
				panic("!")
			}
			nonce++
		}
	} else {
		// time to start guessing imports LMFAO
		for _, pkg := range annotated_packages {
			for _, f1 := range pkg.Files {
				imps := pkg.ImportMap[f1]
				sub_import_map := map[string]string{}
				for k,d := range imps {
					v := d.Path
					if !d.Alias {
						if _, ok := used[v]; !ok {
							extra_imports += `"`+v+"\"\n"
							used[v]=struct{}{}
						}
					} else {
						if _, ok := used[k+ `"`+v+`"`]; !ok {
							extra_imports += k + ` "`+v+"\"\n"
							used[k+ `"`+v+`"`]=struct{}{}
						}
					}

					//if k1, ok := import_map[v]; ok {
					//	if k1 != k {
					//		//panic("Conflicting import aliases for import `"+ v +"`. Imported as both " + k1 + " and " + k)
					//		import_map[v+fmt.Sprintf("-[conflict%d]", nonce)] = k
					//	}
					//} else {
					sub_import_map[v] = k
					//}
				}
				import_map = append(import_map, sub_import_map)
			}
		}
	}

	raws := strings.Replace(string(raw), "#INJECTPKGSHERE#", extra_imports, -1)

	err = os.WriteFile(dir + "/main.go", []byte(raws), 0644)
        if err != nil {
                panic(err)
	}

	// Use the Load convenience function that calls go/packages to load the package. All loaded
	// ast files are decorated to dst.
	pkgs, err := decorator.Load(&packages.Config{Dir: dir, Mode: packages.LoadSyntax}, "root")
	if err != nil {
		panic(err)
	}

	f := pkgs[0].Syntax[0]

        r := decorator.NewRestorerWithImports("main", NewConflictResolver(import_map))

	var b1 bytes.Buffer
	c := bufio.NewWriter(&b1)

	err = r.Fprint(c, f)
	if err != nil {
		panic(err)
	}

	c.Flush()

	// create a go file and run it
	file_name := "macro_generator.go"
	err = os.WriteFile(file_name, b1.Bytes(), 0644)
        if err != nil {
                panic(err)
        }
	defer os.Remove(file_name)

	RunCommand([]string{"go", "run", file_name})
}

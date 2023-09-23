package main

import "github.com/intangere/new_macros/core"

// inject macro definitions here
// and their imports
func main() {
	annotated_packages := core.Build("", []string{"r:(.*)_generated.go$", "macro_generator.go"})
	for _, pkg := range annotated_packages {
		core.BuildMacros(pkg.Funcs, pkg.Consts, pkg.Structs, pkg.Annotations, pkg.Info)
	}
	// now we need to inject/overwite the generated nodes back into the ast
	//core.InjectBlocks(new_func_blocks)
	for _, pkg := range annotated_packages {
		if len(pkg.Annotations) > 0 {
			core.Run(pkg.Dec, pkg, []string{})
		}
	}
}

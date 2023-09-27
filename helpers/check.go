package helpers

import "github.com/intangere/new_macros/core"
import "github.com/dave/dst"

func IsLast(node_count int, macro_name string) bool {
	if macro_count, ok := core.IsLastMap[macro_name]; ok {
		if node_count == macro_count {
			return true
		}
	}
	return false
}

func InterfaceExists(pkg_path string, interface_name string) bool {
	for _, pkg := range core.Annotated_packages {
		if pkg.PkgPath == pkg_path {
			for _, intr := range pkg.Interfaces {
				for _, spec := range intr.(*dst.GenDecl).Specs {
					if spec.(*dst.ValueSpec).Names[0].String() == interface_name {
						return true
					}
				}
			}
		}
	}
	return false
}

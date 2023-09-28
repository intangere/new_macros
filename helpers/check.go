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
					if spec.(*dst.TypeSpec).Name.String() == interface_name {
						return true
					}
				}
			}
		}
	}
	return false
}

func IsMethodImplemented(pkg_path string, type_name string, method_name string) bool {
	for _, pkg := range core.Annotated_packages {
		if pkg.PkgPath == pkg_path {
			for _, file := range pkg.Files {
				for _, decl := range file.Decls {
					if f, ok := decl.(*dst.FuncDecl); ok {
						if f.Recv != nil {
							method_type_name := f.Recv.List[0].Type.String()
							// remove pointer
							if method_type_name[0] == "*" {
								method_type_name = method_type_name[1:]
							}
							if method_type_name == method_name {
								return true
							}
						}
					}
				}
			}
			return false
		}
	}
	return false
}

/*func IsInterfaceImplemented(interface_definition *dst.GenDecl, our_node *dst.GenDecl) {
	// we need to check function signatures

}*/

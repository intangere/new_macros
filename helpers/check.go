package helpers

import "github.com/intangere/new_macros/core"
import "github.com/dave/dst"
import "fmt"

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

// need to define UnwrapStar() which will recuserively iterate to the base type

func IsMethodImplemented(pkg_path string, type_name string, method_name string) bool {
	for _, pkg := range core.Annotated_packages {
		if pkg.PkgPath == pkg_path {
			for _, file := range pkg.Files {
				for _, decl := range file.Decls {
					if f, ok := decl.(*dst.FuncDecl); ok {
						if f.Recv != nil {
							if star, ok := f.Recv.List[0].Type.(*dst.StarExpr); ok {
								method_type_name := star.X.(*dst.Ident).String()
								fmt.Println(method_type_name, method_name)
								if method_type_name == method_name {
									return true
								}
							} else {
								method_type_name := f.Recv.List[0].Type.(*dst.Ident).String()
								if method_type_name == method_name {
									return true
								}
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

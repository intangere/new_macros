package helpers

import "github.com/dave/dst"
import "go/token"
import "github.com/intangere/new_macros/core"

func MethodCall(pkg_name string, method string, args ...dst.Expr) *dst.ExprStmt {
  return &dst.ExprStmt{
    X: &dst.CallExpr{
      Fun: &dst.SelectorExpr{
        X: &dst.Ident{
          Name: pkg_name,
        },
        Sel: &dst.Ident{
          Name: method,
        },
      },
      Args: args,
      },
    }
}

func FuncCall(func_name string, args ...dst.Expr) *dst.ExprStmt {
  return &dst.ExprStmt{
    X: &dst.CallExpr{
      Fun: &dst.Ident{
          Name: func_name,
      },
      Args: args,
      },
    }
}

func FuncDecl(func_name string) *dst.FuncDecl {
	return &dst.FuncDecl{
		Name: &dst.Ident{
			Name: func_name,
		},
		Type: &dst.FuncType{
			Func: true,
			
		},
	}
}

func FuncLit(args []*dst.Field, returns []*dst.Field) *dst.FuncLit {
/*	return &dst.ExprStmt{
		X: &dst.FuncLit{
			Type: &dst.FuncType {
				Params: &dst.FieldList{
					List: args,
				},
				Results: &dst.FieldList{
					List: returns,
				},
			},
		},
	}*/
	return &dst.FuncLit{
		Type: &dst.FuncType {
			Params: &dst.FieldList{
				List: args,
			},
			Results: &dst.FieldList{
				List: returns,
			},
		},
		Body: &dst.BlockStmt{

		},
	}
}

func BasicField(name string, _type string) *dst.Field{
	return &dst.Field{
		Names: []*dst.Ident{
			&dst.Ident{
				Name: name,
			},
		},
		Type: &dst.Ident{
			Name: _type,
		},
	}
}

func BasicUnnamedField(_type string) *dst.Field{
	return &dst.Field{
		Type: &dst.Ident{
			Name: _type,
		},
	}
}

func SelectorField(name string, pkg string, _type string) *dst.Field{
	return &dst.Field{
		Names: []*dst.Ident{
			&dst.Ident{
				Name: name,
			},
		},
		Type: &dst.SelectorExpr{
			Sel: &dst.Ident{
				Name: _type,
			},
			X: &dst.Ident{
				Name: pkg,
			},
		},
	}
}

func SelectorUnnamedField(pkg string, _type string) *dst.Field{
	return &dst.Field{
		Type: &dst.SelectorExpr{
			Sel: &dst.Ident{
				Name: _type,
			},
			X: &dst.Ident{
				Name: pkg,
			},
		},
	}
}

func String(str string) dst.Expr {
     return  &dst.BasicLit{
                  Kind:  token.STRING,
                  Value: `"` + str + `"`,
                }
}

func Fields(fields ...*dst.Field) []*dst.Field {
	return fields
}

func Return(args ...dst.Expr) *dst.ReturnStmt {
	return &dst.ReturnStmt{
		Results: args,
	}
}

func Ident(name string) *dst.Ident {
	return &dst.Ident{
		Name: name,
	}
}

func InsertAfter(body []dst.Stmt, after_node dst.Stmt, to_insert ...dst.Stmt) []dst.Stmt {
	// this function is used to insert a node AFTER another node in a statement block
	for i := range body {
		if body[i] == after_node {
			i++
			for j := range to_insert {
				body = append(body[:i+j+1], body[i+j:]...)
				body[i+j] = to_insert[j]
			}
			return body
		}
	}
	panic("Could not find node to insert after")
}

func CallExprAsExprStmt(stmt *dst.CallExpr) *dst.ExprStmt {
	return &dst.ExprStmt{
		X: stmt,
	}
}

func InsertAfterNode(insert_after *dst.GenDecl, new_nodes []dst.Decl) {

	for _, pkg := range core.Annotated_packages {
		for _, f := range pkg.Files {
			for i := range f.Decls {
				if f.Decls[i] == insert_after {
					f.Decls = append(f.Decls[:i], append(new_nodes, f.Decls[i:]...)...)
					return
				}
			}
		}
	}

	panic("Insert after failed miserably. This is either a case of misuse or an internal bug.")
}

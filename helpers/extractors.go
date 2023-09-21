package helpers

import (
	"github.com/dave/dst"
	"fmt"
	"github.com/intangere/new_macros/core"
	"strings"
)

func WithNode(extra []any) dst.Node {
	// return the FIRST node found passed to our macro.

	var destination_node dst.Node

        found := false
        for _, param := range extra {
                if node, ok := param.(dst.Node); ok {
                        destination_node = node
                        found = true
                        break
                }
        }

        if !found {
                panic("Could not generate macro. Destination node missing")
        }

	return destination_node
}

func FindMethodCall(f *dst.FuncDecl, pkg_name string, method_name string) []dst.Stmt {
	// this should also take type info and compare package names instead of the call expr aliases
	// This ONLY extracts methodcalls from assign statements and direct function calls
	calls := []dst.Stmt{}

	// i am truly not sure how error prone this is but typing out all the error checks rn is not going to happen :(
	for _, n := range f.Body.List {
		if expr_stmt, ok := n.(*dst.ExprStmt); ok {
			if call_stmt, ok := expr_stmt.X.(*dst.CallExpr); ok {
				fmt.Println("Got call expr")
				if _, ok := call_stmt.Fun.(*dst.SelectorExpr); ok {
					fmt.Println(call_stmt.Fun.(*dst.SelectorExpr).X.(*dst.Ident).Name, pkg_name)
					fmt.Println(call_stmt.Fun.(*dst.SelectorExpr).Sel.Name, method_name)
					if call_stmt.Fun.(*dst.SelectorExpr).X.(*dst.Ident).Name == pkg_name && call_stmt.Fun.(*dst.SelectorExpr).Sel.Name == method_name {
						calls = append(calls, n)
					}
				}
			}
		} else if assign_stmt, ok := n.(*dst.AssignStmt); ok {
			for _, stmt := range assign_stmt.Rhs {
				fmt.Println(stmt)
				if call_stmt, ok := stmt.(*dst.CallExpr); ok {
					fmt.Println("Got call expr BIGGG")
					if _, ok := call_stmt.Fun.(*dst.SelectorExpr); ok {
						fmt.Println(call_stmt.Fun.(*dst.SelectorExpr).X.(*dst.Ident).Name, pkg_name)
						fmt.Println(call_stmt.Fun.(*dst.SelectorExpr).Sel.Name, method_name)
						if call_stmt.Fun.(*dst.SelectorExpr).X.(*dst.Ident).Name == pkg_name && call_stmt.Fun.(*dst.SelectorExpr).Sel.Name == method_name {
							calls = append(calls, n)
						}
					} else {
						fmt.Println("Not select")
						fmt.Println(call_stmt.Fun.(*dst.Ident).Path, pkg_name)
						fmt.Println(call_stmt.Fun.(*dst.Ident).Name, method_name)
						if call_stmt.Fun.(*dst.Ident).Path == pkg_name && call_stmt.Fun.(*dst.Ident).Name == method_name {
							calls = append(calls, n)
						}

					}
				}
			}
		}
	}
	return calls
}

func FindAssignmentIdentByType(f *dst.FuncDecl, pkg_name string, method_name string) string {
	for _, n := range f.Body.List {
		if assign_stmt, ok := n.(*dst.AssignStmt); ok {
			for _, stmt := range assign_stmt.Rhs {
				fmt.Println(stmt)
				if call_stmt, ok := stmt.(*dst.CallExpr); ok {
					fmt.Println("Got call expr BIGGG")
					if _, ok := call_stmt.Fun.(*dst.SelectorExpr); ok {
						fmt.Println(call_stmt.Fun.(*dst.SelectorExpr).X.(*dst.Ident).Name, pkg_name)
						fmt.Println(call_stmt.Fun.(*dst.SelectorExpr).Sel.Name, method_name)
						if call_stmt.Fun.(*dst.SelectorExpr).X.(*dst.Ident).Name == pkg_name && call_stmt.Fun.(*dst.SelectorExpr).Sel.Name == method_name {
							return assign_stmt.Lhs[0].(*dst.Ident).Name
						}
					} else {
						fmt.Println("Not select")
						fmt.Println(call_stmt.Fun.(*dst.Ident).Path, pkg_name)
						fmt.Println(call_stmt.Fun.(*dst.Ident).Name, method_name)
						if call_stmt.Fun.(*dst.Ident).Path == pkg_name && call_stmt.Fun.(*dst.Ident).Name == method_name {
							return assign_stmt.Lhs[0].(*dst.Ident).Name
						}

					}
				}
			}
		}
	}
	panic("Could not find assigned node by type")
}

func GetOverlappedVariables(local_scope core.Scope, package_scope core.Scope, typed_params []core.Variable) []core.Variable {
	// variables is ordered to the function parameters
	vars := []core.Variable{}
	for _, param := range typed_params {
		found := false
		for _, local_var := range local_scope.Variables {
			if local_var.BasicType == param.BasicType {
				if found {
					panic("Ambiguous variable. One or more local variables with the same types found!")
				}
				vars = append(vars, local_var)
				found = true
			}
		}

		if found{
			continue
		}

		for _, pkg_var := range package_scope.Variables {
			if pkg_var.BasicType == param.BasicType {
				vars = append(vars, pkg_var)
				found = true
				break
			}
		}

		if !found {
			panic("Could not find local or package level variable to populate function call!" + " name: " + param.Name)
		}
	}

	return vars
}

func GetOverlappedReturns(local_scope core.Scope, package_scope core.Scope, return_types []core.Variable, typed_params []core.Variable) []core.Variable {
	// variables is ordered to the function parameters
	vars := []core.Variable{}

	fmt.Println("looking for", return_types)
	fmt.Println("Using", local_scope, package_scope, typed_params)
	for _, ret := range return_types {
		found := false
		for _, param := range typed_params {
			if param.BasicType == ret.BasicType {
				if found {
					panic("Ambiguous variable. One or more local variables with the same types found!")
				}
				vars = append(vars, param)
				found = true
			}
		}

		for _, local_var := range local_scope.Variables {
			if local_var.BasicType == ret.BasicType {
				if found {
					panic("Ambiguous variable. One or more local variables with the same types found!")
				}
				vars = append(vars, local_var)
				found = true
			}
		}

		if found {
			continue
		}

		for _, pkg_var := range package_scope.Variables {
			if pkg_var.BasicType == ret.BasicType {
				vars = append(vars, pkg_var)
				found = true
				break
			}
		}

		if !found {
			fmt.Println("For", ret)
			panic("Could not find local or package level variable to be used as a new variable returned from function call!")
		}
	}
	return vars
}

func GetTypedParams(node dst.Node) []core.Variable {
        if f, ok := core.Func_descriptors[node]; ok {
		fmt.Println("F PARAMS", f.Params)
                return f.Params
        }
        panic("TypedParams not found..?")
}

func GetReturns(node dst.Node) []core.Variable {
        if f, ok := core.Func_descriptors[node]; ok {
                return f.Returns
        }
        panic("TypedReturns not found..?")
}

func JoinVarNames(vars []core.Variable) string {
        names := []string{}
        for _, v := range vars {
                names = append(names, v.Name)
        }

        return strings.Join(names, ",")
}

func HasErrorVar(vars []core.Variable) bool {
	for _, v := range vars {
		if v.BasicType == "error" {
			return true
		}
	}
	return false
}

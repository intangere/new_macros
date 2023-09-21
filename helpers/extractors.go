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

func contains[T comparable](s []T, n T) bool {
	for _, v := range s {
		if v == n {
			return true
		}
	}
	return false
}

func GetOverlappedVariables(scopes []core.Scope, typed_params []core.Variable, ignore_indexes ...int) []core.Variable {
	// variables is ordered to the function parameters
	vars := []core.Variable{}
	for idx, param := range typed_params {
		if contains(ignore_indexes, idx) {
			continue
		}

		found := false
		for _, scope := range scopes {
			for _, var_ := range scope.Variables {
				if var_.BasicType == param.BasicType {
					if found {
						panic("Ambiguous variable. One or more local variables with the same types found! " + var_.BasicType)
					}
					vars = append(vars, var_)
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			panic("Could not find local or package level variable to populate function call!" + " name: " + param.Name)
		}
	}

	return vars
}

// local scope should take precedence even in the case of ambigous variables!

func GetOverlappedReturns(scopes []core.Scope, return_types []core.Variable, typed_params []core.Variable) []core.Variable {
	// variables is ordered to the function parameters
	vars := []core.Variable{}

	fmt.Println("looking for", return_types)
	fmt.Println("Using", scopes, typed_params)
	for _, ret := range return_types {
		found := false
		for _, param := range typed_params {
			if param.BasicType == ret.BasicType {
				if found {
					panic("Ambiguous variable. One or more local variables with the same types found!" + param.BasicType)
				}
				vars = append(vars, param)
				found = true
			}
		}

		if found {
			continue
		}

		for _, scope := range scopes {
			for _, var_ := range scope.Variables {
				if var_.BasicType == ret.BasicType {
					if found {
						panic("Ambiguous variable. One or more local variables with the same types found! " + var_.BasicType)
					}
					vars = append(vars, var_)
					found = true
					break
				}
			}
			if found {
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

func GetTypedParamByName(params []core.Variable, name string) core.Variable {
	for _, param := range params {
		if param.Name == name {
			return param
		}
	}
        panic("TypedParam by name not found..?")
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

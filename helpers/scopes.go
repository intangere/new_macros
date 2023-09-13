package helpers

import (
	"github.com/dave/dst"
	"github.com/intangere/new_macros/core"
)

func GetScope(node dst.Node) core.Scope {
	if f, ok := core.Func_descriptors[node]; ok {
		return f.Scope
	}
	panic("Scope not found. Did you forget the :with_scope tag?")
}

func GetPackageScope(extra []any) core.PackageScope {
	for _, obj := range extra {
		if scope, ok := obj.(core.PackageScope); ok {
			return scope
		}
	}
	panic("Package scope not found. Did you forget the :with_package_scope tag?")
}

func NewVariable(scope *core.Scope, name string, _type string) {
	if v, ok := scope.Variables[name]; ok {
		if v.BasicType != _type {
			panic("Scope already contains a variable with the same name: " + name + " but type is different: " + _type)
		}
	} else {
		scope.Variables[name] = core.Variable{
			Name: name,
			BasicType: _type,
		}
	}
}

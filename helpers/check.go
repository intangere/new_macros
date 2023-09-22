package helpers

import "github.com/intangere/new_macros/core"
import "github.com/dave/dst"

func IsLast(d dst.Node, macro_name string) bool {
	if n, ok := core.IsLastMap[macro_name]; ok {
		if n == d {
			return true
		}
	}
	return false
}

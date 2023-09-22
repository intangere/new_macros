package helpers

import "github.com/intangere/new_macros/core"
//import "github.com/dave/dst"

func IsLast(node_count int, macro_name string) bool {
	if macro_count, ok := core.IsLastMap[macro_name]; ok {
		if node_count == macro_count {
			return true
		}
	}
	return false
}

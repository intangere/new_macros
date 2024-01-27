package main

import (
	"fmt"

	"github.com/dave/dst"
	"github.com/intangere/new_macros/core"
	"github.com/intangere/new_macros/helpers"
)

// [:macro=:hello]
func hello(n core.Node) {
	_ = dst.Node(nil)

	// get the annotated nodes annotations
	message := "\"Hello world!\""
	if maybe_message, ok := helpers.GetTagValue(n.Annotations, ":message"); ok {
		message = maybe_message
	}

	stmts := core.Compile("fmt.Println("+message+")", n.Node, core.WithImport{
		Path:  "fmt",
		Alias: "fmt",
	})

	func_decl, _ := n.Node.(*dst.FuncDecl)
	func_decl.Body.List = append(func_decl.Body.List, stmts...)
}

// [:hello]
func test() {
}

// [:hello, :message="Some message to print"]
func main() {
	test()
}

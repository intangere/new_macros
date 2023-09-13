package core

import "github.com/dave/dst/decorator/resolver"
import "strings"

type RestorerResolver []map[string]string

func NewConflictResolver(m []map[string]string) RestorerResolver {
	return RestorerResolver(m)
}

func (r RestorerResolver) ResolvePackage(importPath string) (string, error) {
	println("RESOLVER CALLED")
	// search all possible import maps to try to resolve conflicting imports >:)
	for _, m := range r {
		if n, ok := m[importPath]; ok {
			if strings.Contains(n, "[conflict]"){
				return n[:len(n)-10], nil
			}

			return n, nil
		}
	}
	return "", resolver.ErrPackageNotFound
}

// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"cmd/internal/obj"
	"cmd/internal/objabi"
	"fmt"
	"sort"
)

// pkgMap maps a package path to a package.
var pkgMap = make(map[string]*Pkg)

type Pkg struct {
	Path     string // string literal used in import statement, e.g. "runtime/internal/sys"
	Name     string // package name, e.g. "sys"
	Pathsym  *obj.LSym
	Prefix   string // escaped path for use in symbol table
	Imported bool   // export data of this package was parsed
	Direct   bool   // imported directly
	Syms     map[string]*Sym
}

// NewPkg returns a new Pkg for the given package path and name.
// Unless name is the empty string, if the package exists already,
// the existing package name and the provided name must match.
func NewPkg(path, name string) *Pkg {
	if p := pkgMap[path]; p != nil {
		if name != "" && p.Name != name {
			panic(fmt.Sprintf("conflicting package names %s and %s for path %q", p.Name, name, path))
		}
		return p
	}

	p := new(Pkg)
	p.Path = path
	p.Name = name
	p.Prefix = objabi.PathToPrefix(path)
	p.Syms = make(map[string]*Sym)
	pkgMap[path] = p

	return p
}

// ImportedPkgList returns the list of directly imported packages.
// The list is sorted by package path.
func ImportedPkgList() []*Pkg {
	var list []*Pkg
	for _, p := range pkgMap {
		if p.Direct {
			list = append(list, p)
		}
	}
	sort.Sort(byPath(list))
	return list
}

type byPath []*Pkg

func (a byPath) Len() int           { return len(a) }
func (a byPath) Less(i, j int) bool { return a[i].Path < a[j].Path }
func (a byPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

var nopkg = &Pkg{
	Syms: make(map[string]*Sym),
}

// fake package for runtime type info (headers)
var typepkg = NewPkg("type", "type")

func TypePkgLookup(name string) *Sym {
	return typepkg.Lookup(name)
}

func (pkg *Pkg) Lookup(name string) *Sym {
	s, _ := pkg.LookupOK(name)
	return s
}

var InitSyms []*Sym

// LookupOK looks up name in pkg and reports whether it previously existed.
func (pkg *Pkg) LookupOK(name string) (s *Sym, existed bool) {
	// TODO(gri) remove this check in favor of specialized lookup
	if pkg == nil {
		pkg = nopkg
	}
	if s := pkg.Syms[name]; s != nil {
		return s, true
	}

	s = &Sym{
		Name: name,
		Pkg:  pkg,
	}
	if name == "init" {
		InitSyms = append(InitSyms, s)
	}
	pkg.Syms[name] = s
	return s, false
}

func (pkg *Pkg) LookupBytes(name []byte) *Sym {
	// TODO(gri) remove this check in favor of specialized lookup
	if pkg == nil {
		pkg = nopkg
	}
	if s := pkg.Syms[string(name)]; s != nil {
		return s
	}
	str := InternString(name)
	return pkg.Lookup(str)
}

var internedStrings = map[string]string{}

func InternString(b []byte) string {
	s, ok := internedStrings[string(b)] // string(b) here doesn't allocate
	if !ok {
		s = string(b)
		internedStrings[s] = s
	}
	return s
}

// CleanroomDo invokes f in an environment with with no preexisting packages.
// For testing of import/export only.
func CleanroomDo(f func()) {
	saved := pkgMap
	pkgMap = make(map[string]*Pkg)
	f()
	pkgMap = saved
}
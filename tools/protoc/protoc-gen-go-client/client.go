// Copyright (c) 2021 Terminus, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"google.golang.org/protobuf/compiler/protogen"
)

const (
	contextPackage   = protogen.GoImportPath("context")
	grpcPackage      = protogen.GoImportPath("google.golang.org/grpc")
	transgrpcPackage = protogen.GoImportPath("github.com/erda-project/erda-infra/pkg/transport/grpc")
)

func generateFiles(gen *protogen.Plugin, files []*protogen.File) error {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Desc.Name() < files[j].Desc.Name()
	})
	var count int
	var paths []string
	var file *protogen.File
	for _, f := range files {
		if len(f.Services) <= 0 {
			continue
		}
		count += len(f.Services)
		if file == nil {
			file = f
		}
		paths = append(paths, f.Desc.Path())
		if f.GoImportPath != file.GoImportPath {
			return fmt.Errorf("package path conflict between %s and %s", file.GoImportPath, f.GoImportPath)
		}
		if f.Desc.Package() != file.Desc.Package() {
			return fmt.Errorf("package path conflict between %s and %s", file.Desc.Package(), f.Desc.Package())
		}
	}
	if count <= 0 {
		return nil
	}
	sources := strings.Join(paths, ", ")

	err := genClient(gen, files, file, sources)
	if err != nil {
		return err
	}
	return genProvider(gen, files, file, sources)
}

func genClient(gen *protogen.Plugin, files []*protogen.File, root *protogen.File, sources string) error {
	const filename = "client.go"
	const pkgname = "client"
	g := gen.NewGeneratedFile(filename, pkgname)
	g.P("// Code generated by ", genName, ". DO NOT EDIT.")
	g.P("// Sources: ", sources)
	g.P()
	g.P("package ", pkgname)
	g.P()
	g.P("// Client provide all service clients.")
	g.P("type Client interface {")
	for _, file := range files {
		for _, ser := range file.Services {
			g.P("// ", ser.GoName, " ", file.Desc.Path())
			g.P(ser.GoName, "() ", file.GoImportPath.Ident(ser.GoName+"Client"))
		}
	}
	g.P("}")
	g.P()
	g.P("// New create client")
	g.P("func New(cc ", transgrpcPackage.Ident("ClientConnInterface"), ") Client {")
	g.P("	return &serviceClients{")
	for _, file := range files {
		for _, ser := range file.Services {
			g.P(lowerCaptain(ser.GoName), ": ", file.GoImportPath.Ident("New"+ser.GoName+"Client"), "(cc),")
		}
	}
	g.P("	}")
	g.P("}")
	g.P()
	g.P("type serviceClients struct {")
	for _, file := range files {
		for _, ser := range file.Services {
			g.P(lowerCaptain(ser.GoName), " ", file.GoImportPath.Ident(ser.GoName+"Client"))
		}
	}
	g.P("}")
	g.P()
	for _, file := range files {
		for _, ser := range file.Services {
			g.P("func (c *serviceClients) ", ser.GoName, "() ", file.GoImportPath.Ident(ser.GoName+"Client"), " {")
			g.P("	return c.", lowerCaptain(ser.GoName))
			g.P("}")
			g.P()
		}
	}
	g.P()
	for _, file := range files {
		for _, ser := range file.Services {
			typeName := lowerCaptain(ser.GoName) + "Wrapper"
			g.P("type " + typeName + " struct {")
			g.P("	client ", file.GoImportPath.Ident(ser.GoName+"Client"))
			g.P("	opts   []", grpcPackage.Ident("CallOption"))
			g.P("}")
			g.P()
			for _, m := range ser.Methods {
				g.P("func (s *", typeName, ") ", m.GoName, "(ctx ", contextPackage.Ident("Context"), ",req *", m.Input.GoIdent, ") (*", m.Output.GoIdent, ", error) {")
				g.P("	return s.client.", m.GoName, "(ctx, req, append(", transgrpcPackage.Ident("CallOptionFromContext"), "(ctx), s.opts...)...)")
				g.P("}")
				g.P()
			}
		}
	}
	return nil
}

func lowerCaptain(name string) string {
	if len(name) <= 0 {
		return name
	}
	chars := []rune(name)
	pre := chars[0]
	if unicode.IsLower(pre) {
		return name
	}
	for i, c := range chars {
		if unicode.IsUpper(c) != unicode.IsUpper(pre) {
			break
		}
		chars[i] = unicode.ToLower(c)
	}
	return string(chars)
}
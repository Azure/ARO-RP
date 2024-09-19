package swagger

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

type ModelAsString bool

type typeWalker struct {
	pkg            *packages.Package
	enums          map[types.Type][]interface{}
	xmsEnumList    []string
	xmsSecretList  []string
	xmsIdentifiers []string
}

func newTypeWalker(pkgname string, xmsEnumList, xmsSecretList []string, xmsIdentifiers []string) (*typeWalker, error) {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo}, pkgname)
	if err != nil {
		return nil, err
	}
	if len(pkgs) != 1 {
		return nil, fmt.Errorf("found %d packages, expected 1", len(pkgs))
	}

	tw := &typeWalker{
		pkg:            pkgs[0],
		enums:          map[types.Type][]interface{}{},
		xmsEnumList:    xmsEnumList,
		xmsSecretList:  xmsSecretList,
		xmsIdentifiers: xmsIdentifiers,
	}

	// populate enums: walk all types declared at package scope
	for _, n := range pkgs[0].Types.Scope().Names() {
		o := pkgs[0].Types.Scope().Lookup(n)
		if !o.Exported() {
			continue
		}

		if o, ok := o.(*types.Const); ok {
			switch o.Val().Kind() {
			case constant.Int:
				i, _ := constant.Int64Val(o.Val())
				tw.enums[o.Type()] = append(tw.enums[o.Type()], i)
			case constant.String:
				tw.enums[o.Type()] = append(tw.enums[o.Type()], constant.StringVal(o.Val()))
			default:
				panic(o.Val())
			}
		}
	}

	return tw, nil
}

// getNodes returns the hierarchy of ast Nodes for a given token.Pos
func (tw *typeWalker) getNodes(pos token.Pos) (path []ast.Node, exact bool) {
	// find the matching file
	for _, f := range tw.pkg.Syntax {
		if tw.pkg.Fset.File(f.Pos()) == tw.pkg.Fset.File(pos) {
			// find the matching token in the file
			return astutil.PathEnclosingInterval(f, pos, pos)
		}
	}

	return
}

// schemaFromType returns a Schema object populated from the given Type.  If
// references are made to dependent types, these are added to deps
func (tw *typeWalker) schemaFromType(t types.Type, deps map[*types.Named]struct{}) (s *Schema) {
	s = &Schema{}

	switch t := t.(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.Bool:
			s.Type = "boolean"
		case types.Int:
			s.Type = "integer"
			s.Format = "int32"
		case types.String:
			s.Type = "string"
		default:
			panic(t)
		}

	case *types.Map:
		s.Type = "object"
		s.AdditionalProperties = tw.schemaFromType(t.Elem(), deps)

	case *types.Named:
		s.Ref = "#/definitions/" + t.Obj().Name()
		deps[t] = struct{}{}

	case *types.Pointer:
		s = tw.schemaFromType(t.Elem(), deps)

	case *types.Slice:
		if e, ok := t.Elem().(*types.Basic); ok {
			// handle []byte as a string (it'll be base64 encoded by json.Marshal)
			if e.Kind() == types.Uint8 {
				s.Type = "string"
			}
		} else {
			s.Type = "array"
			s.Items = tw.schemaFromType(t.Elem(), deps)
			// https://github.com/Azure/autorest/tree/main/docs/extensions#x-ms-identifiers
			// we do not use this field, but the upstream validation requires at least an empty array
			if tw.xmsIdentifiers != nil {
				s.XMSIdentifiers = &[]string{}
			}
		}

	case *types.Struct:
		s.Type = "object"
		for i := 0; i < t.NumFields(); i++ {
			field := t.Field(i)
			if field.Exported() {
				nodes, _ := tw.getNodes(field.Pos())
				nodeField, ok := getNodeField(nodes)
				if !ok {
					panic("could not find field for nodes")
				}
				tag, _ := strconv.Unquote(nodeField.Tag.Value)

				name := strings.SplitN(reflect.StructTag(tag).Get("json"), ",", 2)[0]
				if name == "-" {
					continue
				}

				properties := tw.schemaFromType(field.Type(), deps)
				properties.Description = strings.Trim(nodeField.Doc.Text(), "\n")

				if swaggerTag, ok := reflect.StructTag(tag).Lookup("swagger"); ok {
					// XXX In theory this would be a comma-delimited
					//     list, but "readonly" is the only value we
					//     currently recognize.
					if strings.EqualFold(swaggerTag, "readOnly") {
						properties.ReadOnly = true
					}
				}

				ns := NameSchema{
					Name:   name,
					Schema: properties,
				}

				for _, xname := range tw.xmsSecretList {
					if xname == name {
						ns.Schema.XMSSecret = true
					}
				}
				s.Properties = append(s.Properties, ns)
			}
			if field.Name() == "proxyResource" {
				s.AllOf = []Schema{
					{
						Ref: "../../../../../../common-types/resource-management/v3/types.json#/definitions/ProxyResource",
					},
				}
			}
		}
	default:
		panic(t)
	}
	return
}

// _define adds a Definition for the given type and recurses on any dependencies
func (tw *typeWalker) _define(definitions Definitions, t *types.Named) {
	deps := map[*types.Named]struct{}{}

	s := tw.schemaFromType(t.Underlying(), deps)

	path, _ := tw.getNodes(t.Obj().Pos())
	if path != nil {
		s.Description = strings.Trim(path[len(path)-2].(*ast.GenDecl).Doc.Text(), "\n")
		s.Enum = tw.enums[t]
		// Enum extensions allows non-breaking api changes
		// https://github.com/Azure/autorest/tree/master/docs/extensions#x-ms-enum
		c := strings.Split(t.String(), ".")
		name := c[(len(c) - 1)]
		for _, xname := range tw.xmsEnumList {
			if xname == name {
				s.XMSEnum = &XMSEnum{
					ModelAsString: true,
					Name:          xname,
				}
			}
		}

		definitions[t.Obj().Name()] = s

		for dep := range deps {
			if _, found := definitions[dep.Obj().Name()]; !found {
				tw._define(definitions, dep)
			}
		}
	}
}

// define adds a Definition for the named type
func (tw *typeWalker) define(definitions Definitions, name string) {
	o := tw.pkg.Types.Scope().Lookup(name)
	tw._define(definitions, o.(*types.TypeName).Type().(*types.Named))
}

// define adds a Definition for the named types in the given package
func define(definitions Definitions, pkgname string, xmsEnumList, xmsSecretList []string, xmsIdentifiers []string, names ...string) error {
	th, err := newTypeWalker(pkgname, xmsEnumList, xmsSecretList, xmsIdentifiers)
	if err != nil {
		return err
	}

	for _, name := range names {
		th.define(definitions, name)
	}
	return nil
}

// Gets the field associate with the node
func getNodeField(nodes []ast.Node) (*ast.Field, bool) {
	if len(nodes) < 2 {
		return nil, false
	}
	node := nodes[1]
	if field, ok := node.(*ast.Field); ok {
		return field, ok
	}

	// Case where field is a pointer, so inspect the third element
	//
	// Example:
	//    type CloudError struct {
	//        StatusCode int `json:"-"`
	//        *CloudErrorBody `json:"error,omitempty"`
	//    }
	//
	// ast package reads the pointer as type "ast.StarExpr"

	if len(nodes) < 3 {
		return nil, false
	}
	node = nodes[2]
	if field, ok := node.(*ast.Field); ok {
		return field, ok
	}

	return nil, false
}

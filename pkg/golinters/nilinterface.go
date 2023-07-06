package golinters

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"reflect"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var NilInterfaceAnalyser = &analysis.Analyzer{
	Name:     "nilinterfacecheck",
	Doc:      "Checks for comparisons of Interfaces to nil which has numereous caveats. https://go.dev/doc/faq#nil_error",
	Run:      runNilInterfaceAnalyser,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

const NilInterfaceAlert = "nilinterfacecheck comparing nil against an interface is usually a mistake"

func isExpressionNil(ident *ast.Ident) bool {
	if ident.Obj == nil && ident.Name == "nil" {
		return true
	}

	return false
}

func runNilInterfaceAnalyser(pass *analysis.Pass) (interface{}, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.BinaryExpr)(nil),
	}

	inspector.Preorder(nodeFilter, func(node ast.Node) {
		binaryExpression := node.(*ast.BinaryExpr)

		printer.Fprint(os.Stdout, token.NewFileSet(), node)
		fmt.Println("")

		// Look for simple a == b statements
		var xIdent *ast.Ident
		var yIdent *ast.Ident
		xIsIdent := false
		yIsIdent := false

		switch binaryExpression.X.(type) {
		case *ast.Ident:
			xIdent = binaryExpression.X.(*ast.Ident)
			xIsIdent = true
		case *ast.CallExpr:
			xIdent, xIsIdent = callExprToIdent(binaryExpression.X.(*ast.CallExpr))
		}

		switch binaryExpression.Y.(type) {
		case *ast.Ident:
			yIdent = binaryExpression.Y.(*ast.Ident)
			yIsIdent = true
		case *ast.CallExpr:
			yIdent, yIsIdent = callExprToIdent(binaryExpression.Y.(*ast.CallExpr))
		}

		// If not we don't need to run the rule
		if !xIsIdent || !yIsIdent {
			fmt.Println("No run")
			return
		}

		nilIdentX := isExpressionNil(xIdent)
		nilIdentY := isExpressionNil(yIdent)

		// Make sure at least one ident is nil
		if !(nilIdentX || nilIdentY) {
			fmt.Println("No nil ident")
			return
		}

		var workingObj *ast.Ident
		if nilIdentX {
			workingObj = yIdent
		} else {
			workingObj = xIdent
		}

		// Look for a variable
		fmt.Printf("Type: %+v\n", workingObj.Obj.Kind)
		switch workingObj.Obj.Kind {
		case ast.Var:
			break
		case ast.Typ:
			break
		default:
			fmt.Println("Not a var: nothing to do")
			return

		}

		switch a := workingObj.Obj.Decl; a.(type) {
		case *ast.ValueSpec: // Clue that we're looking at a variable dec
			typeIdent, ok := handleValueSpecType(a.(*ast.ValueSpec))
			if !ok {
				// Something we dont/can't handle
				return
			}
			alertOnInterfacedIdent(node, pass, typeIdent)
		case *ast.TypeSpec:
			alertOnInterfacedTypeSpec(node, pass, a.(*ast.TypeSpec))
		default:
			fmt.Printf("Working Obj: %+v\n", reflect.TypeOf(a))
		}

		// No reported errors
		return
	})

	return nil, nil
}

func callExprToIdent(a *ast.CallExpr) (*ast.Ident, bool) {

	// First get from call expression to function object
	fnObj := a.Fun.(*ast.Ident).Obj
	fmt.Printf("Fn: %s\n", fnObj.Name)
	// Get from fnObj to FuncDecl
	fnDecl := fnObj.Decl.(*ast.FuncDecl)
	// Get from FnDecl to FuncType
	fnType := fnDecl.Type
	// Get the list of arguments
	fnReturns := fnType.Results.List
	if len(fnReturns) != 1 {
		// We don't care about fn that return multiple values
		return nil, false
	}

	return handleValueSpecType(fnReturns[0])
}

func alertOnInterfacedTypeSpec(n ast.Node, p *analysis.Pass, v *ast.TypeSpec) {
	switch v.Type.(type) {
	case *ast.InterfaceType:
		fmt.Println("REPORT")
		p.Reportf(n.Pos(), NilInterfaceAlert)
		return
	}
}

// Handle when we recieve a ValueSpec
func alertOnInterfacedIdent(n ast.Node, p *analysis.Pass, v *ast.Ident) {
	switch v.Obj.Decl.(*ast.TypeSpec).Type.(type) {
	case *ast.InterfaceType:
		fmt.Println("REPORT")
		p.Reportf(n.Pos(), NilInterfaceAlert)

	}
	return
}

// handleValueSpec type tries to handle multiple different possible places where we
// might reasonably have a variable assigned to that of an interface and return the
// appropriate ident
func handleValueSpecType(v interface{}) (*ast.Ident, bool) {
	var exp ast.Expr

	var t interface{}
	switch v.(type) {
	case *ast.Field:
		t = v.(*ast.Field).Type
	case *ast.ValueSpec:
		t = v.(*ast.ValueSpec).Type
	}

	switch t.(type) {
	case *ast.Ident:
		fmt.Printf("Init Ident: %+v\n", t.(*ast.Ident))
		return t.(*ast.Ident), true
	case *ast.StarExpr:
		exp = t.(*ast.StarExpr).X // TODO: Why does a star expression have an "X"?
	}

	switch exp.(type) {
	case *ast.Ident:
		fmt.Printf("Star Exp Ident: %+v\n", exp.(*ast.Ident))
		return exp.(*ast.Ident), true
	}

	return nil, false // It's not clear if this would ever be the case
}

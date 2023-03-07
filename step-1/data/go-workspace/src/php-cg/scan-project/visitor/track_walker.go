// Package visitor contains walker.visitor implementations
package visitor

import (
	"reflect"
	"strings"
	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/name"
	"github.com/z7zmey/php-parser/node/stmt"
	"github.com/z7zmey/php-parser/node/expr/assign"
	"github.com/z7zmey/php-parser/node/expr/binary"
	l "php-cg/scan-project/logger"
	"github.com/z7zmey/php-parser/walker"
)



// Dumper writes ast hierarchy to an io.Writer
// Also prints comments and positions attached to nodes

var Variable_dependency map[string][]string
var Tracked_variables []string

func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func processName(n node.Node) string {
	switch v:=n.(type) {
	case *expr.Variable:
		switch  v.VarName.(type) {
		case *node.Identifier:
			return v.VarName.(*node.Identifier).Value
		}
	case *node.Identifier:
		return v.Value
	case *expr.PropertyFetch:
		varname := processName(v.Variable)
		property := processName(v.Property)
		if varname == "self" || varname == "this" && property != "not found"{
			varname = var_cname
			return varname + "*" + property
		}
	case *expr.StaticPropertyFetch:
		varname, ok:= v.Class.(*name.Name)
		if ok {
			varNew, ok1 := varname.Parts[len(varname.Parts)-1].(*name.NamePart)
			if ok1 {
				lProp, ok3 := v.Property.(*expr.Variable)
				if ok3 {
					lPropVal, ok2 := lProp.VarName.(*node.Identifier)
					if ok2 {
						if varNew.Value == "self" {
							newobj := var_cname + "*" + lPropVal.Value
							return newobj
						}
					}
				}
			}
		}

	case *expr.ArrayDimFetch:
		arr, ok := v.Variable.(*expr.Variable)
		if ok {
			arrName := ""
			if arrname, ok:= arr.VarName.(*expr.Variable); ok {
				_ = arrname
				arrName = processName(arr.VarName)
			}
			if arrname, ok:= arr.VarName.(*node.Identifier); ok {
				_ = arrname
				arrName = arr.VarName.(*node.Identifier).Value
			}
                        idx, ok := v.Dim.(*expr.Variable)
			if ok && arrName != ""{
				idxname := idx.VarName.(*node.Identifier).Value
				return arrName +"*" + strings.Trim(idxname,"'")
			}
			return arrName
		}
	case *binary.Concat:
		lhs := processName(v.Left)
		rhs := processName(v.Right)
		return lhs + "*" + rhs
	}
	return "not found"

}

// EnterNode is invoked at every node in hierarchy
func (d TrackWalker) EnterNode(w walker.Walkable) bool {
	n := w.(node.Node)
	switch reflect.TypeOf(n).String() {

	case "*stmt.Class":
		class := n.(*stmt.Class)
		if namespacedName, ok := d.NsResolver.ResolvedNames[class]; ok {
			var_cname = namespacedName
		} else {
			className, ok := class.ClassName.(*node.Identifier)
			if !ok {
				l.Log(l.Error, "class name is not resolved: %s:%s", NodeSource(&n), File)
				break
			}
			var_cname = className.Value
		}
		break
	case "*stmt.Expression":
		asgnstmt, ok := n.(*stmt.Expression).Expr.(*assign.Assign)
		_ = asgnstmt
		if ok {
			lhs := processName(asgnstmt.Variable)
			rhs := processName(asgnstmt.Expression)
			if rhs != "not found" && lhs != "not found" {
				lvars := strings.Split(lhs, "*")
				rvars := strings.Split(rhs, "*")
				for _,lvar := range lvars {
					for _,rvar := range rvars {
						if !Contains(Variable_dependency[lvar], rvar) && lvar != "not found" && rvar != "not found" {
							Variable_dependency[lvar] = append(Variable_dependency[lvar], rvar)

							}
					}
				}
			}
			_, ok := asgnstmt.Expression.(*expr.New)
			if ok {
				lvars := strings.Split(lhs, "*")
				for _, lvar := range lvars {
					if lvar != "not found" {
						Tracked_variables = append(Tracked_variables, lvar)
					}
				}
				Tracked_variables = append(Tracked_variables,lhs)

			}
		}
	case "*expr.StaticCall":
		scall := n.(*expr.StaticCall)
		scls := processName(scall.Class)
		sName := processName(scall.Call)
		if scls != "this" && scls != "self" && scls != "not found"{
			Tracked_variables = append(Tracked_variables, scls)
		}
		if sName != "not found" {
			Tracked_variables = append(Tracked_variables, sName)
		}

	case "*expr.MethodCall":
		method := n.(*expr.MethodCall)
		mcls := processName(method.Variable)
		mName := processName(method.Method)
		if mcls != "this" && mcls != "self" && mcls != "not found"{
			if !Contains(Tracked_variables, mcls) {
				Tracked_variables = append(Tracked_variables, mcls)
			}
		}
		if mName != "not found" {
			if !Contains(Tracked_variables, mName) {
				Tracked_variables = append(Tracked_variables, mName)
			}
		}
	case "*expr.FunctionCall":
		function := n.(*expr.FunctionCall)
		fname1, ok := function.Function.(*name.Name)
		fname2, ok1 := function.Function.(*name.FullyQualified)
		if !ok && !ok1 {
			fName := processName(function.Function)
			if !Contains(Tracked_variables, fName) {
			Tracked_variables = append(Tracked_variables, fName)
			}
		} else if ok {
			fname := fname1.Parts[len(fname1.Parts)-1].(*name.NamePart).Value
			if strings.Contains(fname, "call_user_func") {
				if len(function.ArgumentList.Arguments) >= 1 {
					arg1 := processName(function.ArgumentList.Arguments[0].(*node.Argument).Expr)
					if arg1 != "not found" {
						if !Contains(Tracked_variables, arg1) {
							Tracked_variables = append(Tracked_variables, arg1)
						}
					}
				}
			}
		} else if ok1 {
			fname := fname2.Parts[len(fname2.Parts)-1].(*name.NamePart).Value
			if strings.Contains(fname, "call_user_func") {
				if len(function.ArgumentList.Arguments) >= 1 {
					arg1 := processName(function.ArgumentList.Arguments[0].(*node.Argument).Expr)
					if arg1 != "not found" {
						if !Contains(Tracked_variables, arg1) {
							Tracked_variables = append(Tracked_variables, arg1)
						}
					}
				}
			}

		}
		break
	}
	return true
}


// GetChildrenVisitor is invoked at every node parameter that contains children nodes
func (d TrackWalker) GetChildrenVisitor(key string) walker.Visitor {
	return TrackWalker{d.Writer, d.Indent + "    ", d.Comments, d.Positions, d.NsResolver }
}

// LeaveNode is invoked after node process
func (d TrackWalker) LeaveNode(w walker.Walkable) {
	//parse := false
	n := w.(node.Node)

	switch reflect.TypeOf(n).String() {
	case "*stmt.Class":
		var_cname = ""
		break

	}
}

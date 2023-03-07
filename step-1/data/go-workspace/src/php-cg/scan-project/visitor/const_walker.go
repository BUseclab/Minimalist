package visitor

import (
	l "php-cg/scan-project/logger"
	"reflect"

	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/expr/assign"
	"github.com/z7zmey/php-parser/node/name"
	"github.com/z7zmey/php-parser/node/stmt"
	"github.com/z7zmey/php-parser/walker"
)

func (d ConstWalker) EnterNode(w walker.Walkable) bool {

	n := w.(node.Node)

	switch reflect.TypeOf(n).String() {
	case "*expr.FunctionCall":
		// Record
		function := n.(*expr.FunctionCall)
		functionName, ok := function.Function.(*name.Name)

		if ok {
			if functionName.Parts[0].(*name.NamePart).Value == `define` {
				l.Log(l.Debug, "New Definition %s", NodeSource(&n))
				name := function.ArgumentList.Arguments[0].(*node.Argument)
				if len(function.ArgumentList.Arguments) >= 2 {
					value := function.ArgumentList.Arguments[1].(*node.Argument)
					pn := processStringExpr(name.Expr)
					pv := processStringExpr(value.Expr)
					pn.Consolidate()
					if _, ok := (*Constants)[pn.Content]; !ok {
						node := st{}
						node.AddLeaf(pv)
						node.Consolidate()
						(*Constants)[pn.Content] = node
						l.Log(l.Warning, "%s = %+v", pn.Content, (*Constants)[pn.Content])
					} else {
					}
					break
				}
			}
		} else {
			l.Log(l.Warning, "FunctionCall not handled: %s", NodeSource(&n))

		}
	case "*stmt.Expression":
		n := n.(*stmt.Expression)
		if s, ok := n.Expr.(*assign.Assign); ok {
			if varname, ok := s.Variable.(*expr.Variable); ok {
				if varname, ok := varname.VarName.(*node.Identifier); ok {
					assign := processStringExpr(s.Expression)
					assign.Consolidate()
					mutex.Lock()
					Assigns[varname.Value] = assign
					mutex.Unlock()
				}
			}
		} else if s, ok := n.Expr.(*assign.Concat); ok {
			if varname, ok := s.Variable.(*expr.Variable); ok {
				if varname, ok := varname.VarName.(*node.Identifier); ok {
					mutex.Lock()
					vn := Assigns[varname.Value]
					mutex.Unlock()
					result := processStringExpr(s.Expression)
					vn.AddLeaf(result)
					mutex.Lock()
					Assigns[varname.Value] = vn
					mutex.Unlock()
				}
			}

		}

	}
	return true

}

// GetChildrenVisitor is invoked at every node parameter that contains children nodes
func (d ConstWalker) GetChildrenVisitor(key string) walker.Visitor {
	return ConstWalker{d.Writer, d.Indent + "    ", d.Comments, d.Positions}
}

// LeaveNode is invoked after node process
func (d ConstWalker) LeaveNode(w walker.Walkable) {

}

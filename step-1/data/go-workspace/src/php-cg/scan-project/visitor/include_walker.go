// Package visitor contains walker.visitor implementations
package visitor

import (
	"fmt"
	"bytes"
	"reflect"
	"strings"
	"path"
	"path/filepath"
	"strconv"
	"github.com/adam-hanna/copyRecursive"
	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/name"
	"github.com/z7zmey/php-parser/node/stmt"
	"github.com/z7zmey/php-parser/node/scalar"
	"github.com/z7zmey/php-parser/node/expr/binary"
	"github.com/z7zmey/php-parser/printer"
	l "php-cg/scan-project/logger"
	"php-cg/scan-project/visitor/include_string"
	"github.com/z7zmey/php-parser/walker"
)

const Alpha = "abcdefghijklmnopqrstuvwxyz"

var Exists = struct{}{}

// Dumper writes ast hierarchy to an io.Writer
// Also prints comments and positions attached to nodes



var FunctionCalls map[string]bool
var StaticIncludes map[string]bool
var SemiDynamicIncludes map[string]bool
var DynamicIncludes map[string]bool
var ClassInstances map[string]bool
var ClassDefinitions map[string]bool
var Assigns map[string]st

var Includes map[string][]string

var Constants *map[string]st

var includePath []string
var inInclude bool = false
var indexingIntoArray bool = false

type st = includestring.StringTrie

var containsDynamicIncludeItem bool = false
var containsStaticIncludeItem bool = false

var NumIncludes int = 0

func ClearAssigns() {
	Assigns = make(map[string]st)
}

func NodeSource(n *node.Node) string {
	out := new(bytes.Buffer)
	p := printer.NewPrinter(out, "    ")
	p.Print(*n)
	return strings.Replace(out.String(), "\n", "\\ ", -1)
}

func processMagicConstant(n *scalar.MagicConstant) string {
	switch n.Value {
	case "__FILE__":
		return File
	case "__DIR__":
		return path.Dir(File)
	default:
		l.Log(l.Error, "Unhandled MagicConstant: %s", n.Value)

	}
	return ""
}

func resolveInclude(n node.Node) []string {
	path := st{}
	switch v := n.(type) {
		case *expr.Include:
			path = processStringExpr(v.Expr)
		case *expr.IncludeOnce:
			path = processStringExpr(v.Expr)
		case *expr.Require:
			path = processStringExpr(v.Expr)
		case *expr.RequireOnce:
			path = processStringExpr(v.Expr)
	}
	l.Log(l.Info, "%s resolved to %+v", NodeSource(&n), path)
	if path.IsSimpleString(Constants) {
		//l.Log(l.Info, "%s resolved to %s", nodeSource(&n), path.SimpleString())
		resolution := path.SimpleString()
		pattern := ".*" + resolution + ".*"
		if _, ok := Includes[pattern]; !ok {
			Includes[pattern] = []string{}
		}
		l.Log(l.Info, "%s patterened to SS %s", NodeSource(&n), pattern)
		Includes[pattern] = append(Includes[pattern], NodeSource(&n))
	} else {
		pattern := ".*("
		possibilities := path.DfsPaths()
		first := true
		for _, p := range possibilities {
			if !first {
				pattern += "|"
			}
			first = false
			trielink := p
			for skip := false; !skip; {
				if trielink.Dynamic || trielink.Constant != "" {
					pattern += ".*"
				} else {
					pattern += trielink.Content
				}
				if len(trielink.Children) == 1 {
					trielink = trielink.Children[0]
				} else {
					skip = true
				}
			}
		}
		pattern += ").*"
		if _, ok := Includes[pattern]; !ok {
			Includes[pattern] = []string{}
		}
		Includes[pattern] = append(Includes[pattern], NodeSource(&n))
		l.Log(l.Info, "%s patterened to DS %s", NodeSource(&n), pattern)
		l.Log(l.Info, "%s resolved to %s", NodeSource(&n), pattern)
	}
	return []string{path.Content}
}


func processFunctionCall(n *expr.FunctionCall) st {
	result :=st{Dynamic: true}
	if _, ok := n.Function.(*name.Name); !ok {
		l.Log(l.Debug, "Unhandled FunctionCall")
		return result

	}
	switch n.Function.(*name.Name).Parts[0].(*name.NamePart).Value {

	case "dirname":
		result := st{}
		dirs := processStringExpr(n.ArgumentList.Arguments[0].(*node.Argument).Expr)
		dirs.Consolidate()
		iters := 1
		if len(n.ArgumentList.Arguments) > 1 {
			striters := processStringExpr(n.ArgumentList.Arguments[1].(*node.Argument).Expr)
			striters.Consolidate()
			iters, _ = strconv.Atoi(striters.Content)

		}
		for _, pth := range dirs.DfsPaths() {
			pth.Consolidate()
			if len(pth.Children) == 0 {
				p := pth.Content
				for i := iters; i > 0; i-- {
					if len(p) > 1 {
						if p[len(p)-1] == '/' {
							p = p[:len(p)-1]
						}
					}
					p = path.Dir(p)
				}
				result.AddChild(st{Content: p})
			} else{
				l.Log(l.Info, "Not good...")
			}
		}
		return result
	case "realpath":
		result := st{}
		components := processStringExpr(n.ArgumentList.Arguments[0].(*node.Argument).Expr)
		for _, p := range components.DfsPaths() {
			if p.IsSimpleString() {
				fp, _ := filepath.Abs(p.Content)
				result.AddChild(st{Content: fp})

			}

		}
	default:
		// l.Log(l.Existsrror, "Unhandled Function: %s", n.Function.(*name.Name).Parts[0].(*name.NamePart).Value)

	}
	return st{Dynamic: true}

}


func processStringExpr(n node.Node) st {

	switch v := n.(type) {

		case *scalar.String:
			result := st{}
			s := v.Value
			if len(s) > 0 && (s[0] == '"' || s[0] == '\'') {
				s = s[1:]
				}
			if len(s) > 0 && (s[len(s)-1] == '"' || s[len(s)-1] == '\'') {
				s = s[:len(s)-1]
			}
			if strings.HasPrefix(s, "./"){
				s = s[2:]
			}
			result.Content = s
			return result
		case *binary.Concat:
			result := processStringExpr(v.Left)
			result2 := processStringExpr(v.Right)
			// l.Log(l.Debug, "%+v", result2)
			// l.Log(l.Debug, "%+v", result)
			result2.Consolidate()
			result.AddLeaf(result2)
			result.Consolidate()
			return result
		case *scalar.Encapsed:
			result := st{}
			for _, part := range v.Parts {
				if p, ok := part.(*scalar.EncapsedStringPart); ok {
					result.AddLeaf(st{Content: p.Value})
				} else {
					result.AddLeaf(processStringExpr(part))
				}
			}
			result.Consolidate()
			return result
		case *expr.ConstFetch:
			if _, ok := v.Constant.(*name.Name); ok {
				constIdentifier := v.Constant.(*name.Name).Parts[0].(*name.NamePart).Value
				result := st{Constant: constIdentifier}
				return result
			}
			if _, ok := v.Constant.(*name.FullyQualified); ok {
				constIdentifier := v.Constant.(*name.FullyQualified).Parts[0].(*name.NamePart).Value
				result := st{Constant: constIdentifier}
				return result
			}
		case *scalar.MagicConstant:
			return st{Content: processMagicConstant(v)}
		case *scalar.Lnumber:
			return st{Content: v.Value}
		case *expr.FunctionCall:
			return processFunctionCall(v)
		case *expr.Variable:
			if value, ok := v.VarName.(*node.Identifier); ok {
				if val, ok := Assigns[value.Value]; ok {
					return recursiveCopy.Copy(val).(st)
				}
			} else {
				l.Log(l.Warning, "Could not parse variable")
			}
		}
	return st{Dynamic: true}
}


// EnterNode is invoked at every node in hierarchy
func (d IncludeWalker) EnterNode(w walker.Walkable) bool {

	n := w.(node.Node)
	switch reflect.TypeOf(n).String() {

	case "*expr.Include", "*expr.IncludeOnce", "*expr.Require", "*expr.RequireOnce":
		inInclude = true
		resolveInclude(n)
		NumIncludes += 1
		includePath = nil
		break

	case "*stmt.Class":
		class := n.(*stmt.Class)
		if namespacedName, ok := d.NsResolver.ResolvedNames[class]; ok && NsEnable{
			ClassDefinitions[namespacedName] = true
		} else {
			className , ok := class.ClassName.(*node.Identifier)
			if !ok {
				l.Log(l.Error, "couldn't resolve classname:%s", NodeSource(&n))
				break
			}
			ClassDefinitions[className.Value] = true
		}
		extClass := class.Extends
		if extClass != nil {
			if namespacedName, ok  := d.NsResolver.ResolvedNames[extClass]; ok && NsEnable {
				ClassInstances[namespacedName] = true
			} else {
				extName, ok := extClass.ClassName.(*name.Name)
				if !ok {
					l.Log(l.Error, "extend class name is not resolved %s:%s", NodeSource(&n), File)
					break
				}
				eName = extName.Parts[len(extName.Parts)-1].(*name.NamePart).Value
				ClassInstances[eName] = true
			}
		}
		break


	case "*expr.FunctionCall":
		// Record
		function := n.(*expr.FunctionCall)
		functionname, ok := function.Function.(*name.Name)
		functionQname, ok2 := function.Function.(*name.FullyQualified)
		if ok {
			if namespacedName, ok := d.NsResolver.ResolvedNames[functionname]; ok && NsEnable {
				FunctionCalls[namespacedName] = true
			} else {
				parts := functionname.Parts
				lastNamePart, ok := parts[len(parts)-1].(*name.NamePart)
				if ok {
					FunctionCalls[lastNamePart.Value] = true

				}
			}
		} else if ok2{
			if namespacedName, ok := d.NsResolver.ResolvedNames[functionQname]; ok && NsEnable {
				FunctionCalls[namespacedName] = true
			} else {
				parts := functionQname.Parts
				lastNamePart, ok := parts[len(parts)-1].(*name.NamePart)
				if ok {
					FunctionCalls[lastNamePart.Value] = true
				}
			}
		} else {
				l.Log(l.Error, "Function call is not resolved: %s:%s", NodeSource(&n), File)
		}
		break

	case "*expr.StaticCall":

		Numcalls = Numcalls + 1

		lastClassPart := ""
		variableClass := false 
		class := n.(*expr.StaticCall).Class
		call := n.(*expr.StaticCall).Call
		if _, ok := class.(*expr.Variable); ok {
			variableClass = true
			lastClassPart = "$variable"
		}
		className, ok1 := class.(*name.Name)
		fqnClassName, ok2 := class.(*name.FullyQualified)
		classIdentifier, ok3 := class.(*node.Identifier)
		if !(ok1 || ok2 || ok3 || variableClass) {
			l.Log(l.Error,"Static call is not resolved %s:%s", NodeSource(&n), File)
			break
		}
		callName, ok := call.(*node.Identifier)
		if !ok {

			l.Log(l.Info,"%s",NodeSource(&n))
			break
		}
		if ok1 {
			if namespacedName, ok := d.NsResolver.ResolvedNames[className]; ok {
//				fmt.Printf("Came here for %s file (%s)\n", namespacedName, File)
				lastClassPart = namespacedName
			} else {
				lastClassPart = className.Parts[len(className.Parts)-1].(*name.NamePart).Value
			}
		} else if ok2 {
			if namespacedName, ok := d.NsResolver.ResolvedNames[fqnClassName]; ok {
				lastClassPart = namespacedName
			} else {
				lastClassPart = fqnClassName.Parts[len(fqnClassName.Parts)-1].(*name.NamePart).Value
			}
		} else if ok3 {
			lastClassPart = classIdentifier.Value
		}
		FunctionCalls[lastClassPart + "::" + callName.Value ] = true

		break

	case "*expr.New":

		// should handle calls to __construct
		class := n.(*expr.New).Class
		className, ok := class.(*name.Name)
		if ok {
			if namespacedName, ok := d.NsResolver.ResolvedNames[className]; ok {
				ClassInstances[namespacedName] = true
			} else {
				lastClassPart, ok := className.Parts[len(className.Parts)-1].(*name.NamePart)
				if !ok {
					l.Log(l.Warning, "expr.New not handled: %s", NodeSource(&n))
					break
				}
				ClassInstances[lastClassPart.Value] = true
			}
		} else if className, ok := class.(*name.FullyQualified); ok {
			if namespacedName, ok := d.NsResolver.ResolvedNames[className]; ok {
				ClassInstances[namespacedName] = true
			} else {
				lastClassPart := className.Parts[len(className.Parts)-1].(*name.NamePart)
				fmt.Printf("here a fullyqualified class for new expression %s", lastClassPart)
				ClassInstances[lastClassPart.Value] = true
			}
		} else {
			l.Log(l.Warning, "expr.New is not resovled: %s", NodeSource(&n))
			break
		}
	}
	return true
}


// GetChildrenVisitor is invoked at every node parameter that contains children nodes
func (d IncludeWalker) GetChildrenVisitor(key string) walker.Visitor {
	return IncludeWalker{d.Writer, d.Indent + "    ", d.Comments, d.Positions, d.NsResolver}
}

// LeaveNode is invoked after node process
func (d IncludeWalker) LeaveNode(w walker.Walkable) {
	//parse := false
	n := w.(node.Node)

	switch reflect.TypeOf(n).String() {
	case "*stmt.Class":
		CName = ""
		eName = ""
	case "*stmt.Function":
		infunc = false
		functionName = "main"

	case "*stmt.ClassMethod":
		inmeth = false
		methodName = "main"

	}
}

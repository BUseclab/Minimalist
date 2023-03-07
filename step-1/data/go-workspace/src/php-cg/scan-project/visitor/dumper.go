// Package visitor contains walker.visitor implementations
package visitor

import (
	"sync"
	"reflect"
	"strings"
	"strconv"
	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/name"
	"github.com/z7zmey/php-parser/node/stmt"
	l "php-cg/scan-project/logger"
	"github.com/z7zmey/php-parser/walker"
)


// Dumper writes ast hierarchy to an io.Writer
// Also prints comments and positions attached to nodes

var mutex = &sync.Mutex{}
var varmutex = &sync.Mutex{}


var File string
var RelativePath string
var ScriptDep map[string][]string
var CG = make(map[string]string)
var MethodCalls map[string]bool
var UnresCalls = make(map[string]string)
var Functions = make(map[string][]int)
var Methods = make(map[string][]int)

var Funcfilemap = make(map[string]string)
var Methfilemap = make(map[string]string)

var Classes = make(map[string][]string)

var infunc bool
var inmeth bool
var Numcalls = 0
var NsEnable = true
/// handle inheritance in method/function invokes
var Extends = make(map[string]string) // hold a mapping between parent class and sub-class
var Children = make (map[string]map[string]bool) // hold a reverse mapping between sub-class and parent class

var IntExtends = make(map[string]map[string]bool) // hold a mapping between parent interface and sub-interfaces

var usedTraits [] string // hold used traits in a class
var className string // to handle $this variable
var VarAssigns map[string]string // mapping between rhs and lhs in assignment statements

var metfunc int
var rtn int
var Curcls = -1
/////// hashmap for WP
var Actions = make(map[string]map[string]bool)

var methodName string = "main" // hold the name of method
var functionName string = "main" // hold the name of function
var CName string = "" // hold the name of class
var eName string = "" // hold the name of extended class
var tName string = "" // hold the name of trait
var cg_incls = 0


// stat variables
// function call
var Total_func_call = 0
var Dyn_func_call = 0
var Static_func_call = 0
// includes
var Total_inc = 0
var Dyn_inc = 0
var Static_inc = 0
// object init -> autoload
var Total_ins = 0
var Dyn_ins = 0
var Static_ins = 0

// var_functionality
var CallGCString = 0

// EnterNode is invoked at every node in hierarchy
func (d Dumper) EnterNode(w walker.Walkable) bool {

	n := w.(node.Node)
	fname := RelativePath

	switch reflect.TypeOf(n).String() {

	case "*stmt.Interface":
		intface := n.(*stmt.Interface)
		if namespacedName, ok := d.NsResolver.ResolvedNames[intface]; ok && NsEnable{
			CName = namespacedName
		} else {
			intName , ok := intface.InterfaceName.(*node.Identifier)
			if !ok {
				l.Log(l.Error, "couldn't resolve interface:%s", NodeSource(&n))
				break
			}
			CName = intName.Value

		}
		break
	case "*stmt.Class":
		Curcls += 1
		class := n.(*stmt.Class)
		ClearUsedTraits()
		cg_incls = 1
		if namespacedName, ok := d.NsResolver.ResolvedNames[class]; ok && NsEnable{
			CName = namespacedName
		} else {
			className , ok := class.ClassName.(*node.Identifier)
			if !ok {
				l.Log(l.Error, "couldn't resolve classname:%s", NodeSource(&n))
				break
			}
			CName = className.Value
		}
		extClass := class.Extends
		if extClass != nil {
			if namespacedName, ok  := d.NsResolver.ResolvedNames[extClass.ClassName]; ok && NsEnable {
				eName = namespacedName
			} else {
				extName, ok := extClass.ClassName.(*name.Name)
				if !ok {
					l.Log(l.Error, "extend class name is not resolved %s:%s", NodeSource(&n), File)
					break
				}
				eName = extName.Parts[len(extName.Parts)-1].(*name.NamePart).Value
			}
		}
		break
	case "*stmt.Trait":
		trait := n.(*stmt.Trait)
		if namespacedName, ok := d.NsResolver.ResolvedNames[trait]; ok {
			tName = namespacedName
		} else {
			traitName, ok := trait.TraitName. (*node.Identifier)
			if !ok {
				l.Log(l.Error, "Traitname is not resolved %s:%s", NodeSource(&n), File)
				break
			}
			tName = traitName.Value
		}
		break
	case "*stmt.TraitUse":
		uTraitstmt := n.(*stmt.TraitUse)
		for _, trait := range uTraitstmt.Traits {
			if namespacedName, ok := d.NsResolver.ResolvedNames[trait]; ok {
				usedTraits = append(usedTraits, namespacedName)
			} else {
				t := trait.(*name.Name)
				uTrait := t.Parts[len(t.Parts)-1].(*name.NamePart).Value
				usedTraits = append(usedTraits, uTrait)
			}
		}
		break
	case "*stmt.Function":
		function := n.(*stmt.Function)
		funcName , ok := function.FunctionName.(*node.Identifier)
		infunc = true
		if ok {
			if namespacedName, ok := d.NsResolver.ResolvedNames[function]; ok && NsEnable{
				functionName = namespacedName

				_ = functionName
			} else {
			functionName = funcName.Value
			_ = functionName

		}
		} else {
				l.Log(l.Error, "Function Name not handled: %s:%s", File, NodeSource(&n))
		}
		ClearArgsType()
		// try to find out the type of arguments that should be passed to this function
		for idx := 0 ; idx < len(function.Params); idx++ {
			if vt := function.Params[idx].(*node.Parameter).VariableType; vt != nil {
			// if there is value type in the AST of arguments while defining the method
				if namespacedName, ok := d.NsResolver.ResolvedNames[vt]; ok {
					// should find the name of the argument now
					vr, ok := function.Params[idx].(*node.Parameter).Variable.(*expr.Variable)
					if ok {
						vr, ok := vr.VarName.(*node.Identifier)
						if ok {
							vr_name := vr.Value
							ArgsType[vr_name] = namespacedName
						}
					}
				}
			}
		}

		break
	case "*stmt.ClassMethod":
		classMethod := n.(*stmt.ClassMethod)
		mName , ok := classMethod.MethodName.(*node.Identifier)
		inmeth = true
		if ok {
			if namespacedName, ok := d.NsResolver.ResolvedNames[classMethod.MethodName]; ok && NsEnable {
				methodName = namespacedName
			} else {
				if cg_incls == 1 {
					methodName = CName  + "\\" + mName.Value
				} else {
					methodName = tName  + "\\" + mName.Value
				}
		}
		} else {
				l.Log(l.Error, "Method Name not handled: %s:%s", File,NodeSource(&n))
		}
		ClearArgsType()
		for idx := 0 ; idx < len(classMethod.Params); idx++ {
			if vt := classMethod.Params[idx].(*node.Parameter).VariableType; vt != nil {
				l.Log(l.Info, "recording the argument type passed to function %s:%s", var_mname, File)
				if namespacedName, ok := d.NsResolver.ResolvedNames[vt]; ok {
					l.Log(l.Info, "the argtype is %s",namespacedName)
					vr, ok := classMethod.Params[idx].(*node.Parameter).Variable.(*expr.Variable)
					if ok {
						vr, ok := vr.VarName.(*node.Identifier)
						if ok {
							vr_name := vr.Value
							l.Log(l.Info, "trying to see if it is a interface")
							first := true
							if _ , exist := ImplementedIntface[namespacedName]; exist {
								l.Log(l.Info, "It is an interface")
								for _, item := range ImplementedIntface[namespacedName] {
									l.Log(l.Info, "The Argtype can be %s", item)
									if first {
										ArgsType[vr_name] = item
										first = false
									} else {
										ArgsType[vr_name] += "|" + item
									}
									if _, exist := Extends[item]; exist {
										ArgsType[vr_name] += "|" + Extends[item]
									}
								}
							} else {
									ArgsType[vr_name] = namespacedName
							}
						}
					}
				}
			}
		}
		break

	case "*expr.MethodCall":
		l.Log(l.Info, "the method call is %s", NodeSource(&n))
		Total_func_call += 1
		method := n.(*expr.MethodCall)
		num_args := len(method.ArgumentList.Arguments)
		methodname := ""
		clsName := ""
		var dynamic= 0
		switch vr := method.Variable.(type) {
			case *expr.Variable:
				cls , ok2 := vr.VarName.(*node.Identifier)
				if ok2 && (cls.Value == "this" || cls.Value == "parent"){
					clsName = CName
					cnametmp := CName
					
					_, exist := Children[cnametmp]
					if exist {
						children := Children[cnametmp]
						for val, _ := range children {
							clsName = clsName + "|" + val
						}
					}
					
					_, exist = Extends[cnametmp]
					if exist {
						for true  {
							clsName = clsName + "|" + Extends[cnametmp]
							cnametmp = Extends[cnametmp]
							_, exist := Extends[cnametmp]
							if !exist {
								break
							}
						}
					}
					if cls.Value == "this" {
						for _, t := range usedTraits {
							clsName += t
						}
					}
				} else {
					dynamic = 1
					cnametmp := d.GCProcessStringExpr(File, vr, Curcls)
					clsName = cnametmp
					_, exist := Extends[cnametmp]
					if exist {
						for true  {
							clsName = clsName + "|" + Extends[cnametmp]
							cnametmp = Extends[cnametmp]
							_, exist := Extends[cnametmp]
							if !exist {
								break
							}
						}
					}

				}
				break
			case *expr.PropertyFetch:
				dynamic = 1
				clsName = d.GCProcessStringExpr(File, vr, Curcls)
				clsName = strings.ReplaceAll(clsName, "#","")
				break
			case *expr.FunctionCall:
				dynamic = 1
				clsName = d.GCProcessStringExpr(File, vr, Curcls)
				break
			case *expr.StaticPropertyFetch:
				dynamic = 1
				clsName = d.GCProcessStringExpr(File, vr, Curcls)
				break
			default:
				dynamic = 1
				clsName = d.GCProcessStringExpr(File, vr, Curcls)
				break
			}

		switch method.Method.(type) {
		case *node.Identifier:
			methodCall := method.Method.(*node.Identifier)
			if namespacedName, ok := d.NsResolver.ResolvedNames[methodCall]; ok && NsEnable{
				methodname = namespacedName
				MethodCalls[namespacedName] = true
			} else {
				methodname = methodCall.Value
				MethodCalls[methodname] = true
			}
			break
		case *expr.Variable:
			dynamic = 1
			methodname = d.GCProcessStringExpr(File, method.Method, Curcls)
			break
		default:
			dynamic = 1
			methodname = d.GCProcessStringExpr(File, method.Method, Curcls)
			break


		}
		if dynamic == 1 {
			Dyn_func_call += 1
		} else {
			Static_func_call += 1
		}
		if contain_dotStar("("+clsName + ")\\(" + methodname + ")") {
			if infunc == true {
				UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = functionName+ "|" + fname
			} else if inmeth == true {
				UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = methodName+ "|" + fname
			} else{
				UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = "main" + "|" + fname
			}
			l.Log(l.Critical, "%s (%s-->%s)","("+clsName + ")\\(" + methodname + ")", fname, d.Positions[n])
		}
		if infunc == true {
			CG[functionName+"|"+fname] = CG[functionName+"|"+fname]+"#(" + clsName + ")\\(" + methodname + ")#"+ strconv.Itoa(num_args)
			l.Log(l.Info, "(%s\\%s) -> %s|%s", clsName, methodname,functionName, fname)
		} else if inmeth == true {
			CG[methodName+"|"+fname] = CG[methodName+"|"+fname]+"#("+ clsName + ")\\(" + methodname +")#" + strconv.Itoa(num_args)
			l.Log(l.Info, "(%s\\%s) -> %s|%s", clsName, methodname, methodName, fname)
		} else{
			CG["main"+"|"+fname] = CG["main"+"|"+fname] + "#(" + clsName + ")\\(" +methodname + ")#"+ strconv.Itoa(num_args)
			l.Log(l.Info, "(%s\\%s) -> main|%s", clsName, methodname, fname)
		}
		break

	case "*expr.FunctionCall":
		// Record
		Total_func_call += 1
		function := n.(*expr.FunctionCall)
		functionname, ok := function.Function.(*name.Name)
		functionQname, ok2 := function.Function.(*name.FullyQualified)
		num_args := len(function.ArgumentList.Arguments)
		if ok {
			if namespacedName, ok := d.NsResolver.ResolvedNames[functionname]; ok && NsEnable {
				if containsCallback(n) {
					d.handle_callbacks(function, File, fname, Curcls)
				}
				if infunc == true{
					CG[functionName + "|" + fname] = CG[functionName + "|" + fname] + "#"+namespacedName + "#" + strconv.Itoa(num_args)

				} else if inmeth == true {
					CG[methodName + "|"+fname] = CG[methodName + "|"+fname] + "#"+namespacedName + "#" + strconv.Itoa(num_args)

				} else {
					CG["main|" + fname] = CG["main|" + fname] + "#"+namespacedName +"#" + strconv.Itoa(num_args)
				}
				Static_func_call += 1
			} else {
				parts := functionname.Parts
				lastNamePart, ok := parts[len(parts)-1].(*name.NamePart)
				if ok {
					if containsCallback(n) {
						d.handle_callbacks(function, File,fname, Curcls)
					}
					if contain_dotStar(lastNamePart.Value) {
						if infunc == true {
							UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = functionName+ "|" + fname
						} else if inmeth == true {
							UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = methodName+ "|" + fname
						} else{
							UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = "main" + "|" + fname
						}
						l.Log(l.Critical, "%s (%s-->%s)",lastNamePart.Value, fname, d.Positions[n])
					}
					if infunc == true{
						CG[functionName+"|"+fname] = CG[functionName+"|"+fname] + "#"+lastNamePart.Value +"#" +strconv.Itoa(num_args)

					} else if inmeth == true {
						CG[methodName+"|"+fname] = CG[methodName+"|"+fname] + "#"+lastNamePart.Value + "#" +strconv.Itoa(num_args)

					} else {
						CG["main"+"|"+fname] = CG["main"+"|"+fname] + "#"+lastNamePart.Value + "#" +strconv.Itoa(num_args)

					}

					Static_func_call += 1
				}
			}
		} else if ok2{
			if namespacedName, ok := d.NsResolver.ResolvedNames[functionQname]; ok && NsEnable {
				if infunc == true{
					CG[functionName + "|" + fname] = CG[functionName + "|" + fname] + "#"+namespacedName + "#" +strconv.Itoa(num_args)

				} else if inmeth == true {
					CG[methodName + "|"+fname] = CG[methodName + "|"+fname] + "#"+namespacedName + "#" +strconv.Itoa(num_args)

				} else {
					CG["main|" + fname] = CG["main|" + fname] + "#"+namespacedName + "#" +strconv.Itoa(num_args)
				}
				Static_func_call += 1
			} else {
				parts := functionQname.Parts
				lastNamePart, ok := parts[len(parts)-1].(*name.NamePart)
				if ok {
					if contain_dotStar(lastNamePart.Value) {
						if infunc == true {
							UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = functionName+ "|" + fname
						} else if inmeth == true {
							UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = methodName+ "|" + fname
						} else{
							UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = "main" + "|" + fname
						}
						l.Log(l.Critical, "%s(%s-->%s)", lastNamePart.Value, fname, d.Positions[n])
					}
					if infunc == true{
						CG[functionName+"|"+fname] = CG[functionName+"|"+fname] + "#"+lastNamePart.Value + "#" + strconv.Itoa(num_args)

					} else if inmeth == true {
						CG[methodName+"|"+fname] = CG[methodName+"|"+fname] + "#"+lastNamePart.Value + "#" + strconv.Itoa(num_args)

					} else {
						CG["main"+"|"+fname] = CG["main"+"|"+fname] + "#"+lastNamePart.Value + "#" + strconv.Itoa(num_args)
					}
					Static_func_call += 1
				}
			}
		} else {
			Dyn_func_call += 1

			fName := d.GCProcessStringExpr(File, function.Function, Curcls)
			l.Log(l.Info, "the returned value is for function call is %s: %s", fName, NodeSource(&n))
			if fName == "CLOSURE" {
				l.Log(l.Info, "Function call is not resolved: %s:%s", NodeSource(&n), File)
				break
			}
			if fName == "" {
				l.Log(l.Error, "Function call is not resolved: %s:%s", NodeSource(&n), File)
				break
			} else {
					if contain_dotStar(fName) {
						if infunc == true {
							UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = functionName+ "|" + fname
						} else if inmeth == true {
							UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = methodName+ "|" + fname
						} else{
							UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = "main" + "|" + fname
						}
						l.Log(l.Critical, "%s (%s-->%s)", fName, fname, d.Positions[n])
					}
					if infunc == true{
						CG[functionName+"|"+fname] = CG[functionName+"|"+fname] + "#"+fName + "#" + strconv.Itoa(num_args)

					} else if inmeth == true {
						CG[methodName+"|"+fname] = CG[methodName+"|"+fname] + "#"+fName + "#" + strconv.Itoa(num_args)

					} else {
						CG["main"+"|"+fname] = CG["main"+"|"+fname] + "#"+ fName + "#" + strconv.Itoa(num_args)
					}
			}
		}
		break

	case "*expr.StaticCall":

		Total_func_call += 1
		l.Log(l.Info, "the static call is %s", NodeSource(&n))
		lastClassPart := ""
		callName := ""
		class := n.(*expr.StaticCall).Class
		call := n.(*expr.StaticCall).Call
		num_args := len(n.(*expr.StaticCall).ArgumentList.Arguments)
		var dynamic = 0
		switch class.(type) {
		case *name.Name:
			classname := class.(*name.Name)
			if namespacedName, ok := d.NsResolver.ResolvedNames[classname]; ok {
				lastClassPart = namespacedName
			} else {
				l.Log(l.Error,"static call's class is not resovled %s", File)
			}
			if lastClassPart == "self" {
				lastClassPart = CName
			} else if lastClassPart == "parent" {
				lastClassPart = eName
				enametmp := eName
				_, exist := Extends[enametmp]
				if exist {
					for true  {
						lastClassPart = lastClassPart + "|" + Extends[enametmp]
						enametmp = Extends[enametmp]
						// we should go up the chain of class dependency 
						_, exist := Extends[enametmp]
						if !exist {
							break
						}
					}
				}
			}
			l.Log(l.Info,"static call resolves to %s", lastClassPart)
			break
		case *name.FullyQualified:
			classname := class.(*name.FullyQualified)
			if namespacedName, ok := d.NsResolver.ResolvedNames[classname]; ok {
				lastClassPart = namespacedName
			} else {
				l.Log(l.Error,"static call's class is not resovled %s", File)
			}
			break
		case *expr.Variable:
			classname := class.(*expr.Variable).VarName.(*node.Identifier).Value
			if classname == "self" {
				lastClassPart = CName
			} else if classname == "parent" {
				lastClassPart = eName
				enametmp := eName
				_, exist := Extends[enametmp]
				if exist {
					for true  {
						lastClassPart = lastClassPart + "|" + Extends[enametmp]
						enametmp = Extends[enametmp]
						_, exist := Extends[enametmp]
						if !exist {
							break
						}
					}
				}
			}
			break
		default:
			dynamic = 1
			lastClassPart = d.GCProcessStringExpr(File, class, Curcls)
		}

		switch call.(type) {
		case *node.Identifier:
			callName = call.(*node.Identifier).Value
			break
		default:
			dynamic = 1
			callName = d.GCProcessStringExpr(File, call, Curcls)
			break
		}

		if dynamic == 1 {
			Dyn_func_call += 1
		} else {
			Static_func_call += 1
		}
		if contain_dotStar("("+ lastClassPart + ")\\" + callName) {
			if infunc == true {
				UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = functionName+ "|" + fname
			} else if inmeth == true {
				UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = methodName+ "|" + fname
			} else{
				UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = "main"+ "|" + fname
			}
			l.Log(l.Critical, "%s (%s-->%s)", "("+ lastClassPart + ")\\" + callName, fname, d.Positions[n])
		}

		if infunc == true{
			CG[functionName+"|"+fname] = CG[functionName+"|"+fname] + "#("+ lastClassPart + ")\\" + callName + "#" +strconv.Itoa(num_args)

		} else if inmeth == true {
			CG[methodName+"|"+fname] = CG[methodName+"|"+fname] + "#(" + lastClassPart + ")\\" + callName + "#" +strconv.Itoa(num_args)

		} else {
			CG["main"+"|"+fname] = CG["main"+"|"+fname] + "#(" + lastClassPart + ")\\" +  callName + "#" +strconv.Itoa(num_args)

		}
		// For Call-graph
		FunctionCalls[lastClassPart + "\\" + callName ] = true

		break

	case "*expr.New":
		Total_ins += 1
		class := n.(*expr.New).Class
		className, ok := class.(*name.Name)
		num_args := 0
		if tmp := n.(*expr.New).ArgumentList; tmp != nil{
			num_args = len(n.(*expr.New).ArgumentList.Arguments)
		}
		l.Log(l.Info, "the new ex is %s %s", NodeSource(&n), File)
		newopclsName := ""
		if ok {
			Static_ins += 1
			if namespacedName, ok := d.NsResolver.ResolvedNames[className]; ok {
				newopclsName = namespacedName
			} else {
				lastClassPart, ok := className.Parts[len(className.Parts)-1].(*name.NamePart)
				if !ok {
					l.Log(l.Warning, "expr.New not handled: %s", NodeSource(&n))
					break
				}
				newopclsName = lastClassPart.Value
			}
		} else if className, ok := class.(*name.FullyQualified); ok {
			Static_ins += 1
			if namespacedName, ok := d.NsResolver.ResolvedNames[className]; ok {
				newopclsName = namespacedName
			} else {
				lastClassPart := className.Parts[len(className.Parts)-1].(*name.NamePart)
				newopclsName = lastClassPart.Value
			}
		} else {
			Dyn_ins += 1
			newopclsName = d.GCProcessStringExpr(File, class, Curcls)
			l.Log(l.Info, "the resolved instantiated class is %s %s", newopclsName, File)
			if newopclsName == ".*" {
				l.Log(l.Info, "expr.New is not resovled: %s %s", NodeSource(&n), File)
				break
			}
			newopclsName = "(" + newopclsName+ ")"
		}
		if newopclsName == "ReflectionClass" {
			Static_ins += 1
			if num_args >= 0 {
				arg1, ok := n.(*expr.New).ArgumentList.Arguments[0].(*node.Argument)
				if ok {
					cls := d.GCProcessStringExpr(File, arg1.Expr, Curcls)
					if cls != ".*" {
						newopclsName = cls
					}
				}
			}
		}
		if newopclsName != ""  && newopclsName != ".*"{
			if infunc == true{
				CG[functionName+"|"+fname] = CG[functionName+"|"+fname] + "#"+ newopclsName + "\\__construct" + "#" +strconv.Itoa(num_args)
			} else if inmeth == true {
				CG[methodName+"|"+fname] = CG[methodName+"|"+fname] + "#" + newopclsName + "\\__construct" + "#" +strconv.Itoa(num_args)
			} else {
				CG["main"+"|"+fname] = CG["main"+"|"+fname] + "#" + newopclsName + "\\__construct" + "#" +strconv.Itoa(num_args)
			}


		}
	}
	return true
}

// GetChildrenVisitor is invoked at every node parameter that contains children nodes
func (d Dumper) GetChildrenVisitor(key string) walker.Visitor {
	return Dumper{d.Writer, d.Indent + "    ", d.Comments, d.Positions, d.NsResolver}
}

// LeaveNode is invoked after node process
func (d Dumper) LeaveNode(w walker.Walkable) {
	//parse := false
	n := w.(node.Node)

	switch reflect.TypeOf(n).String() {
	case "*stmt.Class":
		CName = ""
		eName = ""
		cg_incls = 0
		ClearUsedTraits()
		break
	case "*stmt.Trait":
		tName = ""
		break
	case "*stmt.Function":
		infunc = false
		functionName = "main"

	case "*stmt.ClassMethod":
		inmeth = false
		methodName = "main"

	}
}

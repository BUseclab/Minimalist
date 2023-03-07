// Package visitor contains walker.visitor implementations
package visitor

import (
	"strings"
	"reflect"
	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/name"
	"github.com/z7zmey/php-parser/node/scalar"
	"github.com/z7zmey/php-parser/node/stmt"
	"github.com/z7zmey/php-parser/node/expr"
	l "php-cg/scan-project/logger"
	"github.com/z7zmey/php-parser/walker"
)



// Dumper writes ast hierarchy to an io.Writer
// Also prints comments and positions attached to nodes


var cname string = "" // hold the name of class
var ename string = "" // hold the name of extended class
var tname string = "" // hold the name of trait
var intname string = "" // hold the name of the interface
var functionname string
var methodname string
var def_inmeth = 0
var def_incls = 0
var def_infunc = 0
var def_inint = 0
var MethodSummary map[string]string 
var UsedNamespaceSummary map[string][]string
var NamespaceSummary map[string]string
var ImplementedIntface map[string][] string
// EnterNode is invoked at every node in hierarchy



func (d DefWalker) EnterNode(w walker.Walkable) bool {

	n := w.(node.Node)
	fname := RelativePath

	switch reflect.TypeOf(n).String() {
	
	case "*stmt.Use":
		use := n.(*stmt.Use)
		useParts, ok := use.Use.(*name.Name)
		if ok {
			ns := ""
			first := true
			err := false
		for _, part := range useParts.Parts {
			val, ok := part.(*name.NamePart)
			if ok {
				if first {
					ns = val.Value
					first = false
				} else {
					ns += "\\" + val.Value
				}
			} else {
				err = true
			}
		}
		if !err {
			UsedNamespaceSummary[File] = append(UsedNamespaceSummary[File],ns)
		}
		}
	case "*stmt.Interface":
		intface := n.(*stmt.Interface)
		def_inint = 1
		if namespacedName, ok := d.NsResolver.ResolvedNames[intface]; ok {
			intname = namespacedName
		} else {
			intName, ok := intface.InterfaceName.(*node.Identifier)
			if !ok {
				l.Log(l.Error, "interface name is not resolved %s:%s", NodeSource(&n), File)
				break
			}
			intname = intName.Value
		}
		extInt := intface.Extends
		if extInt != nil {
			for _, item := range extInt.InterfaceNames {
				if namespacedName, ok := d.NsResolver.ResolvedNames[item]; ok {
					_, exist := IntExtends[intname]
					if !exist {
						IntExtends[intname] = make(map[string]bool)
					}
					IntExtends[intname][namespacedName] = true
				} else {
					extInt, ok := item.(*name.Name)
					if !ok {
						l.Log(l.Error, "the extended interface is not resolved %s:%s", NodeSource(&n), File)
						break
					}
					intename := extInt.Parts[len(extInt.Parts)-1].(*name.NamePart).Value
					_, exist := IntExtends[intname]
					if !exist {
						IntExtends[intname] = make(map[string]bool)
					}
					IntExtends[intname][intename] = true

				}
			}
		}
	case "*stmt.Class":
		class := n.(*stmt.Class)
		def_incls = 1
		if namespacedName, ok := d.NsResolver.ResolvedNames[class]; ok {
			cname = namespacedName
			NamespaceSummary[cname] = File
			Classes[File] = append(Classes[File], cname)
		} else {
			className, ok := class.ClassName.(*node.Identifier)
			if !ok {
				l.Log(l.Error, "class name is not resolved: %s:%s", NodeSource(&n), File)
				break
			}
			cname = className.Value
			Classes[File] = append(Classes[File], cname)
		}
		extClass := class.Extends
		if extClass != nil {
			if namespacedName, ok := d.NsResolver.ResolvedNames[extClass.ClassName]; ok {
				ename = namespacedName
			} else {
				extName, ok := extClass.ClassName.(*name.Name)
				if !ok {
					l.Log(l.Error, "extended class name is not resolved: %s", NodeSource(&n), File)
					break
				}
				ename = extName.Parts[len(extName.Parts)-1].(*name.NamePart).Value
			}
			Extends[cname] = ename // fill out the extends hashmap

			_, exist := Children[ename]
			if exist {
				Children[ename][cname] = true
			} else {
				Children[ename] = make(map[string]bool)
				Children[ename][cname] = true
			}
		}
		intface := class.Implements
		if intface != nil {
			for _, item := range intface.InterfaceNames {
				if namespacedName, ok := d.NsResolver.ResolvedNames[item]; ok {
					ImplementedIntface[namespacedName] = append(ImplementedIntface[namespacedName], cname)
				} else {
					intParts := item.(*name.Name).Parts
					intName := intParts[len(intParts)-1].(*name.NamePart).Value
					ImplementedIntface[intName] = append(ImplementedIntface[intName],cname )
				}

			}
		}
		break
	case "*stmt.Trait":
		trait := n.(*stmt.Trait)
		if namespacedName, ok := d.NsResolver.ResolvedNames[trait]; ok {
			tname = namespacedName
		} else {
			traitName, ok := trait.TraitName.(*node.Identifier)
			if !ok {
				l.Log(l.Error, "TraitName is not resolved: %s:%s", NodeSource(&n), File)
				break
			}
			tname = traitName.Value
		}

	case "*stmt.Function":
		def_infunc = 1
		function := n.(*stmt.Function)
		funcName, ok := function.FunctionName.(*node.Identifier)
		if ok {
			param_len := len(function.Params)
			param_req := 0
			for idx := 0; idx <len(function.Params);idx++ {
				if ok := function.Params[idx].(*node.Parameter).DefaultValue; ok != nil {
					param_req += 1
				}
			}
			pos := d.Positions[n]
			funcpos := 0
			if pos != nil {
				funcpos = pos.EndLine - pos.StartLine
			}
			if namespacedName, ok := d.NsResolver.ResolvedNames[function]; ok {
				functionname = namespacedName
				Functions[functionname +"|"+fname] = append(Functions[functionname+"|"+fname], funcpos, param_len, param_req)
				Funcfilemap[functionname] = File
				_ = functionname
			} else {
				functionname = funcName.Value
				Functions[functionname+"|"+fname] = append(Functions[functionname+"|"+fname], funcpos, param_len, param_req)
				Funcfilemap[functionname] = File
			}
		} else {
			l.Log(l.Error, "function name is not resolved: %s:%s", NodeSource(&n), File)
		}
		break
	case "*stmt.ClassMethod":
		def_inmeth = 1
		classmethod := n.(*stmt.ClassMethod)
		mname, ok := classmethod.MethodName.(*node.Identifier)
		if ok {
			// extract number of arguments needed
			param_len := len(classmethod.Params)
			param_req := 0
			for idx := 0; idx < len(classmethod.Params); idx++ {
				if ok :=classmethod.Params[idx].(*node.Parameter).DefaultValue; ok != nil{
					_ = ok
//					l.Log(l.Info, "the parameter default value is %s", NodeSource(&ok))
					param_req += 1
				}

			}
//			l.Log(l.Info,"(%s) [%d:%d]", mname.Value, param_len, param_req)
			_ = param_len
			pos := d.Positions[n]
			methpos := 0
			if pos != nil {
				methpos = pos.EndLine - pos.StartLine
			}
			if namespacedName, ok := d.NsResolver.ResolvedNames[classmethod]; ok {
				methodname = namespacedName
				Methods[namespacedName+"|"+fname] = append(Methods[namespacedName+"|"+fname], methpos, param_len, param_req)
				Methfilemap[namespacedName] = File
			} else {
				if cname != "" {
					methodname = cname + "\\" + mname.Value
				} else if tname != "" {
					methodname = tname + "\\" + mname.Value
				} else if intname != "" {
					methodname = intname + "\\" + mname.Value
				}
				_ = methodname
				Methods[methodname+"|"+fname] = append(Methods[methodname+"|"+fname], methpos, param_len, param_req )
				Methfilemap[methodname] = File
			}
		} else {
			l.Log(l.Error, "method name is not resolved: %s:%s", NodeSource(&n), File)
		}
		break
	case "*stmt.Return":
		// create a mapping for the returned variables
		// from each methods in the web app
		// can help us identify the type of objects returned
		// and improve our generated call-graph
		mSum := ""
		if def_inmeth == 0 || def_incls == 0 {
			break
		}
		mSum = methodname
		rstmt, ok := n.(*stmt.Return).Expr.(*expr.PropertyFetch)
		if ok {
			rVar, ok1 := rstmt.Variable.(*expr.Variable)
			if ok1 {
				rVar, ok1 := rVar.VarName.(*node.Identifier)
				rProp, ok2 := rstmt.Property.(*node.Identifier)
				if ok1 && ok2 {
					returnVal := ""
					if rVar.Value == "this" || rVar.Value == "self" {
						returnVal = cname + "#" + rProp.Value
						if methodname != "" {
							MethodSummary[mSum] = returnVal
						}
					}
				}
			}
		}
		rstmt2, ok := n.(*stmt.Return).Expr.(*expr.New)
		if ok {
			rNew, ok1 := rstmt2.Class.(*name.Name)
			if ok1 {
				if namespacedName, ok := d.NsResolver.ResolvedNames[rNew]; ok {
					MethodSummary[mSum] = namespacedName
					break
				}
				rVal, ok2 := rNew.Parts[len(rNew.Parts)-1].(*name.NamePart)
				if ok2 {
					MethodSummary[mSum] = rVal.Value
					break
				}
			} else if cName, ok2 := rstmt2.Class.(*name.FullyQualified); ok2 {
				if namespacedName, ok := d.NsResolver.ResolvedNames[cName]; ok {
					MethodSummary[mSum] = namespacedName
					break
				}
			}
		}
		rstmt3, ok := n.(*stmt.Return).Expr.(*expr.StaticPropertyFetch)
		if ok {
			rVar, ok1 := rstmt3.Class.(*node.Identifier)
			if ok1 {
				rProp, ok2 := rstmt3.Property.(*expr.Variable)
				if ok2 {
					rPropVal, ok2 := rProp.VarName.(*node.Identifier)
					if ok1 && ok2 {
						returnVal := ""
						if rVar.Value == "this" || rVar.Value == "self" || rVar.Value == "static" {
							returnVal = cname + "#" + rPropVal.Value
							if mSum != "" {
								MethodSummary[mSum] = returnVal
							}
						}
					}
				}
			}
			rvar, ok1 := rstmt3.Class.(*name.Name)
			if ok1 {
				rvar, ok1 := rvar.Parts[len(rvar.Parts)-1].(*name.NamePart)
				rProp, ok2 := rstmt3.Property.(*expr.Variable)
				if ok2 {
					rPropVal, ok2 := rProp.VarName.(*node.Identifier)
					if ok1 && ok2 {
						returnVal := ""
						if rvar.Value == "this" || rvar.Value == "self" || rvar.Value == "static" {
							returnVal = cname + "#" + rPropVal.Value
							if mSum != "" {
								MethodSummary[mSum] = returnVal
							}
						}
					}
				}
			}
		}
		rstmt4, ok := n.(*stmt.Return).Expr.(*expr.Variable)
		if ok {
			rVar, ok1 := rstmt4.VarName.(*node.Identifier)
			if ok1 {
				if rVar.Value == "this" || rVar.Value == "self"{
					MethodSummary[mSum] = cname
				} else {
					MethodSummary[mSum] = rVar.Value 
				}
			}
		}

		rstmt5, ok := n.(*stmt.Return).Expr.(*expr.ShortArray)
		if ok {
			items := rstmt5.Items
			rVar := ""
			if len(items) == 2 {
				it1, ok := items[0].(*expr.ArrayItem).Val.(*expr.Variable)
				if ok {
					if it1.VarName.(*node.Identifier).Value == "this" {
						rVar = cname
					}
				} else {
					break
				}
				if _, ok := items[1].(*expr.ArrayItem); ok {
				it2, ok := items[1].(*expr.ArrayItem).Val.(*scalar.String)
				if ok {
					tmp := strings.ReplaceAll(it2.Value, "'", "")
					tmp = strings.ReplaceAll(tmp, "\"", "")
					rVar += "\\" + tmp
					MethodSummary[mSum] = rVar
					break
				}
				it3, ok := items[1].(*expr.ArrayItem).Val.(*expr.Variable) 
				if ok {
					tmp := strings.ReplaceAll(it3.VarName.(*node.Identifier).Value, "'", "")
					tmp = strings.ReplaceAll(tmp, "\"", "")
					rVar += "#"+ tmp
					MethodSummary[mSum] = rVar
					break
				}

				}
			}
		}
		
		rstmt6, ok := n.(*stmt.Return).Expr.(*expr.Array)
		if ok {
			items := rstmt6.Items
			rVar := ""
			if len(items) == 2 {
				it1, ok := items[0].(*expr.ArrayItem).Val.(*expr.Variable)
				if ok {
					if it1.VarName.(*node.Identifier).Value == "this" {
						rVar = cname
					}
				} else {
					break
				}
				if items[1] != nil {
					it2, ok := items[1].(*expr.ArrayItem).Val.(*scalar.String)
					if ok {
						tmp := strings.ReplaceAll(it2.Value, "'", "")
						tmp = strings.ReplaceAll(tmp, "\"", "")
						rVar += "\\" + tmp
						MethodSummary[mSum] = rVar
						break
					}
					it3, ok := items[1].(*expr.ArrayItem).Val.(*expr.Variable) 
					if ok {
						tmp := strings.ReplaceAll(it3.VarName.(*node.Identifier).Value, "'", "")
						tmp = strings.ReplaceAll(tmp, "\"", "")
						rVar += "#"+ tmp
						MethodSummary[mSum] = rVar
						break
					}
				}
			}
		}
		rstmt7, ok := n.(*stmt.Return).Expr.(*expr.ArrayDimFetch)
		if ok {
			l.Log(l.Info, "came here for %s", NodeSource(&n))
			for true {
				_, ok = rstmt7.Variable.(*expr.ArrayDimFetch)
				if !ok {
					break
				}
				rstmt7 = rstmt7.Variable.(*expr.ArrayDimFetch)
			}
			rvar, ok1 := rstmt7.Variable.(*expr.StaticPropertyFetch)
			if ok1 {
				rvar1, ok1 := rvar.Class.(*node.Identifier)
				if ok1 {
					rProp, ok2 := rvar.Property.(*expr.Variable)
					if ok2 {
						rPropVal, ok2 := rProp.VarName.(*node.Identifier)
						if ok1 && ok2 {
							returnVal := ""
							if rvar1.Value == "this" || rvar1.Value == "self" || rvar1.Value == "static" {
								returnVal = cname + "#" + rPropVal.Value
								if mSum != "" {
									MethodSummary[mSum] = returnVal
								}
							}
						}
					}
				}
			}
			rVar, ok1 := rstmt7.Variable.(*expr.StaticPropertyFetch)
			if ok1 {
				 rVar1, ok1 := rVar.Class.(*name.Name)
				if ok1{
					rVar1, ok1 := rVar1.Parts[len(rVar1.Parts)-1].(*name.NamePart)
					rProp, ok2 := rVar.Property.(*expr.Variable)
					if ok2 {
						rPropVal, ok2 := rProp.VarName.(*node.Identifier)
						if ok1 && ok2 {
							returnVal := ""
							if rVar1.Value == "this" || rVar1.Value == "self" {
								returnVal = cname + "#" + rPropVal.Value
								if mSum != "" {
									MethodSummary[mSum] = returnVal
								}
							}
						}
					}
				}
			}
		}
		rstmt8, ok := n.(*stmt.Return).Expr.(*expr.FunctionCall)
		if ok {
			if _, ok := rstmt8.Function.(*name.Name); ok {
				parts := rstmt8.Function.(*name.Name).Parts
				fname := parts[len(parts)-1].(*name.NamePart).Value
				if fname == "create_function" {
					MethodSummary[mSum] = fname
				}
			}
		}
	}
	return true
}


// GetChildrenVisitor is invoked at every node parameter that contains children nodes
func (d DefWalker) GetChildrenVisitor(key string) walker.Visitor {
	return DefWalker{d.Writer, d.Indent + "    ", d.Comments, d.Positions, d.NsResolver }
}

// LeaveNode is invoked after node process
func (d DefWalker) LeaveNode(w walker.Walkable) {
	//parse := false
	n := w.(node.Node)

	switch reflect.TypeOf(n).String() {
	case "*stmt.Class":
		cname = ""
		def_incls = 0 
		break
	case "*stmt.Function":
		def_infunc = 0
		break
	case "*stmt.Inteface":
		intname = ""
		def_inint = 0
	case "*stmt.Trait":
		tname = ""
		break
	case "*stmt.ClassMethod":
		def_inmeth = 0
		break
	}
}


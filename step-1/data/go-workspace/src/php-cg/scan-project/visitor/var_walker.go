// Package visitor contains walker.visitor implementations
package visitor

import (
	"regexp"
	"unicode/utf8"
	"reflect"
	"strings"
	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/name"
	"github.com/z7zmey/php-parser/node/stmt"
	"github.com/z7zmey/php-parser/node/expr/assign"
	"github.com/z7zmey/php-parser/node/scalar"
	l "php-cg/scan-project/logger"
	"github.com/z7zmey/php-parser/walker"
)



// Dumper writes ast hierarchy to an io.Writer
// Also prints comments and positions attached to nodes

type VariableTree struct {

	Name string // variable name holder
	Content string // holds the content if not dynamic
	Dynamic bool // defines whether the variable content is dynamic or not
	Op string // holds the operation (usually between variables such as concat)
	Children []VariableTree // holds all possible value for the specific value with the name of "Name"
}
type vt = VariableTree
var VarForest = make(map[string]VariableTree)

var VarTrack = make(map[string]map[string]string)
var var_cname string = "" // hold the name of class
var var_mname string = ""
var var_fname string = ""

var ArgsType = make(map[string]string)
var Globalvars = make(map[string]bool)
var Currentcls = -1

var Regexignore = "{}[]<>#$%@!~=+? :&;,'"
var Scopes [] node.Node

var No_more_var = false

var Improved_vartrack = false

// waiting queue data structure to hold values that needs to be filled
type Vardep struct {
	Src_file string
	Fix_var string
	Asgn_node []node.Node
}
type Vd = Vardep
var WaitingQueue = make(map[string][]Vardep)

// this one is responsible for local dependency of variables
type Localvardep struct {
	Src_var string
	Dst_var string
}
type Lvd = Localvardep
var LocalWaitingQueue = make(map[string][]Lvd)
func ClearScopes() {
	Scopes = nil
}

func Set_local_var_assign(file string, var_name string, value string) {
	set_local_var_assign(file, var_name, value)
}

func set_local_var_assign(file string, var_name string, value string) {
	_, exist := VarTrack[file]
	if !exist {
		VarTrack[file] = make(map[string]string)
	}
	if strings.ContainsAny(value, Regexignore){
		return
	}
	if value == "" || value == "*" || !utf8.ValidString(value){
		return
	}
	if strings.ContainsAny(value, "\t\n []{}") {
		return
	}

	for (strings.Contains(value, "||")) {
		value = strings.ReplaceAll(value, "||","|")
	}
	if strings.HasSuffix(value, "|") {
		value = value[:len(value)-1]
	}
	if strings.HasPrefix(value, "|") {
		value = value[1:len(value)]
	}

	_, exist = VarTrack[file][var_name]
	if !exist {
		VarTrack[file][var_name] = value
		l.Log(l.Info,"adding the value for variable %s: [%s]", var_name, value)
		return
	}
	if exist {
		
		if strings.Contains(value, VarTrack[file][var_name]) {
			VarTrack[file][var_name] = value
			return
		}
		if !strings.ContainsAny(value, "()") {
			values := strings.Split(value,"|")
			for _, item := range values {
				res, err := regexp.MatchString(VarTrack[file][var_name], item)
				if res && err != nil {
					continue
				}
				VarTrack[file][var_name] += "|" + item
			}
			l.Log(l.Info, "the value set for %s is [%s] (%s)",var_name, VarTrack[file][var_name], file)
			return
		} else {
		res, err := regexp.MatchString(VarTrack[file][var_name], value)
		if res && err != nil{
			return
		}
		VarTrack[file][var_name] = VarTrack[file][var_name] + "|" + value
		l.Log(l.Info, "the value set for %s is [%s] (%s)",var_name, VarTrack[file][var_name], file)
		return

		}
	}

}
func set_var_assign(var_name string, value string) {
	if strings.ContainsAny(value, Regexignore){
		return
	}
	if value == "" || value == "*" || !utf8.ValidString(value){
		return
	}
	if strings.ContainsAny(value, "\t\n ") {
		return
	}

	for (strings.Contains(value, "||")) {
		value = strings.ReplaceAll(value, "||","|")
	}

	varmutex.Lock()
	_, exist := VarAssigns[var_name]
	varmutex.Unlock()
	_, err := regexp.Compile(value)
	if err != nil {
	}

	if !exist {
		varmutex.Lock()
		VarAssigns[var_name] = value
		varmutex.Unlock()
		return
	}
	if exist {
		if strings.Contains(VarAssigns[var_name], value) {
			return
		}
		_, err2 := regexp.Compile(VarAssigns[var_name])

		if err2 != nil {
		}
		
		res, err := regexp.MatchString(VarAssigns[var_name], value)
		if res && err != nil{
			return
		}

		if strings.Contains(value, VarAssigns[var_name]) {
			VarAssigns[var_name] = value
		}
		VarAssigns[var_name] = VarAssigns[var_name] + "|" + value

		l.Log(l.Info,"appending the value for the variable %s: [%s](%s)", var_name, value, File)
		return
	}
}

func ClearArgsType() {
	ArgsType = make(map[string]string)
}

func ClearGlobalArgs() {
	Globalvars = make(map[string]bool)
}
func (d VarWalker) EnterNode(w walker.Walkable) bool {
	n := w.(node.Node)

	return d.HandleNode(n)
}

func (d VarWalker) HandleNode(n node.Node) bool {
	switch reflect.TypeOf(n).String() {

	case "*stmt.Class":
		Currentcls += 1
		if len(Classes[File]) >= Currentcls {
			l.Log(l.Info, "current class is %s", Classes[File][Currentcls])
			var_cname = Classes[File][Currentcls]
		} else {
			var_cname = ".*"
		}
	case "*stmt.UseList":
		Scopes = append(Scopes, n)
		break
	case "*stmt.Namespace":
		Scopes = append(Scopes, n)
		break
	case "*stmt.Function":
		function := n.(*stmt.Function)
		funcName, ok := function.FunctionName.(*node.Identifier)
		if ok {
			if namespacedName, ok := d.NsResolver.ResolvedNames[function]; ok {
				var_fname = namespacedName
			} else {
				var_fname = funcName.Value
			}
		}
		ClearArgsType()
		for idx := 0 ; idx < len(function.Params); idx++ {
			if vt := function.Params[idx].(*node.Parameter).VariableType; vt != nil {
				if namespacedName, ok := d.NsResolver.ResolvedNames[vt]; ok {
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
		classmethod := n.(*stmt.ClassMethod)
		methName, ok := classmethod.MethodName.(*node.Identifier)
		if ok {
			if namespacedName, ok := d.NsResolver.ResolvedNames[classmethod]; ok  {
				var_mname = namespacedName
			} else {
				var_mname = var_cname + "\\" + methName.Value
			}
		}
		ClearArgsType()
		for idx := 0 ; idx < len(classmethod.Params); idx++ {
			if vt := classmethod.Params[idx].(*node.Parameter).VariableType; vt != nil {
				l.Log(l.Info, "recording the argument type passed to function %s:%s", var_mname, File)
				if namespacedName, ok := d.NsResolver.ResolvedNames[vt]; ok {
					l.Log(l.Info, "the argtype is %s",namespacedName)
					vr, ok := classmethod.Params[idx].(*node.Parameter).Variable.(*expr.Variable)
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
	

	case "*stmt.Global":
		gvars := n.(*stmt.Global).Vars
		for _, item := range gvars {
			if gvar, ok := item.(*expr.Variable); ok {
				if gname, ok := gvar.VarName.(*node.Identifier); ok {
					Globalvars[gname.Value] = true
				}
			}
		}
		break

	case "*stmt.Property":
		l.Log(l.Info, "the stmt is %s (%s)", NodeSource(&n), File)
		propStmt := n.(*stmt.Property)
		prop, ok1 := propStmt.Variable.(*expr.Variable)
		val, ok2 := propStmt.Expr.(*scalar.String)
		if ok1 && ok2 {
			value := LocalProcessStringExpr(File, val, 0, Currentcls)
			lhsobj := var_cname
			lhsprop, ok := prop.VarName.(*node.Identifier)
			if ok {
			}

			value = LocalProcessStringExpr(File, val, 0, Currentcls)
			if ok {
				lhsName := lhsprop.Value
				set_local_var_assign(File, lhsobj+"#"+lhsName, value)
				l.Log(l.Info, "set %s to %s", lhsName, value)
				
			}
		}
		val2 , ok2 := propStmt.Expr.(*expr.ShortArray)
		if ok1 && ok2 {
			lhsobj := var_cname
			lhsprop, ok := prop.VarName.(*node.Identifier)
			value := LocalProcessStringExpr(File, val2, 0, Currentcls)
			if ok {
				lhsName := lhsprop.Value
				l.Log(l.Info, "set %s to %s", lhsName, value)
				set_local_var_assign(File, lhsobj+"#"+lhsName, value)
			}

		}
		val3, ok3 := propStmt.Expr.(*expr.Array)
		if ok1 && ok3 {
			lhsobj := var_cname
			lhsprop, ok := prop.VarName.(*node.Identifier)
			value := LocalProcessStringExpr(File, val3, 0, Currentcls)
			if ok {
				lhsName := lhsprop.Value
				l.Log(l.Info, "set %s to %s", lhsName, value)
				set_local_var_assign(File, lhsobj+"#"+lhsName, value)
			}
		}
		break

	case "*stmt.Expression":
		asgnStmt := n.(*stmt.Expression)
		s, ok := asgnStmt.Expr.(*assign.Assign)
		if ok {
			if Improved_vartrack {
				lhs := processName(s.Variable)
				lvars := strings.Split(lhs, "*")
				found := false
				for _, lvar := range lvars {
					if Contains(Tracked_variables, lvar) && lvar != "not found" {
						found = true
					}
				}
				if !found {
					l.Log(l.Info, "ignore the statement %s (%s)", NodeSource(&n), lvars)
					break
				}
			}
		l.Log(l.Info, "the stmt is %s (%s)", NodeSource(&n), File)
		var_name := ""
		var_value := ""
		var_local_name := ""
		var_local_value := ""
		// determine var_name -> left hand side of the assignment
		// determine var_value -> right hand side of the assignment
		switch lhs := s.Variable.(type) {
		case *expr.Variable:
			if lvar, ok := lhs.VarName.(*node.Identifier); ok {
				var_name = lvar.Value
				var_local_name = lvar.Value
				break

			} else {
				l.Log(l.Info, "WRONG")
			}
			break
		case *expr.ArrayDimFetch:
			arr, ok := lhs.Variable.(*expr.Variable)
			if ok {
			_, ok := arr.VarName.(*node.Identifier)
			arrName := ""
			if ok {
				arrName = arr.VarName.(*node.Identifier).Value
				switch idx :=lhs.Dim.(type) {
				case *expr.Variable:
					idxname := ProcessStringExpr(idx.VarName, 0)
					idxname_local := LocalProcessStringExpr(File, idx.VarName, 0, Currentcls)
					var_name = arrName + "*" + strings.Trim(idxname,"'")
					var_local_name = arrName + "*" + strings.Trim(idxname_local,"'")
					break
				case *scalar.String:
					idxname := idx.Value
					idxname_local := idx.Value
					var_name = arrName + "*" + strings.Trim(idxname,"'")
					var_local_name = arrName + "*" + strings.Trim(idxname_local, "'")
					break
				case *scalar.Lnumber:
					idxname := idx.Value
					idxname_local := idx.Value
					var_name = arrName + "*" + strings.Trim(idxname,"'")
					var_local_name = arrName + "*" + strings.Trim(idxname_local, "'")
					break
				}
			 } else {
				l.Log(l.Info, "WRONG")
			}
			}else if lvar, ok := lhs.Variable.(*expr.StaticPropertyFetch); ok {

			lVar, ok := lvar.Class.(*node.Identifier)
			if ok {
				if lVar.Value == "self" || lVar.Value == "static" {
					if prop, ok := lvar.Property.(*expr.Variable); ok {
						if _, ok := prop.VarName.(*node.Identifier); ok {
							var_name = var_cname + "#" + prop.VarName.(*node.Identifier).Value
							var_local_name = var_cname + "#" + prop.VarName.(*node.Identifier).Value
						}
					}
				}
			}
			} else {
				l.Log(l.Info, "WRONG")
			}


			break
		case *expr.PropertyFetch:
			lVar, ok := lhs.Variable.(*expr.Variable)
			if ok {
				lhsobj := lVar.VarName.(*node.Identifier).Value
				if lhsobj == "this" {
					lhsobj = var_cname
				}
				lhsprop, ok := lhs.Property.(*node.Identifier)
				if ok {
					lhspropVal := lhsprop.Value
					var_name = lhsobj + "#" + lhspropVal
					var_local_name = lhsobj + "#" + lhspropVal
				} else if lhsprop, ok := lhs.Property.(*expr.Variable); ok{
					lhsprop, ok := lhsprop.VarName.(*node.Identifier)
					if ok {
						lhspropVal := lhsprop.Value
						var_name = lhsobj + "#" + lhspropVal
						var_local_name = lhsobj + "#" + lhspropVal
					} else {
					l.Log(l.Info, "WRONG")
					}
				} else {
					l.Log(l.Info, "WRONG")
				}
			} else {
				l.Log(l.Info, "WRONG")
			}
			break
		case *expr.StaticPropertyFetch:
			lVar, ok := lhs.Class.(*name.Name)
			if ok {
				lVar, ok := lVar.Parts[len(lVar.Parts)-1].(*name.NamePart)
				lProp, ok2 := lhs.Property.(*expr.Variable)
				if ok && ok2{
					lPropVal, ok := lProp.VarName.(*node.Identifier)
					if ok {
						if lVar.Value == "self" {
							var_name = var_cname + "#" + lPropVal.Value
							var_local_name = var_cname + "#" + lPropVal.Value
						} else {
					l.Log(l.Info, "WRONG")
						}
					} else {
				l.Log(l.Info, "WRONG")
					}
				} else {
					l.Log(l.Info, "WRONG")
				}
			}
			break
		default:
			var_name = ProcessStringExpr(lhs, 0)
			var_local_name = ProcessStringExpr(lhs, 0)
			l.Log(l.Info, "for left hand side we came to default")
			break
		}

		switch v := s.Expression.(type) {
		case *expr.New:
			switch v.Class.(type) {
			case *name.Name:
				classname := v.Class.(*name.Name)
				if namespacedName, ok := d.NsResolver.ResolvedNames[classname]; ok {
					var_value = namespacedName
					var_local_value = namespacedName
				} else {
					l.Log(l.Info,"WRONG")
				}
				break
			case *name.FullyQualified:
				classname := v.Class.(*name.FullyQualified)
				if namespacedName, ok := d.NsResolver.ResolvedNames[classname]; ok {
					var_value = namespacedName
					var_local_value = namespacedName
				} else {
					l.Log(l.Info,"WRONG")
				}
				break
			case *expr.Variable:
				classname := v.Class.(*expr.Variable).VarName.(*node.Identifier).Value
				_, exist := VarAssigns[classname]
				if exist {
					var_value = VarAssigns[classname]
				} else {
					var_value = ".*"
				}
				_, exist = VarTrack[File][classname]
				if exist {
					var_local_value = VarTrack[File][classname]
				}
				break
			default:
 				var_value = ProcessStringExpr(s.Expression, 0)
				var_local_value = LocalProcessStringExpr(File, s.Expression, 0, Currentcls)
				break
			}
		case *expr.StaticCall:
			class := v.Class
			call := v.Call
			switch  class.(type) {
			case *name.Name:
				classname := v.Class.(*name.Name)
				if namespacedName, ok := d.NsResolver.ResolvedNames[classname]; ok {
					var_value = namespacedName
					var_local_value = namespacedName
				} else {
					l.Log(l.Info,"WRONG")
				}
				break
			case *name.FullyQualified:
				classname := v.Class.(*name.FullyQualified)
				if namespacedName, ok := d.NsResolver.ResolvedNames[classname]; ok {
					var_value = namespacedName
					var_local_value = namespacedName
				} else {
					l.Log(l.Info,"WRONG")
				}
				break
			case *expr.Variable:
				classname := v.Class.(*expr.Variable).VarName.(*node.Identifier).Value
				_, exist := VarAssigns[classname]
				if exist {
					var_value = VarAssigns[classname]
				} else {
					var_value = ".*"
				}
				_, exist = VarTrack[File][classname]
				if exist {
					var_local_value = VarTrack[File][classname]
				}
				break
			default:
 				var_value = ProcessStringExpr(v.Class, 0)
				var_local_value = LocalProcessStringExpr(File, s.Expression, 0, Currentcls)
				break
			}
			switch call.(type) {
			case *node.Identifier:
				var_value += "\\" + v.Call.(*node.Identifier).Value
				var_local_value += "\\" + v.Call.(*node.Identifier).Value
				break
			default:
				var_value += "\\" + ProcessStringExpr(v.Call, 0)
				var_local_value += "\\" + LocalProcessStringExpr(File, v.Call, 0, Currentcls)
				l.Log(l.Info, "static call local (%s) global(%s) File", var_local_value, var_value, File )
				break
			}
			if _, exist := MethodSummary[var_value]; exist {
				if !strings.Contains(MethodSummary[var_value], "#") {
					var_value = MethodSummary[var_value]
				} else if _, exist := VarAssigns[MethodSummary[var_value]]; exist {
					var_value = VarAssigns[MethodSummary[var_value]]
				} else {
					l.Log(l.Info, "WRONG")
				}
			} else {
				var_value = ".*"
			}
			if _, exist := MethodSummary[var_local_value]; exist {
				if !strings.Contains(MethodSummary[var_local_value], "#") {
					if _, exist := VarTrack[Methfilemap[var_local_value]][MethodSummary[var_local_value]]; exist {
						var_local_value = VarTrack[Methfilemap[var_local_value]][MethodSummary[var_local_value]]
					} else {
						var_local_value = MethodSummary[var_local_value]
					}
				} else if _, exist := VarTrack[File][MethodSummary[var_local_value]]; exist {
					var_local_value = VarTrack[File][MethodSummary[var_local_value]]
				} else if _, exist := VarTrack[Methfilemap[var_local_value]][MethodSummary[var_local_value]]; exist{
					// we searched for the value in the file that has the method
					var_local_value = VarTrack[Methfilemap[var_local_value]][MethodSummary[var_local_value]]
				} else {
					if !No_more_var {
						holdon := Vd{}
						for _, usenode := range (Scopes) {
							holdon.Asgn_node = append(holdon.Asgn_node, usenode)
						}
						holdon.Asgn_node = append(holdon.Asgn_node, n)
						holdon.Fix_var = var_local_name
						holdon.Src_file = File
						if !ExistWaitQueue(holdon, Methfilemap[var_local_value]) {
						l.Log(l.Info, "have to wait for file %s to be procsssed",Methfilemap[var_local_value])
						WaitingQueue[Methfilemap[var_local_value]] = append(WaitingQueue[Methfilemap[var_local_value]], holdon)

						localhold := Lvd{}
						localhold.Src_var = ""
						localhold.Dst_var = var_local_name
						LocalWaitingQueue[File] = append(LocalWaitingQueue[File], localhold)
						l.Log(l.Info, "add variable (%s) [%s]\n",var_local_name, File)

						}
					}
					var_local_value = "not found"
				}
			} else {
				var_local_value = ".*"	
			}
			l.Log(l.Info,"statical call resolves to %s", var_value)
			l.Log(l.Info,"statical call resolves to %s", var_local_value)
			break
		case *expr.MethodCall:
			cls := LocalProcessStringExpr(File, v.Variable, 0, Currentcls)
			meth := LocalProcessStringExpr(File,v.Method, 0, Currentcls)
			if (cls == "not found" || meth == "not found"){
				if !No_more_var {
					holdon := Vd{}
					for _, usenode := range (Scopes) {
						holdon.Asgn_node = append(holdon.Asgn_node, usenode)
					}
					holdon.Asgn_node = append(holdon.Asgn_node, n)
					holdon.Src_file = File
					holdon.Fix_var = var_local_name
					if !ExistWaitQueue(holdon, File) {
					WaitingQueue[File] = append(WaitingQueue[File], holdon)
	
					localhold := Lvd{}
					if cls == "not found" {
						localhold.Src_var = processName(v.Variable)
					} else {
						localhold.Src_var = processName(v.Method)
					}
					localhold.Dst_var = var_local_name
					LocalWaitingQueue[File] = append(LocalWaitingQueue[File], localhold)
					}
				}
				var_local_value = "not found"
			} else {
				if strings.Contains(cls, "|") {
					cls_copy := strings.ReplaceAll(cls, "\\", "\\\\")
					cls_copy = strings.ReplaceAll(cls_copy, "_", "\\_")
					lookupItem := "(" + cls_copy +")\\\\" + meth
					found_match := false
					for k, v := range MethodSummary {
						if res, _ :=regexp.MatchString(lookupItem, k); res {
							found_match = true
							if !strings.Contains(v, "#") {
								if _, exist := VarTrack[Methfilemap[k]][v]; exist {
									v = VarTrack[Methfilemap[k]][v]
								}
								if var_local_value != ""  {
									var_local_value += "|" + v
								} else {
									var_local_value = v
								}
							} else if _, exist := VarTrack[File][v]; exist {
								var_local_value = VarTrack[File][v]
							} else if _, exist := VarTrack[Methfilemap[k]][v]; exist {
								var_local_value = VarTrack[Methfilemap[k]][v]
							} else {
								if !No_more_var {
									l.Log(l.Info, "have to wait for file %s to be procsssed",Methfilemap[v])
									holdon := Vd{}
									for _, usenode := range (Scopes) {
										holdon.Asgn_node = append(holdon.Asgn_node, usenode)
									}
									holdon.Asgn_node = append(holdon.Asgn_node, n)
									holdon.Fix_var = var_local_name
									holdon.Src_file = File
									WaitingQueue[Methfilemap[v]] = append(WaitingQueue[Methfilemap[v]], holdon)

									localhold := Lvd{}
									localhold.Src_var = ""
									localhold.Dst_var = var_local_name
									LocalWaitingQueue[File] = append(LocalWaitingQueue[File], localhold)
								}
								var_local_value = "not found"
							}
						}
					}
					if found_match == false {
						l.Log(l.Info,"didn't find any match :(")
						var_local_value = ".*"
					}
				} else {
				res := get_ancestors_list(cls)
				lookupItem := cls + "\\" + meth
				if _, exist := MethodSummary[lookupItem]; !exist {
					// if the methodsummary doesn't exist
					// we go look into parent classes
					for _, extcls := range res {
						lookupItem = extcls + "\\" + meth
						if _, exist := MethodSummary[lookupItem]; exist {
							cls = extcls
						}
					}
				}
				var_local_value = cls + "\\" + meth
				if _, exist := MethodSummary[var_local_value]; exist {
					if !strings.Contains(MethodSummary[var_local_value], "#") {
						var_local_value = MethodSummary[var_local_value]
					} else if _, exist := VarTrack[File][MethodSummary[var_local_value]]; exist {
						var_local_value = VarTrack[File][MethodSummary[var_local_value]]
					} else if _, exist := VarTrack[Methfilemap[var_local_value]][MethodSummary[var_local_value]]; exist{
						var_local_value = VarTrack[Methfilemap[var_local_value]][MethodSummary[var_local_value]]
					} else {
						if !No_more_var {
							l.Log(l.Info, "have to wait for file %s to be procsssed",Methfilemap[var_local_value])
							holdon := Vd{}
							for _, usenode := range (Scopes) {
								holdon.Asgn_node = append(holdon.Asgn_node, usenode)
							}
							holdon.Asgn_node = append(holdon.Asgn_node, n)
							holdon.Fix_var = var_local_name
							holdon.Src_file = File
							WaitingQueue[Methfilemap[var_local_value]] = append(WaitingQueue[Methfilemap[var_local_value]], holdon)

							localhold := Lvd{}
							localhold.Src_var = ""
							localhold.Dst_var = var_local_name
							LocalWaitingQueue[File] = append(LocalWaitingQueue[File], localhold)
						}
						var_local_value = "not found"
					}
				} else {
					l.Log(l.Info, " the method is not in methodsummary (%s)", var_local_value)
					var_local_value = ".*"
					}
				}
			}
			l.Log(l.Info, "methodcall resolves to local [%s\\%s] ", cls, meth)
			l.Log(l.Info, "the value resolves to %s ", var_local_value)
			break
		case *expr.FunctionCall:
			var_value = ProcessStringExpr(v, 0)
			var_local_value = LocalProcessStringExpr(File, v, 0, Currentcls)

		case *expr.Clone:
			var_local_value = LocalProcessStringExpr(File, v.Expr, 0, Currentcls)
			break
		case *expr.Variable:
			variable , ok:=v.VarName.(*node.Identifier)
			if ok {
				vr := variable.Value
				_, exist := VarAssigns[vr]
				if exist {
					var_value = VarAssigns[vr]
				} else {
					var_value = ".*"
				}
				_, exist = VarTrack[File][vr]
				if exist {
					var_local_value = VarTrack[File][vr]
				}
				if _, exist := ArgsType[vr]; exist {
					var_value = ArgsType[vr]
					var_local_value = ArgsType[vr]
				}
			} else {
				l.Log(l.Info,"WRONG")
			}
			break
		default:
			l.Log(l.Info, "for expression we came to default situation")
			var_value = ProcessStringExpr(v, 0)
			var_local_value = LocalProcessStringExpr(File, v, 0, Currentcls)
			break

		}

		l.Log(l.Info, "the whole thing resolves to var_local_name [%s] var_local_value [%s]", var_local_name, var_local_value)
		if strings.Contains(var_name, "GLOBALS") || strings.Contains(var_name, "SESSION")  {
			l.Log(l.Info, "set global variable %s to %s", var_local_name, var_local_value)
			set_var_assign(var_local_name, var_local_value )
		} else if _,exist := Globalvars[var_name]; exist {
			l.Log(l.Info, "set global variable GLOBALS*%s to %s", var_local_name, var_local_value)
			set_var_assign("GLOBALS*" + var_local_name, var_local_value )

		}
		set_local_var_assign(File, var_local_name, var_local_value )
	}
	break
}
	return true
}

// GetChildrenVisitor is invoked at every node parameter that contains children nodes
func (d VarWalker) GetChildrenVisitor(key string) walker.Visitor {
	return VarWalker{d.Writer, d.Indent + "    ", d.Comments, d.Positions, d.NsResolver }
}

// LeaveNode is invoked after node process
func (d VarWalker) LeaveNode(w walker.Walkable) {
	//parse := false
	n := w.(node.Node)

	switch reflect.TypeOf(n).String() {
	case "*stmt.Class":
		var_cname = ""
		ClearArgsType()
		break
	case"*stmt.ClassMethod":
		ClearArgsType()
		break
	case "*stmt.Function":
		ClearArgsType()
		break
	}
}

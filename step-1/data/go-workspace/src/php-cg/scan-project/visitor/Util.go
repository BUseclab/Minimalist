// Package visitor contains walker.visitor implementations
package visitor

import (
	"crypto/md5"
	"regexp"
	"fmt"
	"io"
	"strings"
	"strconv"
	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/name"
	"github.com/z7zmey/php-parser/node/scalar"
	"github.com/z7zmey/php-parser/node/expr/binary"
	"github.com/z7zmey/php-parser/parser"
	l "php-cg/scan-project/logger"
)


// Dumper writes ast hierarchy to an io.Writer
// Also prints comments and positions attached to nodes


type Dumper struct {
	Writer     io.Writer
	Indent     string
	Comments   parser.Comments
	Positions  parser.Positions
	NsResolver *NamespaceResolver

}

type CsaDumper struct {
	Writer     io.Writer
	Indent     string
	Comments   parser.Comments
	Positions  parser.Positions
	NsResolver *NamespaceResolver

}

type IncludeWalker struct {
	Writer     io.Writer
	Indent     string
	Comments   parser.Comments
	Positions  parser.Positions
	NsResolver *NamespaceResolver

}

type DefWalker struct {
	Writer     io.Writer
	Indent     string
	Comments   parser.Comments
	Positions  parser.Positions
	NsResolver *NamespaceResolver

}

type VarWalker struct {
	Writer     io.Writer
	Indent     string
	Comments   parser.Comments
	Positions  parser.Positions
	NsResolver *NamespaceResolver

}

type TrackWalker struct {
	Writer     io.Writer
	Indent     string
	Comments   parser.Comments
	Positions  parser.Positions
	NsResolver *NamespaceResolver

}

type ConstWalker struct {
	Writer    io.Writer
	Indent    string
	Comments  parser.Comments
	Positions parser.Positions

}

func GetHashStmts(nodes [] node.Node) [16]byte{
	res := ""
	for _, n := range nodes {
		res += NodeSource(&n)
	}
	return md5.Sum([]byte(res))

}

func ExistWaitQueue(holdon Vd,file string) bool {
	if holdon.Src_file != "" {
		for _, item := range WaitingQueue[file] {
			if item.Src_file == holdon.Src_file && item.Fix_var == holdon.Fix_var && GetHashStmts(item.Asgn_node) == GetHashStmts(holdon.Asgn_node) {
				return true
			}
		}
	}

	return false
}

func ClearUsedTraits() {
	usedTraits = nil
}

func get_ancestors_list(cls string) []string {
	var res []string
	first := true
	_, exist := Extends[cls]
	if exist {
		for true {
			if first {
				first = false
				res = append(res, Extends[cls])
				cls = Extends[cls]
				_, exist := Extends[cls]
				if !exist {
					break
				}
			} else {
				res = append(res, Extends[cls])
				cls = Extends[cls]
				_, exist := Extends[cls]
				if !exist {
					break
				}
			}
		}
	}
	return res
}
func get_ancestors(cls string) string {
	res := ""
	first := true
	_, exist := Extends[cls]
	if exist {
		for true {
			if first {
				first = false
				res = Extends[cls]
				cls = Extends[cls]
				_, exist := Extends[cls]
				if !exist {
					break
				}
			} else {
				res = res + "|" + Extends[cls]
				cls = Extends[cls]
				_, exist := Extends[cls]
				if !exist {
					break
				}
			}
		}
	}
	return res
}

func get_descendants(cls string) string {
	res := ""
	_, exist := Children[cls]
	if exist {
		child := Children[cls]
		for val, _ := range child {
			res = res + "|" + val
		}
	}
	return res
}

func put_paranthesis(input_str string) bool {
	if string(input_str[len(input_str)-1]) != ")" && string(input_str[0]) != "(" {
		return true
	}
	stack := ""
	for idx := 0; idx < len(input_str); idx++ {
		char := string(input_str[idx])
		if char == "(" {
			stack += char 
		} else {
			if len(stack) == 0 {
				return true
			} else {
				top := string(stack[len(stack)-1])
				if char == ")" && top == "(" {
					if len(stack) == 1 {
						stack = ""
					} else {
						stack = stack[:len(stack)-1]
					}
				}
			}
		}
	}
	if len(stack) != 0 {
		fmt.Printf("PARANTHESIS DOESN'T MATCH [%s]\n", input_str)
	}
	return false
}


func checkIncludedFilesforVar(file string, item string) string {
	res := ""
	first := true
	for _, incfile := range ScriptDep[file] {
		if _, exist := VarTrack[incfile][item]; exist {
			if (strings.Contains(VarTrack[incfile][item], "|")) {
				if first {
					res = "(" + VarTrack[incfile][item] + ")"
					first = false

				} else {
					res += "|(" + VarTrack[file][item] + ")"
				}
			} else {
				if first {
					res = VarTrack[incfile][item]
					first = false

				} else {
					res += "|" + VarTrack[file][item]
				}
			}
		}
	}
	return res
}

func (d Dumper) GCProcessStringExpr(file string, n node.Node, currentcls int) string {
	CallGCString += 1
	cname = ".*"
	if _, exist := Classes[file]; exist {
		if currentcls < 0 {
			cname = Classes[file][0]
		} else if len(Classes[file]) > currentcls {
			cname = Classes[file][currentcls]
		}
	}
	if cname == ".*" {
		l.Log(l.Info, "cname in localprocess is .* %s", File)
	}
	switch v:=n.(type) {
	case *scalar.String:
		s := v.Value
		if len(s) > 0 && (s[0] == '"' || s[0] == '\'') {
			s = s [1:]
		}
		if len(s) > 0 && (s[len(s)-1] == '"' || s[len(s)-1] == '\'') {
			s = s[:len(s)-1]
		}
		result := strings.Trim(s, "'")
		if strings.ContainsAny(result, "{}[]<>#$%@!~+=? '") {
			return ""
		}
		result = strings.ReplaceAll(result,"(", "")
		result = strings.ReplaceAll(result,")", "")
		result = strings.ReplaceAll(result,"|", "")
		result = strings.ReplaceAll(result, "*","")
		result = strings.ReplaceAll(result, ".","")
		result = strings.ReplaceAll(result, "::", "\\")
		if result == "\\" {
			return ""
		}
		if strings.Contains(result, "parent") {
			tmp := get_ancestors(cname)
			if tmp != "" {
				result = strings.ReplaceAll(result, "parent", tmp)
			}
		} else if strings.Contains(result, "self") {
			if cname != "" {
				result = strings.ReplaceAll(result, "self", cname)
			}
		}

		return result
	case *node.Identifier:
		return v.Value

	case *scalar.MagicConstant:
		s := v.Value
		if s == "__FUNCITON__" {
			return ""
		} else if s == "__CLASS__" {
			return cname
		}
	
	case *expr.Closure:
		return ""
	
	case *binary.Concat:
		result := d.GCProcessStringExpr(file, v.Left, currentcls)
		result2 := d.GCProcessStringExpr(file, v.Right, currentcls)
		l.Log(l.Info, "the returned concat op is %s:%s", result, result2)
		if result == ".*" && result2 == ".*" {
			return result
		}
		if result == "" && result2 == "" {
			return result
		}
		if result == "" {
			return result2
		} else if result2 == "" {
			return result
		}
		if strings.Contains(result, "|") {
			// extra check to see if we should put paranthesis
			if put_paranthesis(result) {
				result = "(" + result + ")"
			}
		}
		if strings.Contains(result2, "|") {
			if put_paranthesis(result2) {
				result2 = "(" + result2 + ")"
			}
		}
		res := result + result2
		res = strings.ReplaceAll(res, "::","\\")
		return res

	case *expr.ConstFetch:
		if _, ok := v.Constant.(*name.Name); ok {
			constIdentifier := v.Constant.(*name.Name).Parts[0].(*name.NamePart).Value
			return constIdentifier
		}
		if _, ok := v.Constant.(*name.FullyQualified); ok {
			constIdentifier := v.Constant.(*name.FullyQualified).Parts[0].(*name.NamePart).Value
			return constIdentifier
		}
	case *scalar.Lnumber:
		return v.Value

	case *scalar.Encapsed:
		parts := v.Parts
		res  := ""
		for _, part  := range parts {
			ret := d.GCProcessStringExpr(file, part, currentcls)
			// concat the results
			if ret == ".*" && res == ".*"{
			} else {
				res = res + ret
			}
		}
		return res

	case *expr.Variable:
		switch v.VarName.(type) {
		case  *node.Identifier:
			varname  := v.VarName.(*node.Identifier).Value
			if varname == "this"{
				cls := cname
				res := get_ancestors(cls)
				if res != "" {
					res += "|" + cls
				}
				res = get_descendants(cls)
				if res != "" {
					res += "|" + cls
				}
				if res != "" {
					return res 
				} else {
					return cls
				}
			}
			_, exist := VarTrack[file][varname]
			if exist {
				return VarTrack[file][varname] 
			} else if _,exist := ArgsType[varname]; exist {
				return ArgsType[varname]
			} else if _, exist := MethodSummary[varname]; exist {
				return MethodSummary[varname]
			} else {
				res := checkIncludedFilesforVar(file, varname)
				l.Log(l.Info, "the returned value from included file is %s", res)
				if res != "" {
					return res
				}
			}

			l.Log(l.Info,"check for %s in global variables (%s) ",varname, file)
			if _, exist := VarAssigns["GLOBALS*"+varname]; exist{
				return VarAssigns["GLOBALS*"+varname] 
			}
			return ".*"
		case *expr.Variable:
			val := d.GCProcessStringExpr(file, v.VarName, currentcls)
			_, exist := VarTrack[file][val]
			if exist {
				return VarTrack[file][val]
			} else if _,exist := ArgsType[val]; exist {
				return ArgsType[val]
			} else {
				res := checkIncludedFilesforVar(file, val)
				if res != "" {
					return res
				}
			}

			_, exist = VarAssigns["GLOBALS*"+val]
			if exist  {
				l.Log(l.Info,"Used GLOBALS[%s] (%s) ",val, file)
				return VarAssigns["GLOBALS*"+val] 
			}
			return ".*"
		}
		break

	case *expr.PropertyFetch:
		var_name, ok := v.Variable.(*expr.Variable)
		if ok {
			VarName, ok := var_name.VarName.(*node.Identifier)
			if ok {
				Varname := VarName.Value
				property := d.GCProcessStringExpr(file, v.Property, currentcls)
				l.Log(l.Info,"we are asking for %s:%s in %s",Varname,property, file)
				if Varname == "this" || Varname == "self" {
					Varname = cname
					_, exist := VarTrack[file][Varname+"#"+property]
					if exist {
						return VarTrack[file][Varname+"#"+property]
					}
				}

			}
		}
		varname := d.GCProcessStringExpr(file, v.Variable, currentcls)
		property := d.GCProcessStringExpr(file, v.Property, currentcls)
		l.Log(l.Info,"we are asking for %s:%s in %s",varname,property, file)
		for key, item := range(VarTrack[file]) {
			l.Log(l.Info, "%s -> %s", key,item)
		}
		if varname == "this" {
			varname = cname 
		}

		_, exist := VarTrack[file][varname+"#"+property]
		if exist {
			return  VarTrack[file][varname+"#"+property] 
		} else {
			res := checkIncludedFilesforVar(file, varname)
			if res == "" {
				return ".*"
			}
			return res
		}

		return ".*"

	case *binary.BooleanAnd:
		return "boolean"

	case *binary.BooleanOr:
		return "boolean"

	case *expr.BooleanNot:
		return "boolean"

	case *expr.ArrayDimFetch:
		arr , ok := v.Variable.(*expr.Variable)
		if ok {
			arrName := arr.VarName.(*node.Identifier).Value
			idx, ok := v.Dim.(*expr.Variable)
			if ok {
				idxname := idx.VarName.(*node.Identifier).Value
				_, exist := VarTrack[file][arrName + "*" + strings.Trim(idxname, "'")]
				if exist {
					return VarTrack[file][arrName + "*" + strings.Trim(idxname, "'")]
				} else {
					res := checkIncludedFilesforVar(file, arrName + "*" + strings.Trim(idxname, "'"))
					if res == "" {
						return ".*"
					}
					return res
				}
			}
			idx2, ok := v.Dim.(*scalar.String)
			if ok {
				idxname := idx2.Value
				_, exist := VarTrack[file][arrName + "*" + strings.Trim(idxname,"'")]
				if exist {
					return VarTrack[file][arrName + "*" + strings.Trim(idxname,"'")]
				} else {
					res := checkIncludedFilesforVar(file, arrName + "*" + strings.Trim(idxname, "'"))
					if res == "" {
						return ".*"
					}
					return res
				}
			}
			idx3, ok := v.Dim.(*scalar.Lnumber)
			if ok {
				idxname := idx3.Value
				_, exist := VarTrack[file][arrName + "*" + strings.Trim(idxname,"'")]
				if exist {
					return VarTrack[file][arrName + "*" + strings.Trim(idxname,"'")]
				} else {
					res := checkIncludedFilesforVar(file, arrName + "*" + strings.Trim(idxname, "'"))
					if res == "" {
						return ".*"
					}
					return res
				}
			}
			_, exist := VarTrack[file][arrName]
			if exist {
				return VarTrack[file][arrName]
			} 
		}
		arr2, ok := v.Variable.(*expr.PropertyFetch)
		if ok {
                         cls := d.GCProcessStringExpr(file, arr2.Variable, currentcls)
                         if clsvar, ok := arr2.Variable.(*expr.Variable); ok {
                                 if clsvar, ok := clsvar.VarName.(*node.Identifier); ok {
                                         if clsvar.Value == "static" || clsvar.Value == "this" || clsvar.Value == "self" {
                                                 cls = cname
                                         }
                                 }
			}
                         if prop, ok := arr2.Property.(*expr.Variable); ok {
                                 if propName, ok := prop.VarName.(*node.Identifier); ok {
                                         arrName := cls + "#" + propName.Value
                                         if _, exist := VarTrack[file][arrName]; exist {
                                                 return VarTrack[file][arrName]
                                         } else {
                                                 return ".*"
                                         }
                                 }
                         }
                         if prop, ok := arr2.Property.(*node.Identifier); ok {
                                 arrName := cls + "#" + prop.Value
                                 if _, exist := VarTrack[file][arrName]; exist {
                                         return VarTrack[file][arrName]
                                 } else {
                                         return ".*"
                                 }
                         }
                 }
		// end
		break
	case *expr.ShortArray:
		items := v.Items
		if len(items) == 2 {
			it1val := ""
			it2val := ""
			it1, ok := items[0].(*expr.ArrayItem)
			if ok {
				switch v:= it1.Val.(type) {
				case *scalar.String:
					it1val = strings.Trim(v.Value, "'")
					break

				case *expr.Variable:
					it1val = v.VarName.(*node.Identifier).Value
					switch it1val {
					case "this":
						it1val = CName
						if tmp := get_ancestors(CName) ; tmp != "" {
							it1val += "|" + get_ancestors(CName)
						}
						if tmp := get_descendants(CName); tmp != "" {
							it1val += "|" + get_descendants(CName)
						}
						break
					default:
						it1val = d.GCProcessStringExpr(file, v, currentcls)
						break
					}
					break

				case *binary.Concat:
					it1val = d.GCProcessStringExpr(file, v, currentcls)
					break
	
				default:
					it1val = d.GCProcessStringExpr(file, v, currentcls)
					break
				}
			}

			// take care of second argument
			it2, ok := items[1].(*expr.ArrayItem)
			if ok {
				switch v:= it2.Val.(type) {
				case *scalar.String:
					it2val = strings.Trim(v.Value, "'")
					break
				case *expr.Variable:
					it2val = d.GCProcessStringExpr(file, v, currentcls)
					break

				case *binary.Concat:
					it2val = d.GCProcessStringExpr(file, v, currentcls)
					break
				default:
					it2val = d.GCProcessStringExpr(file, v, currentcls)
					break
				}
			}
			return "(" + it1val + ")\\(" + it2val +")"
			}
		break
	case *expr.Array:
		items := v.Items
		if len(items) == 2 {
			it1val := ""
			it2val := ""
			it1, ok := items[0].(*expr.ArrayItem)
			if ok {
				switch v:= it1.Val.(type) {
				case *scalar.String:
					it1val = strings.Trim(v.Value, "'")
					break

				case *expr.Variable:
					it1val = v.VarName.(*node.Identifier).Value
					switch it1val {
					case "this":
						it1val = CName
						if tmp := get_ancestors(CName) ; tmp != "" {
							it1val += "|" + get_ancestors(CName)
						}
						if tmp := get_descendants(CName); tmp != "" {
							it1val += "|" + get_descendants(CName)
						}
						break
					default:
						it1val = d.GCProcessStringExpr(file, v, currentcls)
						break
					}
					break

				case *binary.Concat:
					it1val = d.GCProcessStringExpr(file, v, currentcls)
					break
	
				default:
					it1val = d.GCProcessStringExpr(file, v, currentcls)
					break
				}
			}

			// take care of second argument
			it2, ok := items[1].(*expr.ArrayItem)
			if ok {
				switch v:= it2.Val.(type) {
				case *scalar.String:
					it2val = strings.Trim(v.Value, "'")
					break
				case *expr.Variable:
					it2val = d.GCProcessStringExpr(file, v, currentcls)
					break

				case *binary.Concat:
					it2val = d.GCProcessStringExpr(file, v, currentcls)
					break
				default:
					it2val = d.GCProcessStringExpr(file, v, currentcls)
					break
				}
			}
			return "(" + it1val + ")\\(" + it2val +")"
			}
		break
	case *expr.StaticPropertyFetch:
		class, ok := v.Class.(*name.Name)
		if ok {
			classname, ok := class.Parts[len(class.Parts)-1].(*name.NamePart)
			property, ok1 := v.Property.(*expr.Variable)
			if ok && ok1 {
				propval, ok := property.VarName.(*node.Identifier)
				if ok {
					if classname.Value == "self" {
						res := cname + "#" + propval.Value
						if _, exist := VarTrack[file][res]; exist {
							return VarTrack[file][res]
						}
					} else {
						l.Log(l.Info, "WRONG")
					}
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

	case *expr.FunctionCall:
		if _, ok := v.Function.(*name.Name); ok {
			parts := v.Function.(*name.Name).Parts
			fname := parts[len(parts)-1].(*name.NamePart).Value
			if fname == "create_function" {
				return "create_function"
			}
		}
		break

	case *expr.StaticCall:
		class := v.Class
		call := v.Call
		clsName := ""
		callName := ""
		switch class.(type) {
		case *name.Name:
			classname := v.Class.(*name.Name)
			if namespacedName, ok := d.NsResolver.ResolvedNames[classname]; ok {
				clsName = namespacedName
			}
			break
		case *name.FullyQualified:
			classname := v.Class.(*name.FullyQualified)
			if namespacedName, ok := d.NsResolver.ResolvedNames[classname]; ok {
				clsName = namespacedName
			}
			break
		case *expr.Variable:
			clsName = d.GCProcessStringExpr(file, v.Class.(*expr.Variable).VarName, currentcls)
			break
		default:
			clsName = d.GCProcessStringExpr(file, v.Class, currentcls)
			break
		}
		switch call.(type) {
		case *node.Identifier:
			callName = call.(*node.Identifier).Value
			break
		default :
			callName = d.GCProcessStringExpr(file, v.Call, currentcls)
		}
		if strings.Contains(clsName,"|") || strings.Contains(callName, "|") {
			lookupItem := "(" + clsName + ")\\" + callName
			for k, v := range MethodSummary {
				if res, _ := regexp.MatchString(lookupItem, k); res {
					if !strings.Contains(v, "#") {
						return MethodSummary[lookupItem]
					} else if _, exist := VarTrack[File][v]; exist {
						return VarTrack[File][v]
					} else if _, exist := VarTrack[Methfilemap[v]][v]; exist {
						return VarTrack[Methfilemap[v]][v]
					}
				}
			}
		}
		lookupItem := clsName + "\\" + callName
		if _, exist := MethodSummary[lookupItem]; exist {
			if !strings.Contains(MethodSummary[lookupItem], "#") {
				return MethodSummary[lookupItem]
			} else if _, exist := VarTrack[file][MethodSummary[lookupItem]]; exist {
				return VarTrack[file][MethodSummary[lookupItem]]
			} else if _, exist := VarTrack[Methfilemap[lookupItem]][MethodSummary[lookupItem]]; exist {
				return VarTrack[Methfilemap[lookupItem]][MethodSummary[lookupItem]]
			}
		} else {
			return "(.*)"
		}
		break

	case *expr.MethodCall:
		cls := d.GCProcessStringExpr(file, v.Variable, currentcls)
		meth := d.GCProcessStringExpr(file, v.Method, currentcls)

		l.Log(l.Info,"global cls %s meth %s (%s)",cls, meth, file)
		if strings.Contains(cls, "|") {
			cls_copy := strings.ReplaceAll(cls, "\\", "\\\\")
			cls_copy = strings.ReplaceAll(cls_copy, "_", "\\_")
			lookupItem := "(" + cls_copy +")\\\\" + meth
			for k, v := range MethodSummary {
				if res,_ := regexp.MatchString(lookupItem, k); res {
					if !strings.Contains(v, "#") {
						return v
					} else if _, exist := VarTrack[file][v]; exist {
						return VarTrack[file][v]
					} else if _, exist := VarTrack[Methfilemap[k]][v]; exist {
						return VarTrack[Methfilemap[k]][v]
					}
				}
			}
			l.Log(l.Info, "didn't find a match :/")
		} else {
			lookupItem := cls + "\\" + meth
	 		l.Log(l.Info,"the lookup item is %s", lookupItem)
			if _, exist := MethodSummary[lookupItem]; exist {
				if !strings.Contains(MethodSummary[lookupItem], "#") {
					return MethodSummary[lookupItem]
				} else if _, exist := VarTrack[file][MethodSummary[lookupItem]]; exist {
					return VarTrack[file][MethodSummary[lookupItem]]
				} else if _, exist := VarTrack[Methfilemap[lookupItem]][MethodSummary[lookupItem]]; exist {
					return VarTrack[Methfilemap[lookupItem]][MethodSummary[lookupItem]]
				}
			} else {
				l.Log(l.Info,"didn't found the lookupItem in MethodSummary")
				return "(.*)"
			}

		}
		break

		break
	}
	return "(.*)"
}
func LocalProcessStringExpr(file string, n node.Node, pat int, curcls int) string {
	cname = ".*"
	if _, exist := Classes[file]; exist {
		if curcls < 0 {
			cname = Classes[file][0]
		} else if len(Classes[file]) > curcls {
			cname = Classes[file][curcls]
		}
	}
	if cname == ".*" {
		l.Log(l.Info, "cname in localprocess is .* %s", File)
	}
	output_pattern := false
	if pat == 1 {
		output_pattern = true
	}
	switch v:=n.(type) {
	case *scalar.String:
		s := v.Value
		if len(s) > 0 && (s[0] == '"' || s[0] == '\'') {
			s = s [1:]
		}
		if len(s) > 0 && (s[len(s)-1] == '"' || s[len(s)-1] == '\'') {
			s = s[:len(s)-1]
		}
		result := strings.Trim(s, "'")
		if !output_pattern {
			if strings.ContainsAny(result, "{}[]<>#$%@!~+=? :'") {
				return ""
			}
		}
		result = strings.ReplaceAll(result,"(", "")
		result = strings.ReplaceAll(result,")", "")
		result = strings.ReplaceAll(result,"|", "")
		result = strings.ReplaceAll(result, "*","")
		result = strings.ReplaceAll(result, ".","")
		if result == "\\" {
			return ""
		}

		return result
	case *node.Identifier:
		return v.Value

	case *scalar.MagicConstant:
		s := v.Value
		if s == "__FUNCITON__" {
			return ""
		} else if s == "__CLASS__" {
			return cname
		} else if s == "__FILE__" {
			return file
		}
	case *binary.Concat:
		result := LocalProcessStringExpr(file, v.Left, pat, curcls)
		result2 := LocalProcessStringExpr(file, v.Right, pat, curcls)
		if result == ".*" && result2 == ".*" {
			return result
		}
		if result == "" && result2 == "" {
			return result
		}
		if result == "" {
			return result2
		} else if result2 == "" {
			return result
		}
		if result == "not found" && result2 == "not found" {
			return result
		} else if result == "not found" {
			return ".*" + result2
		} else if result2 == "not found" {
			return result + ".*"
		}
		if strings.Contains(result, "|") {
			if put_paranthesis(result) {
				result = "(" + result + ")"
			}
		}
		if strings.Contains(result2, "|") {
			if put_paranthesis(result2) {
				result2 = "(" + result2 + ")"
			}
		}
		res := result + result2
		if output_pattern {
			res = strings.ReplaceAll(res, "::","\\")
		}
		return res

	case *expr.ConstFetch:
		if _, ok := v.Constant.(*name.Name); ok {
			constIdentifier := v.Constant.(*name.Name).Parts[0].(*name.NamePart).Value
			return constIdentifier
		}
		if _, ok := v.Constant.(*name.FullyQualified); ok {
			constIdentifier := v.Constant.(*name.FullyQualified).Parts[0].(*name.NamePart).Value
			return constIdentifier
		}

	case *scalar.Lnumber:
		return v.Value
	
	case *scalar.Encapsed:
		parts := v.Parts
		res  := ""
		for _, part  := range parts {
			ret := LocalProcessStringExpr(file, part, pat, curcls)
			// concat the results
			if ret == ".*" && res == ".*"{
			} else {
				res = res + ret
			}
		}
		return res

	case *scalar.EncapsedStringPart:
		return v.Value
	case *expr.Variable:
		switch v.VarName.(type) {
		case  *node.Identifier:
			varname  := v.VarName.(*node.Identifier).Value
			if varname == "this"{
				res := cname
				tmpcls := res
				if tmp := get_ancestors(tmpcls) ; tmp != "" {
					res += "|" + get_ancestors(tmpcls)
				}
				if tmp := get_descendants(tmpcls); tmp != "" {
					res += "|" + get_descendants(tmpcls)
				}
				return res
			}
			_, exist := VarTrack[file][varname]
			l.Log(l.Info," the requested variable is %s", varname)
			if exist {
				l.Log(l.Info," the requested variable was found %s", VarTrack[file][varname])
				return VarTrack[file][varname] 
			} else {
				if _, exist := LocalWaitingQueue[file]; exist {
					for _, item := range LocalWaitingQueue[file] {
						l.Log(l.Info,"the variable onhold in localwq %s", item.Dst_var)
						if item.Dst_var == varname {
							return "not found"
						}
					}
				}
			}
			_, exist = VarAssigns["GLOBALS*"+varname]
			if exist  {
				l.Log(l.Info,"Used GLOBALS[%s] (%s) ",varname, file)
				return VarAssigns["GLOBALS*"+varname] 
			}
			return ".*"
		case *expr.Variable:
			val := LocalProcessStringExpr(file, v.VarName, pat, curcls)
			_, exist := VarTrack[file][val]
			if exist {
				return VarTrack[file][val]
			}
			_, exist = VarAssigns["GLOBALS*"+val]
			if exist  {
				l.Log(l.Info,"Used GLOBALS[%s] (%s) ",val, file)
				return VarAssigns["GLOBALS*"+val] 
			}
			return ".*"
		}
	case *expr.PropertyFetch:
		varname := LocalProcessStringExpr(file, v.Variable, pat, curcls)
		property := LocalProcessStringExpr(file, v.Property, pat, curcls)
		l.Log(l.Info,"The returned varname (%s) and property is (%s) curcls %d", varname, property, curcls)
		if varname == "this" {
			varname = cname
		}
		_, exist := VarTrack[file][varname+"#"+property]
		if exist {
			return  VarTrack[file][varname+"#"+property] 
		}
		return ".*"

	case *binary.BooleanAnd:
		return "boolean"

	case *binary.BooleanOr:
		return "boolean"

	case *expr.BooleanNot:
		return "boolean"

	case *expr.Closure:
		return "CLOSURE"

	case *expr.ArrayDimFetch:
		arr , ok := v.Variable.(*expr.Variable)
		if ok {
			arrName, ok := arr.VarName.(*node.Identifier)
			if ok {
				arrName := arrName.Value
			idx, ok := v.Dim.(*expr.Variable)
			if ok {
				idxname := idx.VarName.(*node.Identifier).Value
				_, exist := VarTrack[file][arrName + "*" + strings.Trim(idxname, "'")]
				if exist {
					return VarTrack[file][arrName + "*" + strings.Trim(idxname, "'")]
				} else if _, exist := VarTrack[file][arrName]; exist {
					return VarTrack[file][arrName]
				} else {
					return ".*"
				}
					// arrName + "*" + strings.Trim(idxname,"'")
			}
			idx2, ok := v.Dim.(*scalar.String)
			if ok {
				idxname := idx2.Value
				_, exist := VarTrack[file][arrName + "*" + strings.Trim(idxname,"'")]
				if exist {
					return VarTrack[file][arrName + "*" + strings.Trim(idxname,"'")]
				}
				break
			}
			}
		}
		arr1, ok := v.Variable.(*expr.StaticPropertyFetch)
		if ok {
			cls := LocalProcessStringExpr(file, arr1.Class, pat, curcls)
			if prop, ok := arr1.Property.(*expr.Variable); ok {
				if propName, ok := prop.VarName.(*node.Identifier); ok {
					if cls == "self" || cls == "static" {
						cls = cname
						arrName := cls + "#" + propName.Value
						if _, exist := VarTrack[file][arrName]; exist {
							return VarTrack[file][arrName]
						} else {
						return ".*"
						}
					}
				}
			}
		}
		arr2, ok := v.Variable.(*expr.PropertyFetch)
		if ok {
			cls := LocalProcessStringExpr(file, arr2.Variable, pat, curcls)
			if clsvar, ok := arr2.Variable.(*expr.Variable); ok {
				if clsvar, ok := clsvar.VarName.(*node.Identifier); ok {
					if clsvar.Value == "static" || clsvar.Value == "this" || clsvar.Value == "self" {
						cls = cname
					} 
				}
			}
			if prop, ok := arr2.Property.(*expr.Variable); ok {
				if propName, ok := prop.VarName.(*node.Identifier); ok {
					arrName := cls + "#" + propName.Value
					if _, exist := VarTrack[file][arrName]; exist {
						return VarTrack[file][arrName]
					} else {
						return ".*"
					}
				}
			}
			if prop, ok := arr2.Property.(*node.Identifier); ok {
				arrName := cls + "#" + prop.Value
				if _, exist := VarTrack[file][arrName]; exist {
					return VarTrack[file][arrName]
				} else {
					return ".*"
				}
			}
		}
		break
	case *expr.ShortArray:
		items := v.Items
		l.Log(l.Info," came here for stmt %s", NodeSource(&n))
		if output_pattern {

			if len(items) == 2 {
				it1val := ""
				it2val := ""

				it1, ok := items[0].(*expr.ArrayItem)
				if ok {
					switch v:= it1.Val.(type) {
					case *scalar.String:
						it1val = strings.Trim(v.Value, "'")
						break

					case *expr.Variable:
						it1val = v.VarName.(*node.Identifier).Value
						switch it1val {
						case "this":
							it1val = cname
							if tmp := get_ancestors(CName) ; tmp != "" {
								it1val += "|" + get_ancestors(CName)
							}
							if tmp := get_descendants(CName); tmp != "" {
								it1val += "|" + get_descendants(CName)
							}
							break
						default:
							it1val = LocalProcessStringExpr(file, v, pat, curcls)
							break
						}
						break

					case *binary.Concat:
						it1val = LocalProcessStringExpr(file, v, pat, curcls)
						break
					default:
						it1val = LocalProcessStringExpr(file, v, pat, curcls)
						break
					}
				}

				// take care of second argument
				it2, ok := items[1].(*expr.ArrayItem)
				if ok {
					switch v:= it2.Val.(type) {
					case *scalar.String:
						it2val = strings.Trim(v.Value, "'")
						break
					case *expr.Variable:
						it2val = LocalProcessStringExpr(file, v, pat, curcls)
						break

					case *binary.Concat:
						it2val = LocalProcessStringExpr(file, v, pat, curcls)
						break
					default:
						it2val = LocalProcessStringExpr(file, v, pat, curcls)
						break
					}
				}
				return "(" + it1val + ")\\(" + it2val +")"
				}
		} else {
			result := ""
			first := true
			for _, item := range items {
				itval := ""
				if !first {
					result += "|"
				}
				first = false
				it, ok := item.(*expr.ArrayItem)
				if ok {
					switch v:= it.Val.(type) {
					case *scalar.String:
						itval = LocalProcessStringExpr(file, v, pat, curcls)
						break

					case *expr.Variable:
						switch v.VarName.(type) {
						case *node.Identifier:
							itval = v.VarName.(*node.Identifier).Value
							switch itval {
							case "this":
								itval = cname
								if tmp := get_ancestors(cname) ; tmp != "" {
									itval += "|" + get_ancestors(cname)
								}
								if tmp := get_descendants(cname); tmp != "" {
									itval += "|" + get_descendants(cname)
								}
								break
							default:
								itval = LocalProcessStringExpr(file, v, pat, curcls)
								break
							}
							break
						case *expr.Variable:
							itval = LocalProcessStringExpr(file, v.VarName, pat, curcls)
							switch itval {
							case "this":
								itval = cname
								if tmp := get_ancestors(cname) ; tmp != "" {
									itval += "|" + get_ancestors(cname)
								}
								if tmp := get_descendants(cname); tmp != "" {
									itval += "|" + get_descendants(cname)
								}
								break
							default:
								itval = LocalProcessStringExpr(file, v, pat, curcls)
								break
							}
							break
						}

					case *binary.Concat:
						itval = LocalProcessStringExpr(file, v, pat, curcls)
						break
					default:
						itval = LocalProcessStringExpr(file, v, pat, curcls)
						break
					}
				}
				result += itval
			}
			return result
		}
		break
	case *expr.Array:
		items := v.Items
		l.Log(l.Info," came here for stmt %s", NodeSource(&n))
		if output_pattern {

			if len(items) == 2 {
				it1val := ""
				it2val := ""
				it1, ok := items[0].(*expr.ArrayItem)
				if ok {
					switch v:= it1.Val.(type) {
					case *scalar.String:
						it1val = strings.Trim(v.Value, "'")
						break

					case *expr.Variable:
						it1val = v.VarName.(*node.Identifier).Value
						switch it1val {
						case "this":
							it1val = cname
							if tmp := get_ancestors(cname) ; tmp != "" {
								it1val += "|" + get_ancestors(cname)
							}
							if tmp := get_descendants(CName); tmp != "" {
								it1val += "|" + get_descendants(cname)
							}
							break
						default:
							it1val = LocalProcessStringExpr(file, v, pat, curcls)
							break
						}
						break

					case *binary.Concat:
						it1val = LocalProcessStringExpr(file, v, pat, curcls)
						break
					default:
						it1val = LocalProcessStringExpr(file, v, pat, curcls)
						break
					}
				}

				it2, ok := items[1].(*expr.ArrayItem)
				if ok {
					switch v:= it2.Val.(type) {
					case *scalar.String:
						it2val = strings.Trim(v.Value, "'")
						break
					case *expr.Variable:
						it2val = LocalProcessStringExpr(file, v, pat, curcls)
						break

					case *binary.Concat:
						it2val = LocalProcessStringExpr(file, v, pat, curcls)
						break
					default:
						it2val = LocalProcessStringExpr(file, v, pat, curcls)
						break
					}
				}
				return "(" + it1val + ")\\(" + it2val +")"
				}
		} else {
			result := ""
			first := true
			for _, item := range items {
				itval := ""
				if !first {
					result += "|"
				}
				first = false
				it, ok := item.(*expr.ArrayItem)
				if ok {
					switch v:= it.Val.(type) {
					case *scalar.String:
						itval = LocalProcessStringExpr(file, v, pat, curcls)
						break

					case *expr.Variable:
						switch v.VarName.(type) {
						case *node.Identifier:
							itval = v.VarName.(*node.Identifier).Value
							switch itval {
							case "this":
								itval = cname
								if tmp := get_ancestors(cname) ; tmp != "" {
									itval += "|" + get_ancestors(cname)
								}
								if tmp := get_descendants(cname); tmp != "" {
									itval += "|" + get_descendants(cname)
								}
								break
							default:
								itval = LocalProcessStringExpr(file, v, pat, curcls)
								break
							}
							break
						case *expr.Variable:
							itval = LocalProcessStringExpr(file, v.VarName, pat, curcls)
							switch itval {
							case "this":
								itval = cname
								if tmp := get_ancestors(cname) ; tmp != "" {
									itval += "|" + get_ancestors(cname)
								}
								if tmp := get_descendants(cname); tmp != "" {
									itval += "|" + get_descendants(cname)
								}
								break
							case "parent":
								itval = get_ancestors(cname)
								break
							default:
								itval = LocalProcessStringExpr(file, v, pat, curcls)
								break
							}
							break
						}

					case *binary.Concat:
						itval = LocalProcessStringExpr(file, v, pat, curcls)
						break
					default:
						itval = LocalProcessStringExpr(file, v, pat, curcls)
						break
					}
				}
				result += itval
			}
			return result
		}
		break

	case *expr.New:
		newop, ok := v.Class.(*name.Name)
		if ok {
			parts := newop.Parts
			return ".*" + parts[len(parts)-1].(*name.NamePart).Value
		}
	case *expr.Ternary:
		iftrue := LocalProcessStringExpr(file, v.IfTrue, pat, curcls)
		iffalse := LocalProcessStringExpr(file, v.IfFalse, pat, curcls)
		_, ok := v.IfTrue.(*scalar.String)
		_, ok1 := v.IfFalse.(*scalar.String)
		if ok && ok1 {
			if strings.Contains(iftrue, "|") || strings.Contains(iffalse, "|"){
				return "(" + iftrue + "|" + iffalse + ")"
			}
			return iftrue + "|" + iffalse
		} else if ok {
			return iftrue
		} else if ok1 {
			return iffalse
		} 
		if iftrue != ".*" && iftrue != "" {
			if iffalse != ".*" && iffalse != "" {
				return iftrue + "|" + iffalse
			} else {
				return iftrue
			}
		} else {
			if iffalse != ".*" && iffalse != "" {
				return iffalse
			}
		}

	case *expr.FunctionCall:
		if _, ok := v.Function.(*name.Name); ok {
			parts := v.Function.(*name.Name).Parts
			fname := parts[len(parts)-1].(*name.NamePart).Value
			if fname == "create_function" {
				return "create_function"
			}
			if fname == "ucfirst" || fname == "strtolower" || fname == "ucwords" || fname == "strtoupper" {
				return ".*"
			} else if _, exist := MethodSummary[fname]; exist {
				return MethodSummary[fname]
			}
		}
		break

	case *expr.StaticCall:
		class := v.Class
		className, ok := class.(*name.Name)
		call := v.Call
		callName, ok1 := call.(*node.Identifier)

		if ok && ok1 {
			classPart := ""
			first := true
			for item, _ := range (className.Parts) {
				if first {
					first = false
					classPart = className.Parts[item].(*name.NamePart).Value
				} else {
					classPart = classPart + "\\" + className.Parts[item].(*name.NamePart).Value
				}
			}
			lookupItem := classPart + "\\" + callName.Value
			if _, exist := MethodSummary[lookupItem]; exist {
				if !strings.Contains(MethodSummary[lookupItem], "#") {
					return MethodSummary[lookupItem]
				} else if _, exist := VarTrack[file][MethodSummary[lookupItem]]; exist {
					return VarTrack[file][MethodSummary[lookupItem]]
				}
			}
			for _, ns := range (UsedNamespaceSummary[file]) {
				lookupItem := ns + "\\" + callName.Value
				if _, exist := MethodSummary[lookupItem]; exist {
				     if  !strings.Contains(MethodSummary[lookupItem], "#") {
						return MethodSummary[lookupItem]
					} else if _, exist := VarTrack[file][MethodSummary[lookupItem]]; exist {
						return VarTrack[file][MethodSummary[lookupItem]]
					} else if _, exist := VarAssigns[MethodSummary[lookupItem]]; exist {
						return VarAssigns[MethodSummary[lookupItem]]
					}
				}
			}
		}
		return ".*"
	case *expr.MethodCall:
		if pat == 0 {
		cls := LocalProcessStringExpr(file, v.Variable, pat, curcls)
		meth := LocalProcessStringExpr(file, v.Method, pat, curcls)
		l.Log(l.Info,"local cls %s meth %s (%s)",cls, meth, file)
		if strings.Contains(cls,"|") || strings.Contains(cls,".*") {
			cls_copy := strings.ReplaceAll(cls, "\\","\\\\")
			cls_copy = strings.ReplaceAll(cls, "_", "\\_")
			result := ""
			found_match := false
			lookupItem := "(" + cls_copy +")\\\\" + meth
			for k, v := range MethodSummary {
				if res, _ := regexp.MatchString(lookupItem, k); res {
					found_match = true
					if !strings.Contains(v, "#"){
						if result != "" {
							result += "|" +v
						} else {
							result = v
						}
					} else if _, exist := VarTrack[file][v]; exist {
						if result != "" {
							result += "|" + VarTrack[file][v]
						} else {
							result = VarTrack[file][v]
						}
					} else  if _, exist := VarTrack[Methfilemap[k]][v]; exist {
						if result != "" {
							result += "|" + VarTrack[Methfilemap[k]][v]
						} else {
							result = VarTrack[Methfilemap[k]][v]
						}
					}
				}
			}
			if found_match {
				return result
			} else {
				return ".*"
			}
		} else {
			lookupItem := cls + "\\" + meth
			if _, exist := MethodSummary[lookupItem]; exist {
				if !strings.Contains(MethodSummary[lookupItem], "#") {
					return MethodSummary[lookupItem]
				} else if _, exist := VarTrack[file][MethodSummary[lookupItem]]; exist {
					return VarTrack[file][MethodSummary[lookupItem]]
				} else if _, exist := VarAssigns[MethodSummary[lookupItem]]; exist {
					return VarAssigns[MethodSummary[lookupItem]]
				}
			} else {
				return ".*"
			}
		}
		}

		break
	}
	return "not found"
}

func ProcessStringExpr(n node.Node, pat int) string {
	output_pattern := false
	if pat == 1 {
		output_pattern = true
	}
	switch v:=n.(type) {
	case *scalar.String:
		s := v.Value
		if len(s) > 0 && (s[0] == '"' || s[0] == '\'') {
			s = s [1:]
		}
		if len(s) > 0 && (s[len(s)-1] == '"' || s[len(s)-1] == '\'') {
			s = s[:len(s)-1]
		}
		result := strings.Trim(s, "'")
		if !output_pattern {
			if strings.ContainsAny(result, "{}[]<>#$%@!~+=? :'") {
				return ""
			}
		}
		result = strings.ReplaceAll(result,"(", "")
		result = strings.ReplaceAll(result,")", "")
		result = strings.ReplaceAll(result,"|", "")
		result = strings.ReplaceAll(result, "*","")
		result = strings.ReplaceAll(result, ".","")
		if result == "\\" {
			return ""
		}

		return result

	case *node.Identifier:
		return v.Value

	case *scalar.MagicConstant:
		s := v.Value
		if s == "__FUNCITON__" {
			return ""
		} else if s == "__CLASS__" {
			return CName
		}
	case *binary.Concat:
		result := ProcessStringExpr(v.Left, pat)
		result2 := ProcessStringExpr(v.Right, pat)
		if result == ".*" && result2 == ".*" {
			return result
		}
		if result == "" && result2 == "" {
			return result
		}
		if result == "" {
			return result2
		} else if result2 == "" {
			return result
		}
		if strings.Contains(result, "|") {
			if put_paranthesis(result) {
				result = "(" + result + ")"
			}
		}
		if strings.Contains(result2, "|") {
			if put_paranthesis(result2) {
				result2 = "(" + result2 + ")"
			}
		}
		res := result + result2
		if output_pattern {
			res = strings.ReplaceAll(res, "::","\\")
		}
		return res

	case *expr.ConstFetch:
		if _, ok := v.Constant.(*name.Name); ok {
			constIdentifier := v.Constant.(*name.Name).Parts[0].(*name.NamePart).Value
			return constIdentifier
		}
		if _, ok := v.Constant.(*name.FullyQualified); ok {
			constIdentifier := v.Constant.(*name.FullyQualified).Parts[0].(*name.NamePart).Value
			return constIdentifier
		}

	case *scalar.Lnumber:
		return v.Value

	case *expr.Closure:
		return "CLOSURE"
	case *expr.FunctionCall:
		fname, ok := v.Function.(*name.Name)
		if ok {
			parts := fname.Parts
			functionName := parts[len(parts)-1].(*name.NamePart).Value
			if functionName == "create_function" {
				return "create_function"
			}
			if _, exist := MethodSummary[functionName]; exist {
				return MethodSummary[functionName]
			}
		}
		return ".*"

	case *scalar.EncapsedStringPart:
		return v.Value

	case *scalar.Encapsed:
		parts := v.Parts
		res  := ""
		for _, part  := range parts {
			ret := ProcessStringExpr(part, pat)
			if ret == ".*" && res == ".*"{
			} else {
				res = res + ret
			}
		}
		return res

	case *expr.Variable:
		switch v.VarName.(type) {
		case  *node.Identifier:
			varname  := v.VarName.(*node.Identifier).Value
			if varname == "this" {
				return CName
			}
			varmutex.Lock()
			_, exist := VarAssigns[varname]
			varmutex.Unlock()
			if exist {
				return VarAssigns[varname] 
			}
			varmutex.Lock()
			_, exist = VarAssigns["GLOBALS*"+varname]
			varmutex.Unlock()
			if exist  {
				return VarAssigns["GLOBALS*"+varname] 
			}
			return ".*"
		case *expr.Variable:
			val := ProcessStringExpr(v.VarName, pat)
			varmutex.Lock()
			_, exist := VarAssigns[val]
			varmutex.Unlock()
			if exist {
				return VarAssigns[val]
			}
			// check for global variables
			varmutex.Lock()
			_, exist = VarAssigns["GLOBALS*"+val]
			varmutex.Unlock()
			if exist  {
				return VarAssigns["GLOBALS*"+val] 
			}
			return ".*"

			
		}
	case *expr.PropertyFetch:
		varname := ProcessStringExpr(v.Variable, pat)
		property := ProcessStringExpr(v.Property, pat)
		varmutex.Lock()
		_, exist := VarAssigns[varname+"#"+property]
		varmutex.Unlock()
		if exist {
			return  VarAssigns[varname+"#"+property] 
		}
		return ".*"

	case *expr.ShortArray:
		items := v.Items

		if output_pattern {

			if len(items) == 2 {
				it1val := ""
				it2val := ""

				it1, ok := items[0].(*expr.ArrayItem)
				if ok {
					switch v:= it1.Val.(type) {
					case *scalar.String:
						it1val = strings.Trim(v.Value, "'")
						break

					case *expr.Variable:
						it1val = v.VarName.(*node.Identifier).Value
						switch it1val {
						case "this":
							it1val = CName
							if tmp := get_ancestors(CName) ; tmp != "" {
								it1val += "|" + get_ancestors(CName)
							}
							if tmp := get_descendants(CName); tmp != "" {
								it1val += "|" + get_descendants(CName)
							}
							break
						case "parent":
							it1val = get_ancestors(CName)
							break
						default:
							it1val = ProcessStringExpr(v, pat)
							break
						}
						break

					case *binary.Concat:
						it1val = ProcessStringExpr(v, pat)
						break
					default:
						it1val = ProcessStringExpr(v, pat)
						break
					}
				}

				it2, ok := items[1].(*expr.ArrayItem)
				if ok {
					switch v:= it2.Val.(type) {
					case *scalar.String:
						it2val = strings.Trim(v.Value, "'")
						break
					case *expr.Variable:
						it2val = ProcessStringExpr(v, pat)
						break

					case *binary.Concat:
						it2val = ProcessStringExpr(v, pat)
						break
					default:
						it2val = ProcessStringExpr(v, pat)
						break
					}
				}
				return "(" + it1val + ")\\(" + it2val +")"
				}
		} else {
			result := ""
			first := true
			for _, item := range items {
				l.Log(l.Info, "the item is %s", NodeSource(&item))
				itval := ""
				if !first {
					result += "|"
				}
				first = false
				it, ok := item.(*expr.ArrayItem)
				if ok {
					switch v:= it.Val.(type) {
					case *scalar.String:
						itval = ProcessStringExpr(v, pat)
						break

					case *expr.Variable:
						switch v.VarName.(type) {
						case *node.Identifier:
							itval = v.VarName.(*node.Identifier).Value
							switch itval {
							case "this":
								itval = CName
								if tmp := get_ancestors(CName) ; tmp != "" {
									itval += "|" + get_ancestors(CName)
								}
								if tmp := get_descendants(CName); tmp != "" {
									itval += "|" + get_descendants(CName)
								}
								break
							case "parent":
								itval = get_ancestors(CName)
								break
							default:
								itval = ProcessStringExpr(v, pat)
								break
							}
							break
						case *expr.Variable:
							l.Log(l.Error, "the node is %s",NodeSource(&n))
							itval = ProcessStringExpr(v.VarName, pat)
							switch itval {
							case "this":
								itval = CName
								if tmp := get_ancestors(CName) ; tmp != "" {
									itval += "|" + get_ancestors(CName)
								}
								if tmp := get_descendants(CName); tmp != "" {
									itval += "|" + get_descendants(CName)
								}
								break
							case "parent":
								itval = get_ancestors(CName)
								break
							default:
								itval = ProcessStringExpr(v, pat)
								break
							}
							break
						}

					case *binary.Concat:
						itval = ProcessStringExpr(v, pat)
						break
					default:
						itval = ProcessStringExpr(v, pat)
						break
					}
				}
				result += itval
			}
			return result
		}
		break
	case *expr.Array:
		items := v.Items

		if output_pattern {

			if len(items) == 2 {
				it1val := ""
				it2val := ""
				// take care of first argument

				it1, ok := items[0].(*expr.ArrayItem)
				if ok {
					switch v:= it1.Val.(type) {
					case *scalar.String:
						it1val = strings.Trim(v.Value, "'")
						break

					case *expr.Variable:
						it1val = v.VarName.(*node.Identifier).Value
						switch it1val {
						case "this":
							it1val = CName
							if tmp := get_ancestors(CName) ; tmp != "" {
								it1val += "|" + get_ancestors(CName)
							}
							if tmp := get_descendants(CName); tmp != "" {
								it1val += "|" + get_descendants(CName)
							}
							break
						case "parent":
							it1val = get_ancestors(CName)
							break
						default:
							it1val = ProcessStringExpr(v, pat)
							break
						}
						break

					case *binary.Concat:
						it1val = ProcessStringExpr(v, pat)
						break
					default:
						it1val = ProcessStringExpr(v, pat)
						break
					}
				}

				it2, ok := items[1].(*expr.ArrayItem)
				if ok {
					switch v:= it2.Val.(type) {
					case *scalar.String:
						it2val = strings.Trim(v.Value, "'")
						break
					case *expr.Variable:
						it2val = ProcessStringExpr(v, pat)
						break

					case *binary.Concat:
						it2val = ProcessStringExpr(v, pat)
						break
					default:
						it2val = ProcessStringExpr(v, pat)
						break
					}
				}
				return "(" + it1val + ")\\(" + it2val +")"
				}
		} else {
			result := ""
			first := true
			for _, item := range items {
				l.Log(l.Info, "the item is %s", NodeSource(&item))
				itval := ""
				if !first {
					result += "|"
				}
				first = false
				it, ok := item.(*expr.ArrayItem)
				if ok {
					switch v:= it.Val.(type) {
					case *scalar.String:
						itval = ProcessStringExpr(v, pat)
						break

					case *expr.Variable:
						switch v.VarName.(type) {
						case *node.Identifier:
							itval = v.VarName.(*node.Identifier).Value
							switch itval {
							case "this":
								itval = CName
								if tmp := get_ancestors(CName) ; tmp != "" {
									itval += "|" + get_ancestors(CName)
								}
								if tmp := get_descendants(CName); tmp != "" {
									itval += "|" + get_descendants(CName)
								}
								break
							case "parent":
								itval = get_ancestors(CName)
								break
							default:
								itval = ProcessStringExpr(v, pat)
								break
							}
							break
						case *expr.Variable:
							l.Log(l.Error, "the node is %s",NodeSource(&n))
							itval = ProcessStringExpr(v.VarName, pat)
							switch itval {
							case "this":
								itval = CName
								if tmp := get_ancestors(CName) ; tmp != "" {
									itval += "|" + get_ancestors(CName)
								}
								if tmp := get_descendants(CName); tmp != "" {
									itval += "|" + get_descendants(CName)
								}
								break
							case "parent":
								itval = get_ancestors(CName)
								break
							default:
								itval = ProcessStringExpr(v, pat)
								break
							}
							break
						}

					case *binary.Concat:
						itval = ProcessStringExpr(v, pat)
						break
					default:
						itval = ProcessStringExpr(v, pat)
						break
					}
				}
				result += itval
			}
			return result
		}
		break

	case *expr.Ternary:
		iftrue := ProcessStringExpr(v.IfTrue, pat)
		iffalse := ProcessStringExpr(v.IfFalse, pat)
		if _, ok := v.IfTrue.(*scalar.String); ok {
			return iftrue
		}
		if _, ok1 := v.IfFalse.(*scalar.String); ok1{
			return iffalse
		}

		if strings.Contains(iftrue, "|") || strings.Contains(iffalse, "|"){
			return "(" + iftrue + "|" + iffalse + ")"
		}
		return iftrue + "|" + iffalse

	case *binary.BooleanAnd:
		return "boolean"

	case *binary.BooleanOr:
		return "boolean"

	case *expr.BooleanNot:
		return "boolean"

	case *expr.ArrayDimFetch:
		arr , ok := v.Variable.(*expr.Variable)
		if ok {
			arrName, ok := arr.VarName.(*node.Identifier)
			if ok {
				arrName := arrName.Value
			idx, ok := v.Dim.(*expr.Variable)
			if ok {
				idxname := idx.VarName.(*node.Identifier).Value
				varmutex.Lock()
				_, exist := VarAssigns[arrName + "*" + strings.Trim(idxname, "'")]
				varmutex.Unlock()
				if exist {
					return VarAssigns[arrName + "*" + strings.Trim(idxname, "'")]
				}
					return ".*"
			}
			idx2, ok := v.Dim.(*scalar.String)
			if ok {
				idxname := idx2.Value
				varmutex.Lock()
				_, exist := VarAssigns[arrName + "*" + strings.Trim(idxname,"'")]
				varmutex.Unlock()
				if exist {
					return VarAssigns[arrName + "*" + strings.Trim(idxname,"'")]
				}
				break
			}
			}
		}
	case *expr.StaticCall:
		if pat == 0 {
		class := v.Class
		className, ok := class.(*name.Name)
		call := v.Call
		callName, ok1 := call.(*node.Identifier)

		if ok && ok1 {
			classPart := ""
			first := true
			for item, _ := range (className.Parts) {
				if first {
					first = false
					classPart = className.Parts[item].(*name.NamePart).Value
				} else {
					classPart = classPart + "\\" + className.Parts[item].(*name.NamePart).Value
				}
			}
			lookupItem := classPart + "\\" + callName.Value
			if _, exist := MethodSummary[lookupItem]; exist {
				if !strings.Contains(MethodSummary[lookupItem], "#") {
					return MethodSummary[lookupItem]
				} else if _, exist := VarAssigns[MethodSummary[lookupItem]]; exist {
					return VarAssigns[MethodSummary[lookupItem]]
				}
			}
			for _, ns := range (UsedNamespaceSummary[File]) {
				lookupItem := ns + "\\" + callName.Value
				if _, exist := MethodSummary[lookupItem]; exist {
				     if  !strings.Contains(MethodSummary[lookupItem], "#") {
						return MethodSummary[lookupItem]
					} else if _, exist := VarAssigns[MethodSummary[lookupItem]]; exist {
						return VarAssigns[MethodSummary[lookupItem]]
					}
				}
			}
		}



		}
		class := v.Class

		if _, ok := class.(*expr.Variable); ok {
			return ProcessStringExpr(class, pat)
		} else {
			className, ok1 := class.(*name.Name)
			if ok1{
				lastClassPart := className.Parts[len(className.Parts)-1].(*name.NamePart).Value
				return "(" + lastClassPart + ")"
			}
		}
		break
	case *expr.MethodCall:
		if pat == 0 {
		cls := ProcessStringExpr(v.Variable, pat)
		meth := ProcessStringExpr(v.Method, pat)

		lookupItem := cls + "\\" + meth

		if _, exist := MethodSummary[lookupItem]; exist {
			if !strings.Contains(MethodSummary[lookupItem], "#") {
				return MethodSummary[lookupItem]
			} else if _, exist := VarAssigns[MethodSummary[lookupItem]]; exist {
				return VarAssigns[MethodSummary[lookupItem]]
			}
		}

		}
		return ProcessStringExpr(v.Variable, pat)
	case *expr.New:
		class, ok := v.Class.(*name.Name)
		if ok {
			return class.Parts[0].(*name.NamePart).Value // the class was mentioned explicitly
		}
		clsName, ok := v.Class.(*expr.Variable)
		if ok {
			rhs := clsName.VarName.(*node.Identifier).Value
			varmutex.Lock()
			_,exist := VarAssigns[rhs]
			varmutex.Unlock()
			if exist {
				return "(" + VarAssigns[rhs] + ")"
			} else {
				return ".*"
			}
		}

	}
	return ".*"
}


func containsCallback(n node.Node) bool{
	function := n.(*expr.FunctionCall)
	functionname, ok := function.Function.(*name.Name)
	parts := functionname.Parts
	lastNamePart, ok := parts[len(parts)-1].(*name.NamePart)
	if ok {
		if lastNamePart.Value == "call_user_func" || lastNamePart.Value == "call_user_func_array" ||
		   lastNamePart.Value == "preg_replace_callback" || lastNamePart.Value == "ldap_set_rebind_proc" ||
		   lastNamePart.Value == "mb_ereg_replace_callback" || lastNamePart.Value == "readline_completion_function" ||
		   lastNamePart.Value == "readline_callback_handler_install" || lastNamePart.Value == "header_register_callback" ||
		   lastNamePart.Value == "array_walk" || lastNamePart.Value == "array_walk_recursive" ||
		   lastNamePart.Value == "array_reduce" || lastNamePart.Value == "array_intersect_ukey" ||
		   lastNamePart.Value == "array_uintersect" || lastNamePart.Value == "array_uintersect_assoc" ||
		   lastNamePart.Value == "array_intersect_uassoc" || lastNamePart.Value == "array_uintersect_uassoc" ||
		   lastNamePart.Value == "array_diff_ukey" || lastNamePart.Value == "array_udiff" ||
		   lastNamePart.Value == "array_udiff_assoc" || lastNamePart.Value == "array_diff_uassoc" ||
		   lastNamePart.Value == "array_udiff_uassoc" || lastNamePart.Value == "array_filter" ||
		   lastNamePart.Value == "array_map" || lastNamePart.Value == "usort" || lastNamePart.Value == "uasort" ||
		   lastNamePart.Value == "register_shutdown_function" || lastNamePart.Value == "register_tick_function" ||
		   lastNamePart.Value == "set_error_handler" || lastNamePart.Value == "set_exception_handler" ||
		   lastNamePart.Value == "spl_autoload_register" {
			   return true
		   }

	}
	return false
}
// this function handle callbacks that are passed as an argument to a set of specific PHP functions such as call_user_func, array_map
func (d Dumper) handle_callbacks(n node.Node, abspath string, fname string, currentcls int) {
	function := n.(*expr.FunctionCall)
	functionname, ok  := function.Function.(*name.Name)
	len_args := len(function.ArgumentList.Arguments)
	parts := functionname.Parts
	lastNamePart, ok := parts[len(parts)-1].(*name.NamePart)
	Total_func_call += 1
	Dyn_func_call += 1
	if ok {
		//
		switch lastNamePart.Value {
		case "call_user_func", "call_user_func_array", "readline_completion_function", "header_register_callback", "array_map", "set_error_handler", "set_exception_handler", "register_shutdown_function", "register_tick_function", "spl_autoload_register":
			// first argument is a callback function
			arg , ok := function.ArgumentList.Arguments[0].(*node.Argument)
			l.Log(l.Info, "the callback stmt is %s", NodeSource(&n))
			if ok {
				tobecalled := d.GCProcessStringExpr(abspath, arg.Expr, currentcls)
				l.Log(l.Info, "callback arg %s:%s", tobecalled, NodeSource(&n))
				if tobecalled == "(.*)" || tobecalled == ".*"  || tobecalled == "(.*)\\(.*)" || tobecalled == "(.*)\\((.*))"{
					l.Log(l.Error, "callback function is not resolved %s:%s", NodeSource(&n), File)
					break
				}
				if tobecalled != "" {
					if strings.Contains(tobecalled, "|") {
						if s := strings.Split(tobecalled, "|"); len(s) == 2 {
							tobecalled = tobecalled + "|" + strings.ReplaceAll(tobecalled, "|", "\\")
						}
					}
					if contain_dotStar(tobecalled) {
						if infunc == true {
							UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = functionName + "|" + fname
						} else if inmeth == true {
							UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = methodName + "|" + fname
						} else{
							UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = "main|" + fname
						}
						l.Log(l.Critical, "%s (%s-->%s)", tobecalled, fname, d.Positions[n])
					}
					if infunc == true {
						CG[functionName+"|"+fname] = CG[functionName+"|"+fname] + "#" + tobecalled + "#" + strconv.Itoa(len_args-1)
					} else if inmeth == true {
						CG[methodName+"|"+fname] = CG[methodName+"|"+fname] + "#" + tobecalled+ "#" + strconv.Itoa(len_args-1)
					} else {
						CG["main"+"|"+fname] = CG["main"+"|"+fname] + "#" + tobecalled+ "#" + strconv.Itoa(len_args-1)
					}
					FunctionCalls[tobecalled] = true
					l.Log(l.Notice, "callback argument : (%s)", tobecalled)
					l.Log(l.Notice, "%s", NodeSource(&n))
				}
			}
			break
		case "array_walk", "array_walk_recursive", "array_reduce", "preg_replace_callback", "ldap_set_rebind_proc", "readline_callback_handler_install", "array_filter", "uasort", "usort":
			// second argument is a callback function
			l.Log(l.Info, "the callback stmt is %s", NodeSource(&n))
			if len(function.ArgumentList.Arguments) >= 2 {
				arg , ok := function.ArgumentList.Arguments[1].(*node.Argument)
				if ok {
					tobecalled := d.GCProcessStringExpr(abspath, arg.Expr, currentcls)
					l.Log(l.Info, "callback arg %s:%s", tobecalled, NodeSource(&n))
					if tobecalled == "(.*)" || tobecalled == ".*"  || tobecalled == "(.*)\\(.*)" || tobecalled == "(.*)\\((.*))"{
						l.Log(l.Error,"callback function is not resolved %s:%s", NodeSource(&n), File)
						break
					}
					if tobecalled != "" {
						if contain_dotStar(tobecalled) {
							if infunc == true {
								UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = functionName+ "|" + fname
							} else if inmeth == true {
								UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = methodName+ "|" + fname
							} else{
								UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = "main"+ "|" + fname
							}
							l.Log(l.Critical, "%s (%s-->%s)", tobecalled, fname, d.Positions[n])
						}
						if infunc == true {
							CG[functionName+"|"+fname] = CG[functionName+"|"+fname] + "#" + tobecalled+ "#" + strconv.Itoa(len_args-2)
						} else if inmeth == true {
						CG[methodName+"|"+fname] = CG[methodName+"|"+fname] + "#" + tobecalled+ "#" + strconv.Itoa(len_args-2)
							} else {
							CG["main"+"|"+fname] = CG["main"+"|"+fname] + "#" + tobecalled+ "#" + strconv.Itoa(len_args-2)
						}
						FunctionCalls[tobecalled] = true
						l.Log(l.Notice, "callback argument: (%s)", tobecalled)
						l.Log(l.Notice, "%s", NodeSource(&n))
					}
				}
			}
			break
		case "callback_key_compare_func", "array_uintersect", "array_intersect_uassoc", "array_diff_ukey", "array_udiff", "array_udiff_assoc", "array_diff_uassoc":
			// third argument is a callback function
			if len(function.ArgumentList.Arguments) > 3 {
				arg , ok := function.ArgumentList.Arguments[2].(*node.Argument)
				if ok {
					tobecalled := d.GCProcessStringExpr(abspath ,arg.Expr, currentcls)
					if tobecalled != "" {
						if contain_dotStar(tobecalled) {
							if infunc == true {
								UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = functionName + "|" + fname
							} else if inmeth == true {
								UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = methodName+ "|" + fname
							} else{
								UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = "main" + "|" + fname
							}
							l.Log(l.Critical, "%s (%s-->%s)", tobecalled, fname, d.Positions[n])
						}
						if infunc == true {
							CG[functionName+"|"+fname] = CG[functionName+"|"+fname] + "#" + tobecalled+ "#" + strconv.Itoa(len_args-1)
						} else if inmeth == true {
							CG[methodName+"|"+fname] = CG[methodName+"|"+fname] + "#" + tobecalled+ "#" + strconv.Itoa(len_args-1)
						} else {
							CG["main"+"|"+fname] = CG["main"+"|"+fname] + "#" + tobecalled+ "#" + strconv.Itoa(len_args-1)
						}
						FunctionCalls[tobecalled] = true
						l.Log(l.Notice, "callback argument: (%s)", tobecalled)
						l.Log(l.Notice, "%s", NodeSource(&n))
					}
				}
			}
			break
		case "array_uintersect_uassoc", "array_udiff_uassoc":
			// third and fourth argument is a callback function
			if len(function.ArgumentList.Arguments) > 3 {
				arg , ok := function.ArgumentList.Arguments[2].(*node.Argument)
				if ok {
					tobecalled := d.GCProcessStringExpr(abspath, arg.Expr, currentcls)
					if tobecalled != "" {
						if contain_dotStar(tobecalled) {
							if infunc == true {
								UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = functionName + "|" + fname
							} else if inmeth == true {
								UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = methodName + "|" + fname
							} else{
								UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = "main" + "|" + fname
							}
							l.Log(l.Critical, "%s (%s-->%s)", tobecalled, fname, d.Positions[n])
						}
						if infunc == true {
							CG[functionName+"|"+fname] = CG[functionName+"|"+fname] + "#" + tobecalled+ "#" + strconv.Itoa(len_args-1)
						} else if inmeth == true {
							CG[methodName+"|"+fname] = CG[methodName+"|"+fname] + "#" + tobecalled+ "#" + strconv.Itoa(len_args-1)
						} else {
							CG["main"+"|"+fname] = CG["main"+"|"+fname] + "#" + tobecalled+ "#" + strconv.Itoa(len_args-1)
						}
						FunctionCalls[tobecalled] = true
						l.Log(l.Notice, "callback argument: (%s)", tobecalled)
						l.Log(l.Notice, "%s", NodeSource(&n))
					}
				}
			}
			if len(function.ArgumentList.Arguments) > 4 {
				arg , ok := function.ArgumentList.Arguments[3].(*node.Argument)
				if ok {
					tobecalled := d.GCProcessStringExpr(abspath, arg.Expr, currentcls)
					if tobecalled != "" {
						if contain_dotStar(tobecalled) {
							if infunc == true {
								UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = functionName+ "|" + fname
							} else if inmeth == true {
								UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = methodName+ "|" + fname
							} else{
								UnresCalls[fname + ":"+ strconv.Itoa(d.Positions[n].StartLine)] = "main" + "|" + fname
							}
							l.Log(l.Critical, "%s (%s-->%s)", tobecalled, fname, d.Positions[n])
						}
						if infunc == true {
							CG[functionName+"|"+fname] = CG[functionName+"|"+fname] + "#" + tobecalled+ "#" + strconv.Itoa(len_args-1)
						} else if inmeth == true {
							CG[methodName+"|"+fname] = CG[methodName+"|"+fname] + "#" + tobecalled+ "#" + strconv.Itoa(len_args-1)
						} else {
							CG["main"+"|"+fname] = CG["main"+"|"+fname] + "#" + tobecalled+ "#" + strconv.Itoa(len_args-1)
						}
						FunctionCalls[tobecalled] = true
						l.Log(l.Notice, "callback argument: (%s)", tobecalled)
						l.Log(l.Notice, "%s", NodeSource(&n))
					}
				}
			}
			break
		default:
			break
		}
	}
}

func contain_dotStar(call string) bool{
	if strings.Contains(call, ".*"){
		if strings.Contains(call, "\\") {
			re := regexp.MustCompile("\\(*(\\.\\*)+\\)*\\\\\\(*(\\.\\*)+\\)*")
			if idx := re.FindStringIndex(call); len(idx) != 0 {
				return true;
			}
		} else {
			re := regexp.MustCompile("\\(*(\\.\\*)+\\)*")
			if idx := re.FindStringIndex(call); len(idx) != 0 {
				return true;
			}
		}
	}
	return false
}

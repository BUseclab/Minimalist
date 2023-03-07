package WPvisitor

import (
	"io"
	"reflect"
	"strings"
	"regexp"
	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/stmt"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/name"
	"github.com/z7zmey/php-parser/parser"
	"github.com/z7zmey/php-parser/walker"
	visit "php-cg/scan-project/visitor"
	l "php-cg/scan-project/logger"
)

type Dumper struct {
	Writer     io.Writer
	Indent     string
	Comments   parser.Comments
	Positions  parser.Positions
	NsResolver *NamespaceResolver
}


type ConstWalker struct {
	Writer     io.Writer
	Indent     string
	Comments   parser.Comments
	Positions  parser.Positions
}

var Actions = make (map[string]map[string]bool)
var infunc bool
var inmeth bool
var methodName string = "main"
var functionName string = "main"
var File string
var CsaCurcls = -1

func Resolve_include() {
	
	if strings.HasSuffix(File, "Scripts.class.php") {
		visit.Includes[".*(get_image.js|get_scripts.js|messages).php.*"] = append(visit.Includes[".*(get_image.js|get_scripts.js|messages).php.*"] , "include_once '' . $cfg['DefaultTabDatabase']")
	}

	if strings.HasSuffix(File, "db_create.php") {
		visit.Includes[".*db_create.php.*"] = append(visit.Includes[".*db_create.php.*"] , "include_once '' . $cfg['DefaultTabDatabase']")
		visit.Includes[".*(db_structure).php.*"] = append(visit.Includes[".*(db_structure).php.*"] , "include_once '' . $cfg['DefaultTabDatabase']")
	}

	if strings.HasSuffix(File, "import.php") {
		visit.Includes["import.php"] = append(visit.Includes["import.php"] , "include '' . $goto")
		visit.Includes[".*(tbl_import|db_import|server_import|tbl_structure|db_structure|server_sql).php.*"] = append(visit.Includes[".*(tbl_import|db_import|server_import|tbl_structure|db_structure|server_sql).php.*"] , "include '' . $goto")
	}

	if strings.HasSuffix(File, "index.php") {
		visit.Includes["index.php"] = append(visit.Includes["index.php"] , "include $_REQUEST['target']")
		visit.Includes["config.inc.php"] = append(visit.Includes["config.inc.php"] , "include $_REQUEST['target']")
		visit.Includes[".*(prefs_manage|db_datadict|db_structure|db_search|db_operation|db_sql|db_events|db_export|db_qbe|db_import|db_strcuture|db_routines|import|export|db_importdocsql|pdf_pages|pdf_schema|server_binlog|server_collations|server_databases|server_engines|server_export|server_import|server_privileges|server_sql|server_status|server_status_advisor|server_status_monitor|server_status_queries|server_status_variables|server_variables|sql|tbl_addfield|tbl_change|tbl_create|tbl_import|tbl_indexes|tbl_sql|tbl_export|tbl_operations|tbl_structure|tbl_relation|tbl_replace|tbl_row_action|tbl_select|tbl_zoom_select|transformation_overview|transformation_wrapper|user_password).php.*"] = append(visit.Includes[".*(prefs_manage|db_datadict|db|structure|db_search|db_operation|db_sql|db_events|db_export|db_qbe|db_import|db_strcuture|db_routines|import|export|db_importdocsql|pdf_pages|pdf_schema|server_binlog|server_collations|server_databases|server_engines|server_export|server_import|server_privileges|server_sql|server_status|server_status_advisor|server_status_monitor|server_status_queries|server_status_variables|server_variables|sql|tbl_addfield|tbl_change|tbl_create|tbl_import|tbl_indexes|tbl_sql|tbl_export|tbl_operations|tbl_structure|tbl_relation|tbl_replace|tbl_row_action|tbl_select|tbl_zoom_select|transformation_overview|transformation_wrapper|user_password).php.*"] , "include $_REQUEST['target']")
	}
	if strings.HasSuffix(File, "libraries/Config.class.php") {
		visit.Includes["libraries/Config.class.php"] = append(visit.Includes["libraries/Config.class.php"] , "include $this->default_source")
		visit.Includes[".*(libraries/config.default.php).*"] = append(visit.Includes[".*(libraries/config.default.php).*"] , "include $this->default_source")
		visit.Includes[".*(phpmyadmin.css.php).*"] = append(visit.Includes[".*(phpmyadmin.css.php).*"] , "include $this->default_source")
	}

	if strings.HasSuffix(File, "libraries/DisplayResults.class.php") {
		visit.Includes["libraries/DisplayResults.class.php"] = append(visit.Includes["libraries/DisplayResults.class.php"] , "include_once $include_file")
		visit.Includes["libraries/DisplayResults.class.php"] = append(visit.Includes["libraries/DisplayResults.class.php"] , "include_once $this->syntax_highlighting_column_info[strtolower($this->__get('db'))][strtolower($this->__get('table'))][strtolower($meta->name)][0]")
		visit.Includes[".*(libraries/plugins/transformations/.*.php).*"] = append(visit.Includes[".*(libraries/plugins/transformations/.*.php).*"] , "include_once $include_file")
		visit.Includes[".*(libraries/plugins/transformations/Text_Plain_Formatted.class.php).*"] = append(visit.Includes[".*(libraries/plugins/transformations/Text_Plain_Formatted.php).*"] , "include_once $this->syntax_highlighting_column_info[strtolower($this->__get('db'))][strtolower($this->__get('table'))][strtolower($meta->name)][0]")
	}

	if strings.HasSuffix(File, "libraries/StorageEngine.class.php") {
		visit.Includes["libraries/StorageEngine.class.php"] = append(visit.Includes["libraries/StorageEngine.class.php"] , "include_once $filename")
		visit.Includes[".*(libraries/engines/.*.lib.php).*"] = append(visit.Includes[".*(libraries/engines/.*.lib.php).*"] , "include_once $filename")
	}

	if strings.HasSuffix(File, "libraries/Theme.class.php") {
		visit.Includes["libraries/Theme.class.php"] = append(visit.Includes["libraries/Theme.class.php"] , "include $path")
		visit.Includes["libraries/Theme.class.php"] = append(visit.Includes["libraries/Theme.class.php"] , "include $fallback")
		visit.Includes[".*(themes/.*/css/.*.css.php).*"] = append(visit.Includes[".*(themes/.*/css/.*.css.php).*"] , "include $path")
		visit.Includes[".*(themes/.*/css/.*.css.php).*"] = append(visit.Includes[".*(themes/.*/css/.*.css.php).*"] , "include $fallback")
	}

	if strings.HasSuffix(File, "libraries/Theme_Manager.class.php") {
		visit.Includes["libraries/Theme_Manager.class.php"] = append(visit.Includes["libraries/Theme_Manager.class.php"] , "include $this->theme->getLayoutFile()")
		visit.Includes[".*(themes/.*/layout.inc.php).*"] = append(visit.Includes[".*(themes/.*/layout.inc.php).*"] , "include $this->theme->getLayoutFile()")
	}

	if strings.HasSuffix(File, "libraries/Util.class.php") {
		visit.Includes["libraries/Util.class.php"] = append(visit.Includes["libraries/Util.class.php"] , "include_once $escape[2]")
		visit.Includes[".*(libraries/plugins/export/.*.class.php).*"] = append(visit.Includes[".*(libraries/plugins/export/.*.class.php).*"] , "include_once $escape[2]")
	}

	if strings.HasSuffix(File, "libraries/common.inc.php") {
		visit.Includes["libraries/common.inc.php"] = append(visit.Includes["libraries/common.inc.php"] , "include $_SESSION['PMA_Theme']->getLayoutFile()")
		visit.Includes["libraries/common.inc.php"] = append(visit.Includes["libraries/common.inc.php"] , "include $__redirect")
		visit.Includes[".*(themes/.*/layout.inc.php).*"] = append(visit.Includes[".*(themes/.*/layout.inc.php).*"] , "include $_SESSION['PMA_Theme']->getLayoutFile()")
		visit.Includes[".*(db_datadict|db_sql|db_events|db_export|db_qbe|db_import|db_strcuture|db_routines|import|export|db_importdocsql|pdf_pages|pdf_schema|server_binlog|server_collations|server_databases|server_engines|server_export|server_import|server_privileges|server_sql|server_status|server_status_advisor|server_status_monitor|server_status_queries|server_status_variables|server_variables|sql|tbl_addfield|tbl_change|tbl_create|tbl_import|tbl_indexes|tbl_sql|tbl_export|tbl_operations|tbl_structure|tbl_relation|tbl_replace|tbl_row_action|tbl_select|tbl_zoom_select|transformation_overview|transformation_wrapper|user_password).php.*"] = append(visit.Includes[".*(db_datadict|db_sql|db_events|db_export|db_qbe|db_import|db_strcuture|db_routines|import|export|db_importdocsql|pdf_pages|pdf_schema|server_binlog|server_collations|server_databases|server_engines|server_export|server_import|server_privileges|server_sql|server_status|server_status_advisor|server_status_monitor|server_status_queries|server_status_variables|server_variables|sql|tbl_addfield|tbl_change|tbl_create|tbl_import|tbl_indexes|tbl_sql|tbl_export|tbl_operations|tbl_structure|tbl_relation|tbl_replace|tbl_row_action|tbl_select|tbl_zoom_select|transformation_overview|transformation_wrapper|user_password).php.*"] , "include $__redirect")
	}

	if strings.HasSuffix(File, "libraries/insert_edit.lib.php") {
		visit.Includes["libraries/insert_edit.lib.php"] = append(visit.Includes["libraries/insert_edit.lib.php"] , "include_once $include_file")
	        visit.Includes[".*(libraries/plugins/transformations/.*.php).*"] = append(visit.Includes[".*(libraries/plugins/transformations/.*.php).*"] , "include_once $include_file")
	}
	
	if strings.HasSuffix(File, "libraries/navigation/NodeFactory.class.php") {
		visit.Includes["libraries/navigation/NodeFactory.class.php"] = append(visit.Includes["libraries/navigation/NodeFactory.class.php"] , "include_once sprintf(self::$_path, $class)")
		visit.Includes[".*(libraries/navigation/Nodes/.*.class.php).*"] = append(visit.Includes[".*(libraries/navigation/Nodes/.*.class.php).*"] , "include_once sprintf(self::$_path, $class)")
	}

	if strings.HasSuffix(File, "libraries/plugin_interface.lib.php") {
		visit.Includes["libraries/plugin_interface.lib.php"] = append(visit.Includes["libraries/plugin_interface.lib.php"] , "include_once $plugins_dir . $file")
		visit.Includes[".(libraries/plugins/(import|export|schema)/.*).php.*"] = append(visit.Includes[".(libraries/plugins/(import|export|schema)/.*).php.*"] , "include_once $plugins_dir . $file")
	}

	if strings.HasSuffix(File, "libraries/plugins/auth/AuthenticationSignon.class.php") {
		visit.Includes["libraries/plugins/auth/AuthenticationSignon.class.php"] = append(visit.Includes["libraries/plugins/auth/AuthenticationSignon.class.php"] , "include $script_name")
		visit.Includes[".*(libraries/plugins/auth/AuthenticationSignon.class.php).*"] = append(visit.Includes[".*(libraries/plugins/auth/AuthenticationSignon.class.php).*"] , "include $script_name")
	}

	if strings.HasSuffix(File, "libraries/tcpdf/tcpdf.php") {
		visit.Includes["libraries/tcpdf/tcpdf.php"] = append(visit.Includes["libraries/tcpdf/tcpdf.php"] , "include $fontfile")
		visit.Includes[".*(libraries/tcpdf/fonts/.*.php).*"] = append(visit.Includes[".*(libraries/tcpdf/fonts/.*.php).*"] , "include $fontfile")
	}

	if strings.HasSuffix(File, "setup/config.php") {
		visit.Includes["setup/config.php"] = append(visit.Includes["setup/config.php"] , "include_once $config_file_path")
		visit.Includes[".*(config/config.inc.php).*"] = append(visit.Includes[".*(config/config.inc.php).*"] , "include_once $config_file_path")
	}
	if strings.HasSuffix(File, "sql.php") {
		visit.Includes["sql.php"] = append(visit.Includes["sql.php"] , "include '' . $goto")
		visit.Includes["sql.php"] = append(visit.Includes["sql.php"] , "include '' . PMA_securePath($goto)")
		visit.Includes[".*(sql|tbl_structure|db_sql|index|db_structure).php.*"] = append(visit.Includes[".*(sql|tbl_structure|db_sql|index|db_structure).php.*"] , "include '' . $goto")
		visit.Includes[".*(sql|tbl_structure|db_sql|index|db_structure).php.*"] = append(visit.Includes[".*(sql|tbl_structure|db_sql|index|db_structure).php.*"] , "include '' . PMA_securePath($goto)")
	}
	if strings.HasSuffix(File, "tbl_create.php") {
		visit.Includes["tbl_create.php"] = append(visit.Includes["tbl_create.php"] , "include '' . $cfg['DefaultTabTable']")
		visit.Includes[".*(db_structure).php.*"] = append(visit.Includes[".*(db_structure).php.*"] , "include '' . $cfg['DefaultTabTable']")
	}

	if strings.HasSuffix(File, "tbl_replace.php") {
		visit.Includes["tbl_replace.php"] = append(visit.Includes["tbl_replace.php"] , "include_once $filename")
		visit.Includes["tbl_replace.php"] = append(visit.Includes["tbl_replace.php"] , "include '' . PMA_securePath($goto_include)")
		visit.Includes["tbl_replace.php"] = append(visit.Includes["tbl_replace.php"] , "require '' . PMA_securePath($goto_include)")
		visit.Includes[".*(libraries/plugins/transformations/.*.php).*"] = append(visit.Includes[".*(libraries/plugins/transformations/.*.php).*"] , "include_once $filename")
		visit.Includes[".*(db_datadict|db_sql|db_events|db_export|db_qbe|db_import|db_strcuture|db_routines|import|export|db_importdocsql|pdf_pages|pdf_schema|server_binlog|server_collations|server_databases|server_engines|server_export|server_import|server_privileges|server_sql|server_status|server_status_advisor|server_status_monitor|server_status_queries|server_status_variables|server_variables|sql|tbl_addfield|tbl_change|tbl_create|tbl_import|tbl_indexes|tbl_sql|tbl_export|tbl_operations|tbl_structure|tbl_relation|tbl_replace|tbl_row_action|tbl_select|tbl_zoom_select|transformation_overview|transformation_wrapper|user_password).php.*"] = append(visit.Includes[".*(db_datadict|db_sql|db_events|db_export|db_qbe|db_import|db_strcuture|db_routines|import|export|db_importdocsql|pdf_pages|pdf_schema|server_binlog|server_collations|server_databases|server_engines|server_export|server_import|server_privileges|server_sql|server_status|server_status_advisor|server_status_monitor|server_status_queries|server_status_variables|server_variables|sql|tbl_addfield|tbl_change|tbl_create|tbl_import|tbl_indexes|tbl_sql|tbl_export|tbl_operations|tbl_structure|tbl_relation|tbl_replace|tbl_row_action|tbl_select|tbl_zoom_select|transformation_overview|transformation_wrapper|user_password).php.*"] , "include '' . PMA_securePath($goto_include)")
		visit.Includes[".*(db_datadict|db_sql|db_events|db_export|db_qbe|db_import|db_strcuture|db_routines|import|export|db_importdocsql|pdf_pages|pdf_schema|server_binlog|server_collations|server_databases|server_engines|server_export|server_import|server_privileges|server_sql|server_status|server_status_advisor|server_status_monitor|server_status_queries|server_status_variables|server_variables|sql|tbl_addfield|tbl_change|tbl_create|tbl_import|tbl_indexes|tbl_sql|tbl_export|tbl_operations|tbl_structure|tbl_relation|tbl_replace|tbl_row_action|tbl_select|tbl_zoom_select|transformation_overview|transformation_wrapper|user_password).php.*"] = append(visit.Includes[".*(db_datadict|db_sql|db_events|db_export|db_qbe|db_import|db_strcuture|db_routines|import|export|db_importdocsql|pdf_pages|pdf_schema|server_binlog|server_collations|server_databases|server_engines|server_export|server_import|server_privileges|server_sql|server_status|server_status_advisor|server_status_monitor|server_status_queries|server_status_variables|server_variables|sql|tbl_addfield|tbl_change|tbl_create|tbl_import|tbl_indexes|tbl_sql|tbl_export|tbl_operations|tbl_structure|tbl_relation|tbl_replace|tbl_row_action|tbl_select|tbl_zoom_select|transformation_overview|transformation_wrapper|user_password).php.*"] , "require '' . PMA_securePath($goto_include)")
	}
	if strings.HasSuffix(File, "view_create.php") {
		visit.Includes["view_create.php"] = append(visit.Includes["view_create.php"] , "include './' . $cfg['DefaultTabDatabase']")
		visit.Includes[".*(db_structure).php.*"] = append(visit.Includes[".*(db_structure).php.*"] , "include './' . $cfg['DefaultTabDatabase']")
	}

	return
}

func clean_cg(method string) {
	callString := visit.CG[method]
	calls := strings.Split(callString, "#")
	newCalls := ""
	for indx := 1; indx < len(calls)-1 ; indx += 2 {
		call := calls[indx]
		if call == ".*" || call == "" {
			newCalls = newCalls
		} else if strings.Contains(call, ".*"){
			for true {
				if strings.HasPrefix(call, ".*|") {
					call = call[len(".*|"):]
				} else {
					break
				}
			}
			if call == "(.*)" || call == "((.*))\\(.*)"  || call == "(.*)\\(.*)"{
				call = ""
			}
			for true {
				if strings.HasSuffix(call, "|.*") {
					call = call[:len(call)-len("|.*")]
				} else {
					break
				}
			}
			for true {
				re := regexp.MustCompile(`\((\.\*)+\|`)
				if idx := re.FindStringIndex(call); len(idx) != 0 {
					call = call[:idx[0]] + "(" + call[idx[1]:]
				} else {
					break
				}
			}

			for true {
				re := regexp.MustCompile(`\|(\.\*)+\)`)
				if idx := re.FindStringIndex(call); len(idx) != 0 {
					call = call[:idx[0]] + ")" + call[idx[1]:]
				} else {
					break
				}
			}
			for true {
				re := regexp.MustCompile(`\|(\.\*)+\|`)
				if idx := re.FindStringIndex(call); len(idx) != 0 {
					call = call[:idx[0]] + "|" + call[idx[1]:]

				} else {
					break
				}
			}
			if strings.HasSuffix(call, "|") {
				call = call[:len(call)-1]
			} else if strings.HasPrefix(call, "|") {
				call = call[1:len(call)]
			}
			if call != ".*" && call != ""{
				newCalls = newCalls + "#" + call + "#" + calls[indx+1]
			} else {
				newCalls = newCalls
			}
		} else {
			newCalls = newCalls + "#" + call + "#" + calls[indx+1]
		}
	}
	visit.CG[method] = newCalls
}

func (d Dumper) EnterNode(w walker.Walkable) bool {
	n := w.(node.Node)
	File = visit.File
	switch reflect.TypeOf(n).String() {
	
	case "*stmt.Class":
		CsaCurcls += 1
		class := n.(*stmt.Class)
		if namespacedName, ok := d.NsResolver.ResolvedNames[class]; ok && visit.NsEnable {
			visit.CName = namespacedName
		} else {
			className , ok := class.ClassName.(*node.Identifier)
			if !ok{
				l.Log(l.Error,"couldn't resolve classname:%s", visit.NodeSource(&n))
				break
			}
			visit.CName = className.Value
		}


	case "*expr.StaticCall":
		sCallRoot := n.(*expr.StaticCall)
		class := n.(*expr.StaticCall).Class
		call := n.(*expr.StaticCall).Call
		clsname := ""
		staticname := ""
		if _, ok := class.(*expr.Variable); ok {
			clsname = visit.LocalProcessStringExpr(visit.File, class, 1, CsaCurcls)
			}
		className, ok1 := class.(*name.Name)
		fqnClassName, ok2 := class.(*name.FullyQualified)
		classIdentifier, ok3 := class.(*node.Identifier)
		if ok1 {
			clsname = className.Parts[len(className.Parts)-1].(*name.NamePart).Value
		} else if ok2 {
			clsname = fqnClassName.Parts[len(fqnClassName.Parts)-1].(*name.NamePart).Value
		} else if ok3 {
			clsname = classIdentifier.Value
		}
		if clsname == "self" {
			clsname = visit.CName
		}
		staticname = visit.LocalProcessStringExpr(visit.File, call, 1, CsaCurcls)
		sCall := clsname + "\\" + staticname
		if sCall == "PMA_NodeFactory\\getInstance" {
			// take a look at the argument
			// and add it to the visit.CG
			if len(sCallRoot.ArgumentList.Arguments) < 1 {
				break
			}
			arg, ok := sCallRoot.ArgumentList.Arguments[0].(*node.Argument)
			if ok {
				tobecalled := visit.LocalProcessStringExpr(visit.File, arg.Expr, 1, CsaCurcls)
				if tobecalled != ".*" {
					visit.CG["PMA_NodeFactory\\getInstance|libraries/navigation/NavigationTree.class.php"] += "#" + tobecalled + "\\__construct#3"
				}
			}
		}
		break
	
	case "*stmt.ClassMethod":
		classMethod := n.(*stmt.ClassMethod)
		mName, ok := classMethod.MethodName.(*node.Identifier)
		inmeth = true
		if ok{
			if namespacedName, ok :=d.NsResolver.ResolvedNames[classMethod]; ok && visit.NsEnable {
				methodName = namespacedName
			} else {
				methodName = visit.CName + "\\" + mName.Value
			}
		}
		switch (methodName) {
		case "ShapeFile\\_openDBFFile":
			clean_cg("ShapeFile\\_openDBFFile|libraries/bfShapeFiles/ShapeFile.lib.php")
			break
		case " PMA_DisplayResults\\_getRowValues":
			visit.CG["PMA_DisplayResults\\_getRowValues|libraries/DisplayResults.class.php"] += "#" + "(Image|Text_Plain)_.*\\__construct#1"
			break

		case "PMA_Util\\pow":
			clean_cg("PMA_Util\\pow|libraries/Util.class.php")
			break
		case "ExportCodegen\\exportData":
			clean_cg("ExportCodegen\\exportData|libraries/plugins/export/ExportCodegen.class.php")
			break

		case "PMA_DisplayResults\\_getDataCellForNonNumericAndNonBlobColumns":
			clean_cg("PMA_DisplayResults\\_getDataCellForNonNumericAndNonBlobColumns|libraries/DisplayResults.class.php")
			break
		case "TCPDF\\Image":
			clean_cg("TCPDF\\Image|libraries/tcpdf/tcpdf.php")
			break
		case "PMA_Util\\expandUserString":
			clean_cg("PMA_Util\\expandUserString|libraries/Util.class.php")
			visit.CG["PMA_Util\\expandUserString|libraries/Util.class.php"] += "#ExportLatex\\__construct#0#ExportLatex\\texEscape#1"
			break
		}
	case "*stmt.Function":
		function := n.(*stmt.Function)
		funcName, ok := function.FunctionName.(*node.Identifier)
		infunc = true
		if ok{
			if namespacedName, ok :=d.NsResolver.ResolvedNames[function]; ok && visit.NsEnable {
				functionName = namespacedName
			} else {
				functionName = funcName.Value
			}
		}
		switch (functionName) {

		case "_get_reader":
			visit.CG["_get_reader|libraries/php-gettext/gettext.inc"] += "#gettext_reader\\gettext_reader#2"
			break
		case "PMA_DBI_fetch_value":
			clean_cg("PMA_DBI_fetch_value|libraries/database_interface.lib.php")
			break
		case "PMA_DBI_fetch_result":
			clean_cg("PMA_DBI_fetch_result|libraries/database_interface.lib.php")
			break
		case "PMA_DBI_fetch_single_row":
			clean_cg("PMA_DBI_fetch_single_row|libraries/database_interface.lib.php")
			break
		case "PMA_getTransformationDescription":
			visit.CG["PMA_getTransformationDescription|libraries/transformations.lib.php"] += "#" + ".*_getInfo#1"
			clean_cg("PMA_getTransformationDescription|libraries/transformations.lib.php")
			break
		case "PMA_usort_comparison_callback":
			clean_cg("PMA_usort_comparison_callback|libraries/database_interface.lib.php")
			break
		case "PMA_arrayWalkRecursive":
			clean_cg("PMA_arrayWalkRecursive|libraries/core.lib.php")
			break
		case "PMA_buildHtmlForDb":
			clean_cg("PMA_buildHtmlForDb|libraries/build_html_for_db.lib.php")
			visit.CG["PMA_buildHtmlForDb|libraries/build_html_for_db.lib.php"] += "#" + "PMA_getCollationDescr#1"
			break
		case "PMA_getPlugins":
			visit.CG["PMA_getPlugins|libraries/plugin_interface.lib.php"] += "#" + "(Export|Import).*\\__construct#0"
			break

		case "PMA_getPlugin":
			visit.CG["PMA_getPlugin|libraries/plugin_interface.lib.php"] += "#" + "(Export|Import).*\\__construct#0"
			break

		case "PMA_transformEditedValues":
			visit.CG["PMA_transformEditedValues|libraries/insert_edit.lib.php"  ] += "#" + "(Image|Text_Plain)_*\\__construct#1"
			break

		case "PMA_config_validate":
			clean_cg("PMA_config_validate|libraries/config/validate.lib.php")
			if _, exist := visit.VarTrack["libraries/config.values.php"]["cfg_db*_userValidators"]; exist {
				visit.CG["PMA_config_validate|libraries/config/validate.lib.php"] = visit.CG["PMA_config_validate|libraries/config/validate.lib.php"] + "#" + visit.VarTrack["libraries/config.values.php"]["cfg_db*_userValidators"] + "#2"
			}
			if _, exist := visit.VarTrack["libraries/config.values.php"]["cfg_db*_validators"]; exist {
				visit.CG["PMA_config_validate|libraries/config/validate.lib.php"] = visit.CG["PMA_config_validate|libraries/config/validate.lib.php"] + "#" + visit.VarTrack["libraries/config.values.php"]["cfg_db*_validators"] + "#2"
			}
			break

		}
	}
		switch visit.RelativePath {
		case "libraries/common.inc.php":
			visit.CG["main|libraries/common.inc.php"] += "#PMA\\__get#1#PMA\\__set#1"
			break
		case "libraries/display_import_ajax.lib.php":
			visit.CG["main|libraries/display_import_ajax.lib.php"] += "#" + "PMA_import_.*Check#1"
			clean_cg("main|libraries/display_import_ajax.lib.php")
			break
		case "server_databases.php":
			clean_cg("main|server_databases.php")
			break
		}
	return true
}

func (d Dumper) GetChildrenVisitor(key string) walker.Visitor {
	return Dumper{d.Writer, d.Indent + "    ", d.Comments, d.Positions, d.NsResolver}
}

func (d Dumper) LeaveNode(w walker.Walkable) {
	n := w.(node.Node)
	switch reflect.TypeOf(n).String() {
	case "*stmt.Class":
		visit.CName = ""
	case "*stmt.Function":
		infunc = false
		functionName = "main"
	case "*stmt.ClassMethod":
		inmeth = false
		methodName = "main"
	}
}

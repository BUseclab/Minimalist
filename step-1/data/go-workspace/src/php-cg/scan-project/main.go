package main

import (
	progressbar "github.com/schollz/progressbar"
	"sort"
	"crypto/md5"
	"bytes"
	"fmt"
	"sync"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"php-cg/db"
	"github.com/z7zmey/php-parser/php7"
	"github.com/z7zmey/php-parser/node"
	"regexp"
	"strings"
	"php-cg/scan-project/visitor/include_string"
	l "php-cg/scan-project/logger"
	visit "php-cg/scan-project/visitor"
	// csavisit can be any of the custom static analysis in csa_visitor folder
	// depending on what web app you are analyzing
	csavisit "php-cg/scan-project/csa_visitor/phpmyadmin/4.0"

)

var numIncludes int = 0
var numStaticIncludes int = 0
var numResolvedIncludes int = 0
var numDynamicIncludes int = 0
var numSemiDynamicIncludes int = 0

var fqnClassDefinitions = make(map[string][]string)
var classDefinitions = make(map[string][]string)
var classInstances = make(map[string][]string)
var fileList []string
var Constants = make(map[string]includestring.StringTrie)



var constmutex = &sync.Mutex{}

func printSet(m *map[string]bool) {
	for k, _ := range *m {
		fmt.Printf("%s\n", k)
	}
}

func getScriptDependency(Db *db.DB,file string, basepath string ) {
	relativepath := strings.TrimPrefix(file, basepath)
	for _, incFile := range Db.GetIncludesOfFile(relativepath) {
		if incFile != "" && incFile != relativepath{
			visit.ScriptDep[relativepath] = append(visit.ScriptDep[relativepath], incFile)
		}
	}
}

func recordResult(path string, basepath string, tx *db.Tx) {
	relativePath := strings.TrimPrefix(path, basepath)
	err := tx.CreateFile(relativePath)
	if err != nil {
		l.Log(l.Warning, "Bad File statement")
	}

	for k, _ := range visit.ClassDefinitions {
		name := strings.Split(k, `\`)
		shortname := name[len(name)-1]
		err = tx.CreateClass(k, shortname, relativePath)
		if err != nil {
			l.Log(l.Warning, "Bad ClassDefinitions statement", k)

		}
		classDefinitions[k] = append(classDefinitions[k], relativePath)
		fqnClassDefinitions[k] = append(classDefinitions[k], relativePath)
	}


	for k, _ := range visit.FunctionCalls {
		if classname := strings.Split(k, `::`); len(classname) > 1 {
			tx.CreateClassInstance(relativePath, classname[0], classname[0])
			classInstances[classname[0]] = append(classInstances[classname[0]], relativePath)
		}
		name := strings.Split(k, `\`)
		shortname := name[len(name)-1]
		err = tx.CreateFunctionCall(relativePath, shortname)
		if err != nil {
			l.Log(l.Warning, "Bad FunctionCall statement", k)
		}
	}

	for k, v := range visit.Includes {
		if !strings.ContainsAny(k, visit.Alpha) {
			l.Log(l.Notice, "(%s) contained visit.Alpha\n", k)
			for _, includeString := range v {
				err = tx.CreateInclude(relativePath, relativePath, 0, 0, includeString)
			}
			continue
		}
		// TODO: Deal with "//"
		k := strings.Replace(k, "//", "/", -1)    // No good.
		k = strings.Replace(k, ".*./", ".*/", -1) // No good.
		k = strings.Replace(k, "(./", "(/", -1)   // No good.
		k = strings.Replace(k, "|./", "|/", -1)   // No good.
		k = strings.Replace(k, basepath, "", -1)  // No good.
		count := 0
		regex := regexp.MustCompile(k)
		for _, p := range fileList {
			trimmed := strings.TrimPrefix(p, basepath)
			if regex.MatchString(trimmed) {
				l.Log(l.Notice, "Match: %s -> %s", k, trimmed)
				for _, includeString := range v {
					err = tx.CreateInclude(relativePath, trimmed, count, 0, includeString)
				}
				count++
			}
		}
		if count == 0 {
			l.Log(l.Notice, "No Dice: %s", k)

		} else if count > 1 {
			l.Log(l.Notice, "Multiple matches: %s", k)

		}
	}

	for k, _ := range visit.ClassInstances {
		name := strings.Split(k, `\`)
		shortname := name[len(name)-1]
		err = tx.CreateClassInstance(relativePath, k, shortname)
		if err != nil {
			panic(err)

		}
		classInstances[k] = append(classInstances[k], relativePath)
	}

}


func method_def_analysis(path string, basepath string) {

	fileContents, _ := ioutil.ReadFile(path)
	parser := php7.NewParser(bytes.NewBufferString(string(fileContents)), path)
	parser.Parse()

	rootNode := parser.GetRootNode()

	nsresolver := visit.NewNamespaceResolver()
	rootNode.Walk(nsresolver)
	defvisit := visit.DefWalker {
		Writer: os.Stdout,
		Indent: "",
		Positions: parser.GetPositions(),
		NsResolver: nsresolver,
	}

	// enable this for drupal and joomla 
//	csadefvisit := csavisit.DefWalker {
//		Writer: os.Stdout,
//		Indent: "",
//		Positions: parser.GetPositions(),
//	}
	//end
	_ = defvisit
	visit.RelativePath = strings.TrimPrefix(path, basepath)
	visit.File = strings.TrimPrefix(path, basepath)
	visit.File = path
	rootNode.Walk(defvisit)
	//enable this for drupal and joomla
//	rootNode.Walk(csadefvisit)
	//end
}


func preprocessIncludes(path string, basepath string) {
	fileContents, _ := ioutil.ReadFile(path)
	parser := php7.NewParser(bytes.NewBufferString(string(fileContents)), path)
	parser.Parse()

	rootNode := parser.GetRootNode()

	visitor := visit.ConstWalker{
		Writer: os.Stdout,
		Indent: "",
	}
	_ = visitor

	Constants["DIRECTORY_SEPARATOR"] = includestring.StringTrie{Content: "/"}
	visit.Constants = &Constants
	visit.ClearAssigns()
	visit.File = path //strings.TrimPrefix(path, basepath)

	rootNode.Walk(visitor)

}

func processIncludes(path string, basepath string, tx *db.Tx) {
	fileContents, _ := ioutil.ReadFile(path)
	parser := php7.NewParser(bytes.NewBufferString(string(fileContents)), path)

	parser.Parse()

	rootNode := parser.GetRootNode()

	nsresolver := visit.NewNamespaceResolver()
	rootNode.Walk(nsresolver)

	visitor := visit.IncludeWalker{
		Writer:  os.Stdout,
		Indent:  "",
		NsResolver: nsresolver,
	}


	visit.Includes = make(map[string][]string)
	visit.StaticIncludes = make(map[string]bool)
	visit.DynamicIncludes = make(map[string]bool)
	visit.SemiDynamicIncludes = make(map[string]bool)
	visit.ClassInstances = make(map[string]bool)
	visit.ClassDefinitions = make(map[string]bool)
	visit.ClearAssigns()
	visit.NumIncludes = 0
	visit.File = path
	//for all csa
	csavisit.File = path
	//end
	visit.FunctionCalls = make(map[string]bool)
	visit.MethodCalls = make(map[string]bool)
	Constants["DIRECTORY_SEPARATOR"] = includestring.StringTrie{Content: "/"}
	visit.Constants = &Constants

	rootNode.Walk(visitor)
	//for all csa
	csavisit.Resolve_include()
	//end
	recordResult(path, basepath, tx)
}


func preprocessOpt( path string, basepath string ) {
	fileContents, _ := ioutil.ReadFile(path)
	parser := php7.NewParser(bytes.NewBufferString(string(fileContents)), path)
	parser.Parse()

	rootNode := parser.GetRootNode()
	nsresolver := visit.NewNamespaceResolver()
	rootNode.Walk(nsresolver)
	
	visitor := visit.TrackWalker {
		Writer: os.Stdout,
		Indent: "",
		NsResolver: nsresolver,
	}

	visit.File = path
	rootNode.Walk(visitor)
}
func processDelayedvars(path string,nodes [] node.Node) {
	visit.File = path
	visit.ClearScopes()
	nsresolver := visit.NewNamespaceResolver()
	for _, t := range nodes {
		t.Walk(nsresolver)
	}

	varvisit := visit.VarWalker {
		Writer: os.Stdout,
		Indent: "",
		NsResolver: nsresolver,
	}
	visit.No_more_var = false
	visit.Currentcls = 0 
	for _,t := range nodes {
		t.Walk(varvisit)
	}
}

func chainedFixvars(fixedItem visit.Vd) {
	for _, item := range visit.LocalWaitingQueue[fixedItem.Src_file] {
		if item.Src_var == fixedItem.Fix_var {
			for _, t:= range visit.WaitingQueue[fixedItem.Src_file] {
				if t.Fix_var == item.Dst_var {
					processDelayedvars(t.Src_file, t.Asgn_node)
						tmp := [] visit.Vd{}
						for _, noderm := range visit.WaitingQueue[fixedItem.Src_file] {
							if noderm.Src_file  == t.Src_file && noderm.Fix_var == t.Fix_var && visit.GetHashStmts(t.Asgn_node) == visit.GetHashStmts(noderm.Asgn_node){
								continue
							} else {
								tmp = append(tmp, noderm)
							}
						}
						visit.WaitingQueue[fixedItem.Src_file] = tmp
						chainedFixvars(t)

				}
			}
		}
	}

}


func computeDelayvariables(path string){
	for _, item := range visit.WaitingQueue[path] {
		processDelayedvars(item.Src_file, item.Asgn_node)
	}

}

func preprocessFile(path string, basepath string) {
	l.Log(l.Info, "Preprocessing file: %s", path)
	fileContents, _ := ioutil.ReadFile(path)
	parser := php7.NewParser(bytes.NewBufferString(string(fileContents)), path)
	parser.Parse()

	rootNode := parser.GetRootNode()

	nsresolver := visit.NewNamespaceResolver()
	rootNode.Walk(nsresolver)


	varvisit := visit.VarWalker {
		Writer: os.Stdout,
		Indent: "",
		NsResolver: nsresolver,
	}

	//enable this only for joomla
//	cnsresolver := csavisit.NewNamespaceResolver()
//	rootNode.Walk(cnsresolver)
//	csavarvisit := csavisit.VarWalker {
//		Writer: os.Stdout,
//		Indent: "",
//		NsResolver: cnsresolver,
//	}
	// end
	visit.File = path //strings.TrimPrefix(path, basepath)
	visit.ClearScopes()
	visit.ClearGlobalArgs()
	visit.No_more_var = false
	visit.Currentcls = -1
	rootNode.Walk(varvisit)
	//enable this for joomla
//	rootNode.Walk(csavarvisit)
	// end

	for it, val := range visit.VarTrack[visit.File] {
		for strings.Contains(val, "||") {
			val = strings.ReplaceAll(val, "||", "|")
			visit.VarTrack[visit.File][it] = val
		}
	}
	for it, val := range visit.VarTrack[visit.File] {
		for strings.Contains(val, "|)") {
			val = strings.ReplaceAll(val, "|)", ")")
			visit.VarTrack[visit.File][it] = val
		}
	}
	for it, val := range visit.VarTrack[visit.File] {
		for strings.Contains(val, "(|") {
			val = strings.ReplaceAll(val, "(|", "(")
			visit.VarTrack[visit.File][it] = val
		}
	}
	for it, val := range visit.VarTrack[visit.File] {
		if strings.HasPrefix(val, "|") {
			visit.VarTrack[visit.File][it] = val[1:]

		}
		if strings.HasSuffix(val, "|") {
			visit.VarTrack[visit.File][it] = val[:len(val)-1]
		}
	}

	if _,exist := visit.WaitingQueue[path]; exist {
		for _, v := range visit.WaitingQueue[path] {
			for _, item := range visit.LocalWaitingQueue[v.Src_file] {
				if item.Dst_var == v.Fix_var && item.Src_var == "" {
					before_hash := md5.Sum([] byte(visit.VarTrack[v.Src_file][v.Fix_var]))
					processDelayedvars(v.Src_file, v.Asgn_node)
					after_hash := md5.Sum([] byte(visit.VarTrack[v.Src_file][v.Fix_var]))
					if after_hash != before_hash {
						tmp := [] visit.Vd{}
						for _, noderm := range visit.WaitingQueue[path] {
							if noderm.Src_file  == v.Src_file && noderm.Fix_var == v.Fix_var{
								continue
							} else {
								tmp = append(tmp, noderm)
							}
						}
						visit.WaitingQueue[path] = tmp
						chainedFixvars(v)
					}
				}
			}
		}
	}
	
}


func processFile(path string, basepath string, tx *db.Tx) {
	fileContents, _ := ioutil.ReadFile(path)
	parser := php7.NewParser(bytes.NewBufferString(string(fileContents)), path)

	parser.Parse()

	rootNode := parser.GetRootNode()

	nsresolver := visit.NewNamespaceResolver()
	rootNode.Walk(nsresolver)
	visitor := visit.Dumper{
		Writer:  os.Stdout,
		Indent:  "",
		Positions: parser.GetPositions(),
		NsResolver: nsresolver,
	}

	// for all csa
	nsResolver := csavisit.NewNamespaceResolver()
	rootNode.Walk(nsResolver)
	csavisitor := csavisit.Dumper{
		Writer:     os.Stdout,
		Indent:     "",
		NsResolver: nsResolver,
	}
	//end	
	visit.FunctionCalls = make(map[string]bool)
	visit.MethodCalls = make(map[string]bool)
	visit.File = path //strings.TrimPrefix(path, basepath)
	visit.RelativePath = strings.TrimPrefix(path, basepath)
	visit.Curcls = -1
	rootNode.Walk(visitor)
	// for all csa
	csavisit.CsaCurcls = -1
	rootNode.Walk(csavisitor)
	//end
	recordResult(path, basepath, tx)
}



func preprocessWorker(id int, jobs<-chan string, project_path string) {
	for j := range jobs{
		preprocessFile(j, project_path)
	}
	l.Log(l.Info,"finished running worker %d",id)
}

func main() {
	l.Level = l.Debug
	project_path := os.Args[1]

	visit.VarAssigns = make(map[string]string)
	visit.MethodSummary = make(map[string]string)

	err := filepath.Walk(project_path, func(path string, f os.FileInfo, err error) error {
		if filepath.Ext(path) == ".php" ||
			filepath.Ext(path) == ".engine" ||
			filepath.Ext(path) == ".phtml" ||
			filepath.Ext(path) == ".html" ||
			filepath.Ext(path) == ".module" ||
			filepath.Ext(path) == ".theme" ||
			filepath.Ext(path) == ".inc" {
			fileList = append(fileList, path)
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	Db, err := db.OpenDb(os.Args[2])
	if err != nil {
		panic(err)
	}

	visit.UsedNamespaceSummary = make(map[string][]string)
	visit.ImplementedIntface = make(map[string][]string)
	visit.NamespaceSummary = make(map[string]string)
	bar := progressbar.Default(int64(len(fileList)), "analyze files for methods/function definitions")
	sort.Strings(fileList)
	for _, file := range fileList {
		method_def_analysis(file, project_path)
		bar.Add(1)
	}
	// write functions to a file
	tx, err := Db.Begin()
	if err != nil {
		fmt.Printf("%s\n", err)
	}
	f, err := os.Create("functions.txt")
	if err == nil {
		for funcs := range visit.Functions {
			err = tx.CreateFunction(funcs, visit.Functions[funcs][0],visit.Functions[funcs][1] ,visit.Functions[funcs][2] )
			if err != nil {
				fmt.Printf("%s\n", err)
			}
			fmt.Fprintf(f,"%s\n",funcs)
		}
	}
	f.Close()

	// write methods to a file
	f, err = os.Create("methods.txt")
	if err == nil {
		for meths := range visit.Methods {
			err = tx.CreateFunction(meths, visit.Methods[meths][0], visit.Methods[meths][1], visit.Methods[meths][2])
			if err != nil {
				fmt.Printf("%s\n", err)
			}
			fmt.Fprintf(f,"%s\n",meths)
		}
	}
	f.Close()
	err = tx.Commit()


	
	visit.Variable_dependency = make(map[string][]string)

	if visit.Improved_vartrack {
		for _,file := range fileList {
			preprocessOpt(file, project_path)
		}
		for _,val := range visit.MethodSummary {
			if !strings.Contains(val, "#") {
				visit.Tracked_variables = append(visit.Tracked_variables, val)
			}
		}
		tmp := visit.Tracked_variables
		for len(tmp) > 0 {
			v := tmp[0]
			for _, item := range(visit.Variable_dependency[v]) {
				if !visit.Contains(visit.Tracked_variables, item) {
					visit.Tracked_variables = append(visit.Tracked_variables, item)
					tmp = append(tmp, item)
				}
			}
			tmp = tmp[1:]
		}
	}

	tx, err = Db.Begin()

	for _, file := range fileList {
		preprocessIncludes(file, project_path)
	}

	bar = progressbar.Default(int64(len(fileList)), "Analyze script inclusion")
	for _, file := range fileList {
		processIncludes(file, project_path, tx)
		bar.Add(1)
	}
	err = tx.Commit()
	ResolveClassInstances(Db, &fqnClassDefinitions, &classDefinitions, &classInstances)
		
	var fileQ []string
	var waitedfiles []string
	for _, file := range fileList {
		relpath := strings.TrimPrefix(file, project_path)
		if _, exist := visit.ScriptDep[relpath]; !exist {
			fileQ = append(fileQ, file)
		} else {
			waitedfiles = append(waitedfiles, file)
		}
	}
	timer := 0
	for len(waitedfiles) > 0 {
		wf := strings.TrimPrefix(waitedfiles[0], project_path)
		safe := true
		for _, uns := range visit.ScriptDep[wf]{
			if !visit.Contains(fileQ, project_path + uns) {
				safe = false
				break
			}
		}
		if safe {
			fileQ = append(fileQ, project_path + wf)
			waitedfiles = waitedfiles[1:]
			timer = 0
		} else {
			timer += 1
			waitedfiles = waitedfiles[1:]
			waitedfiles = append(waitedfiles,project_path + wf)
		}
		if timer/10 > len(waitedfiles) {
			fmt.Printf("stuck in a loop\n")
			for _, file := range waitedfiles {
				wf := strings.TrimPrefix(file, project_path)
				fmt.Printf("the file (%s) didn't have these constraints\n", file)
				for _, f := range visit.ScriptDep[wf] {
					if !visit.Contains(fileQ, project_path + f) {
						fmt.Printf("file: %s\n", f)
					}
				}
				fileQ = append(fileQ, project_path + file)
			}
			break
		}
	}
	

	tx, err = Db.Begin()

	bar = progressbar.Default(int64(len(fileList)), "Analyze Variables")
	for _, file := range fileList {
		preprocessFile(file, project_path)
		bar.Add(1)
	}


//	visit.MethodSummary = make(map[string]string)

	visit.CG = make(map[string]string)
	bar = progressbar.Default(int64(len(fileList)), "Generate call-graph entries")
	for _, file := range fileList {
		processFile(file, project_path, tx)
		bar.Add(1)
	}
	err = tx.Commit()

	f, err = os.Create("calls.txt")
	if err == nil {
		keys := make([]string, 0, len(visit.CG))
		for k := range visit.CG {
			keys = append(keys,k)
		}
		sort.Strings(keys)
		for _, caller := range keys{
			fmt.Fprintf(f, "%s->%s\n",caller,visit.CG[caller])
		}
	}
	f.Close()

	f, err = os.Create("unresolved.txt")
	if err == nil {
		for fl,fn := range visit.UnresCalls {
			fmt.Fprintf(f, "{%s}{%s}\n", fl, fn)
		}
	}
	f.Close()
	fmt.Printf("total number of calls: %d\nDynamic calls: %d\nLiteral calls: %d\n",visit.Total_func_call, visit.Dyn_func_call, visit.Static_func_call)
}

#!/usr/bin/python3
from concurrent.futures import ThreadPoolExecutor
import sys, os, sqlite3
import re, regex
from collections import OrderedDict
import json
import logging
import getopt
import multiprocessing as mp
from functools import partial
from tqdm import tqdm
import copy



IMPROVE_CANDID_SELECTION = True

unhandled = set()


alpha = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
def checkforalpha(string):
    for i in string:
        if i.isalpha():
            return True

    return False



def db_connect(path):
    con = sqlite3.connect(path)
    con.text_factory = str
    return con

def get_params(con, fName):
    if fName =="":
        return (0,0)
    query = """
    SELECT len_param, req_param
    FROM function
    WHERE
        name == ?
    """
    cursor = con.execute(query, [fName])
    data = cursor.fetchall()
    len_param = int(data[0][0])
    opt_param = int(data[0][1])
    req_param = len_param - opt_param
    return (len_param, req_param)

def get_function_lines(con, fName):
    if fName == "":
        return 0
    query = """
    SELECT line
    FROM function
    WHERE
        name == ?
    """
    cursor = con.execute(query, [fName])
    data = cursor.fetchall()
    ln = int(data[0][0])
    return ln

def get_unres_script_inclusion(con):
    query = """
    SELECT distinct(file) FROM include AS inc WHERE
        inc.fully_dynamic = 1
        AND
        inc.resolved_include = ""
    """
    cursor = con.execute(query)
    data = cursor.fetchall()
    res = set()
    for f in data:
        if f[0] != '':
            res.add(f[0])

    return res


def get_dependencies(con, scriptName):
    script_dep = []
    query = """
    WITH RECURSIVE 
    includes(x) AS ( 
        SELECT ? 
            UNION 
        SELECT DISTINCT INC.resolved_include 
        FROM include INC, includes 
        WHERE INC.file = includes.x 
        ), 
    class_instances(x) AS ( 
        SELECT ? 
            UNION 
        SELECT DISTINCT RCI.resolved_class_file 
        FROM resolved_class_instance RCI, class_instances 
        WHERE RCI.file = class_instances.x 
        ) 
    SELECT DISTINCT x FROM includes WHERE x is not NULL 
    union 
    SELECT DISTINCT x FROM class_instances"""

    unres_query = """
    SELECT f.path FROM file AS f WHERE 
    EXISTS
    ( 
    SELECT * FROM include AS inc 
    WHERE 
        inc.file = ?
        AND 
        inc.fully_dynamic = 1 
        AND
        NOT EXISTS 
        (
        SELECT * FROM include AS inc1 
        WHERE 
            inc1.file = ?
            AND 
            inc1.fully_dynamic = 0 
            AND 
            inc1.include_string = inc.include_string
        )
    )
    """
    cursor = con.execute(query, (scriptName, scriptName))
    data = cursor.fetchall()
    for f in data:
        if f[0] != '':
            script_dep.append(f[0])

    ## take care of unresolved dependencies
    logSet = set()
    for script in script_dep:
        cursor = con.execute(unres_query, (script, script))
        data = cursor.fetchall()
        for f in data:
            if f[0] != '' and f[0] not in script_dep:
                logSet.add(script)
                script_dep.append(f[0])

    return script_dep, logSet


def read_call_file(path):
    f = open(path).readlines()
    edges = set()
    for line in f:
        if line != "" and "->" in line:
            caller = line.strip("\n").split("->")[0]
            callee = line.strip("\n").split("->")[1]
            if caller == "(null)" or callee=="(null)":
               continue 
            if caller == "{main}":
                edges.add(("main", callee))
            else:
                edges.add((caller, callee))
    return edges 

def read_log(path, ip, uri):
    nv_filetype = [".js",".png", ".jpg",".jpeg",".txt",".tiff",".css"]
    f = open(path).read()
    entpoints = set()
    files = set()
    totalNum = 0
    pat = (r''
           '(\d+.\d+.\d+.\d+)\s-\s-\s' #IP address
           '\[(.+)\]\s' #datetime
           '"\w+\s(.+)\s\w+/.+"\s' #requested file
           '(\d+)\s' #status
           '(\d+)\s' #bandwidth
           '"(.+)"\s' #referrer
           '"(.+)"' #user agent
           )
    requests = set()
    x = re.findall(pat,f)
    if len(x) != 0:
        for req in x:
            if ip == req[0] and "404" not in req[3]:
                requests.add(req[2])
    for req in requests:
        if "?" in req:
            tmp = req.split("?")[0]
            if not tmp.endswith(".php") and not tmp.endswith("/"):
                continue
            res = tmp.split("/")
            for nv in nv_filetype:
                if res[-1].endswith(nv):
                    continue
            if tmp.endswith(".php") and tmp.startswith(uri):
                entpoints.add("main|" + tmp[len(uri):])
                files.add(tmp[1:])
            elif req.endswith("/"):
                if req == "/":
                    entpoints.add("main|index.php")
                    files.add("index.php")
                elif req.endswith(".php/") and req.startswith(uri):
                    entpoints.add("main|" + tmp[len(uri):-1])
                else:
                    entpoints.add("main|" + tmp[len(uri):] + "index.php")
                    files.add(tmp[len(uri):]+"index.php")
            
        elif req.endswith(".php") and req.startswith(uri):
            entpoints.add("main|" + req[len(uri):])
            files.add(req[len(uri):])
        elif req.endswith("/") and req.startswith(uri):
            res = req.split("/")
            for nv in nv_filetype:
                if res[-1].endswith(nv):
                    continue
            if req == "/":
                entpoints.add("main|index.php")
                files.add("index.php")
            elif req.endswith(".php/") and req.startswith(uri):
                entpoints.add("main|" + req[len(uri):-1] )
            else:
                entpoints.add("main|" + req[len(uri):] + "index.php")
                files.add(req[len(uri):]+"index.php")
    return entpoints,files

def getList(DIR):
    lst = []
    for path, dirs, files, in os.walk(DIR):
        for fn in files:
            if fn.endswith(".php") :
                if len(path) == len(DIR):
                    lst.append("main|" + fn)
                else:
                    lst.append("main|" + path[len(DIR):] + "/" +fn)
    return lst

def get_len_nodes(edgelist):
    nodes= set ()
    for item in edgelist:
        nodes.add(item[0])
        nodes.add(item[1])
    return len(nodes)

def get_nodes(edgelist):
    nodes= set ()
    for item in edgelist:
        nodes.add(item[0])
        nodes.add(item[1])
    return nodes

def load_files(path):
    ## read function files
    f = open(path+"functions.txt","r").readlines()
    funcs = set()
    i = 0
    for line in f:
        funcs.add(line.strip("\n"))

    max_func = len(funcs)
    ## read method files
    m = open(path+"methods.txt", "r").readlines()
    methods = set()
    for line in m:
        methods.add(line.strip("\n"))

    max_method = len(methods)

    unres = open(path+"unresolved.txt","r").readlines()
    unres_funcs = set()
    for line in unres:
        x = re.search("{(.*)}{(.*)}", line.strip("\n"))
        if x.group(1) and x.group(2):
            unres_funcs.add((x.group(1), x.group(2)))
   ## read calls file which include all the calls in the web app
    return (methods, funcs, max_method, max_func, unres_funcs)

## extract a set of items from List that are similar to Item
def getMatchFuncs(lst, Item):
    l = []
    if Item.startswith('|'):
        Item = Item[1:]
    Item = Item.replace("\\","\\\\")
    Item = Item +"\|"
    try:
        r = regex.compile(Item, re.IGNORECASE)
        l = list(filter(r.match, lst))
    except Exception as e:
        logging.warning("getMatchFuncs[%s]:Item is [%s]"%(e, Item))
    return l

def getMatchMethods(lst, item):
    l = []
    if item.startswith('|'):
        item = item[1:]
    item = item.replace("\\","\\\\")
    item = item + "\|"
    try:
        r = regex.compile(item, re.IGNORECASE)
        l = list(filter(r.match, lst))
        ## check for replacement of __construct with the classname
        if "__construct" in item and len(l) == 0:
            clsName = item.split("\\")[0]
            res = [x for x in lst if x.startswith(clsName+"\\"+clsName)]
            return res
    except Exception as e :
        logging.warning("getMatchMethods[%s]: Item is [%s]"%(e, item))
    return l


data = {}

def gen_cg(methods, max_method, funcs, max_func, path, con):
    cg_file = open(path+"calls.txt",'r').readlines()
    cg = {}
    cg_rev = {}
    for line in tqdm(cg_file, desc="Processing call-graph file"):
        caller = line.strip("\n").split("->")[0]
        #logging.warning("Processing [%s]"%(caller))
        callees = set()
        if len(line.strip("\n").split("->")) < 2:
            continue
            return {}
        if len(line.strip("\n").split("->")[1].split("#")) < 2 :
                #print (line)
            continue
        hmap = {}
        if IMPROVE_CANDID_SELECTION:
            tmp = line.strip("\n").split("->")[1].split("#")[1:]
            for i in range(0,len(tmp), 2):
                c = tmp[i]
                hmap[c] = int(tmp[i+1])
                callees.add(c)
        else:
            for c in line.strip("\n").split("->")[1].split("#")[1:]:
                callees.add(c)

        cg[caller] = []
        data[caller] = {}
        logging.warning("processing caller [%s]"%(caller))
        for callee in list(callees):
            mList = set()
            fList = set()
            
            if "\\" not in  callee:
                fList = list(getMatchFuncs(funcs, callee))
                for f in getMatchMethods(methods, callee):
                    fList.append(f)
            else:
                mList = list(getMatchMethods(methods, callee))
            if len(mList) == max_method:
                logging.error("caller with maximum number of methods: %s"%(caller))
                unhandled.add(caller)
                continue
            if len(fList) == max_func or len(fList) == max_func+ max_method:
                logging.error("caller with maximum number of functions: %s [%s]"%(caller,callee))
                unhandled.add(caller)
                continue
            if "call_user_func" in callee: 
                unhandled.add(caller)
                logging.error("caller invokes call_user_func_*: %s"%(caller))
                continue
    
            logging.warning("----- find matching items to callee [%s][%d]"%(callee, len(fList)+len(mList)))
            logging.warning("----- find matching items to callee [%s][%s]"%(callee, ' '.join(mList) + "  " + ' '.join(fList)))
            
            if len(fList) != 0:
                for i in list(fList):
                    if "Test" not in i and "simpletest" not in callee:
                        ## cg[caller].append(i)
                        ## we create a reverse call-graph 
                        ## easier to traverse
                        cg[caller].append(i)
                        if i not in cg_rev.keys():
                            cg_rev[i] = []
                        cg_rev[i].append(caller)
                        if callee not in data[caller]:
                            data[caller][callee] = []
                            data[caller][callee].append(i)
                        else:
                            data[caller][callee].append(i)


            if len(mList) != 0:
                for i in list(mList):
                    if "Test" not in i and "simpletest" not in i:
                        cg[caller].append(i)
                        if i not in cg_rev.keys():
                            cg_rev[i] = []
                        cg_rev[i].append(caller)
                        if callee not in data[caller]:
                            data[caller][callee] = []
                            data[caller][callee].append(i)
                        else:
                            data[caller][callee].append(i)

    return cg, cg_rev, data


def parse_trace_files(d):
    dynamic_trace_function = {}
    dynamic_trace_file = {}
    # We check for json files of already parsed traced files
    if os.path.exists(d+"/dyn_trace_func.json") and os.path.exists(d+"/dyn_trace_file.json"):
        with open(d+"/dyn_trace_func.json", "r") as f:
            dyn_trace_function = json.load(f)
        with open(d+"/dyn_trace_file.json", "r") as f:
            dyn_trace_file = json.load(f)
        return dyn_trace_function, dyn_trace_file

    # if there is no dynamic trace file, then empty dictionary returns
    elif not os.path.exists(d+"/traces/"):
        return dynamic_trace_function, dynamic_trace_file
    
    else:
        tmp_dir = d + "/traces/"
    # There is no pre-parsed files, so we should do it ourselves
        for fl in tqdm(os.listdir(tmp_dir),desc="parsing trace files"):
            fname = os.path.join(tmp_dir, fl)
            with open(fname, errors='ignore') as f:
                l = f.readline()
                while l:
                    try:
                        l = f.readline()
                    except:
                        continue
                    info = l.split("\t")
                    if len(info) > 9:
                        # info[5] --> invoked function
                        # info[7] --> included script (if exists)
                        # info[8,9] --> the executed file and line number
                        # 1 --> identifies include statement, 0 --> function call
                        if (info[5] == "require" or info[5] == "include" or info[5] == "require_once" or info[5] == "include_once") and info[7] != "":
                            ## it is an included script
                            ## we  should get rid of the /var/www/html
                            key = info[8][len("/var/www/html/"):]
                            if key not in dynamic_trace_file.keys():
                                dynamic_trace_file[key] = []
                            if info[7][len("/var/www/html/"):] not in dynamic_trace_file[key]:
                                dynamic_trace_file[key].append(info[7][len("/var/www/html/"):])
                            #print("{%s}{%s:%s}" %(info[7], info[8], info[9]))
                        else:
                            key = info[8][len("/var/www/html/"):] + ":" + info[9]
                            if key not in dynamic_trace_function.keys():
                                dynamic_trace_function[key] = []
                            if info[5] not in dynamic_trace_function[key]:
                                dynamic_trace_function[key].append(info[5])
                            #print("{%s}{%s:%s}" %(info[5], info[8], info[9]))
#                    except:
#                        pass
        # Dump the parsed  traces data
        with open(d+"/dyn_trace_func.json", "w") as f:
            json.dump(dynamic_trace_function, f, indent =4)
        with open(d+"/dyn_trace_file.json", "w") as f:
            json.dump(dynamic_trace_file, f, indent = 4)

    return dynamic_trace_function, dynamic_trace_file


def prune_cg(cg, entryList, db_con, dyn_trace_file, entstat = None):
    num_calls = []
    edgeList = set()
    unres = 0
    seen = set()
    for en in tqdm(entryList, desc="Pruning the Call-graph based on the entrypoints"):
        logging.warning("going over entry-point %s", en)
        ## Now we have cg that contains list of all the calls
        ## create list of edges
        callers = [] ## only get call graph for specific files and functions
        if en in cg.keys():
            callers.append(en)
        else:
            continue
        ## here we look at the script dependencies of each entry-point

        edges = set()
        edgeList.add((en,""))
        for caller in callers:
            if caller in seen:
                continue
            if db_con:
                if len(caller.split("|")) == 2:
                    files, unres = get_dependencies(db_con, caller.split("|")[1])
                    for f in files:
                        if "main|"+f not in callers:
                            callers.append("main|"+f)
                            edgeList.add(("main|"+f, ""))

            
            if caller in cg.keys():
                for item in cg[caller]:
                    if item not in callers:
                        callers.append(item)
                    edgeList.add((caller, item))
                    edges.add((caller, item))

            seen.add(caller)

        num_calls.append((get_len_nodes(edges)))
    return edgeList, num_calls


def add_magic_callees(cg, method):
    items = [method]
    seen = []
    res = [method]
    for item in items:
        if item in cg.keys():
            if item not in seen:
                seen.append(item)
                for callee in cg[item]:
                    res.append(callee)
                    items.append(callee)

        
    return res

def add_magic_methods(allowlist, methods, cg):
    magics = ["__construct","__destruct","__wakeup","__sleep","__call","__set","__get","__unset","__isset","__serialize","__unserialize", "getIterator","__toString"]
    lists = set()
    for item in tqdm(allowlist, desc="add magic methods"):
        lists.add(item)
        parts = item.split("|")
        if "\\" in parts[0]:
            method = parts[0].split("\\")
            for magic in magics:
                ## build the method signature
                lookupitem = "\\".join(method[0:-1]) + "\\" + magic
                logging.warning("----- search for magic method [%s]"%(lookupitem))
                ## add path
                lookupitem += "|" + parts[1]
                if lookupitem in methods:
                    ## if this magic method exists in the class, we add it to the allowedlist
                    lists.add(lookupitem)
                    res = add_magic_callees(cg, lookupitem)
                    for item in res:
                        if item not in lists:
                            lists.add(item)
            lookupitem = "\\".join(method[0:-1]) + "\\" + "\\".join(method[0:-1])
            logging.warning("----- search for magic method [%s]"%(lookupitem))
            lookupitem += "|" + parts[1]
            if lookupitem in methods:
                ## if this magic method exists in the class, we add it to the allowedlist
                lists.add(lookupitem)
                res = add_magic_callees(cg, lookupitem)
                for item in res:
                    if item not in lists:
                        lists.add(item)

    return lists

def remove_ns(allowlist, methods, funcs):
    lists = set()
    for item in allowlist:
        if "\\" in item:
            if item in methods:
                # it has a namespace so we remove 
                it = item.split("\\")
                if len(it) > 2:
                    fname = "\\".join([it[-2],it[-1]])
                    lists.add(fname)
                else:
                    lists.add(item)

            elif item in funcs:
                it = item.split("\\")
                if len(it) > 1:
                    fname = it[-1]
                    lists.add(fname)
                else:
                    lists.add(item)
        else:
            lists.add(item)
    return lists



def show_help():
    print('''
    get_cg.py -c -d -p root_dir -l log_file -I ip -u uri -i json_input -o json_output
    ---------------------------------------------------------------------------------
    -h : show this help menu
    -c : outputs the set of scripts needs to be crawled to resolved dynamic function calls
    -d : outputs the list of allowed functions and scripts in the web app --> the rest will be de-bloated
    -p : the path to generated file by the static anlaysis
    -l : the path to the log file
    -I : the IP to filter out requests in the access-log
    -u : the uri path used to filter the accessed file
    -i : the path for input json file (used for analyze option). The default value is output.json
    -o : the path for output json file ( used for -g to output the CG). The default value is input.json
        ''')

def main(argv):
    PATH = ''
    log_file = ''
    uri_path = ''
    ip = ''
    json_output = "output.json"
    json_input = "input.json"
    graph_input = ""
    deb_output = "debloated_output.json"
    deb_cg = False
    crawl = False
    compare = False
    dynamic_coverge = False
    compare_unres = False
    first_cg = ''
    second_cg = ''

    try:
        opts, args = getopt.getopt(argv, "hCcdDUp:l:I:u:i:o:f:s:",["help","compare","crawl","deb","dyncover","Unrescom","rootdir=","logfile=","ip=","uri=", "input=","output=", "first=", "second="])
    except getopt.GetoptError:
        print("exception in the getopt")
        sys.exit(2)

    for opt, arg in opts:
        if opt == '-h':
            show_help()
            sys.exit(2)
        elif opt in ("-d", "--deb"):
            deb_cg = True
        elif opt == '-C':
            compare = True
        elif opt == '-c':
            crawl = True
        elif opt == '-U':
            compare_unres = True
        elif opt == '-D':
            dynamic_coverge = True
        elif opt == '-p':
            PATH = arg
        elif opt == '-l':
            log_file = arg
        elif opt == '-u':
            uri_path = arg
        elif opt == '-I':
            ip = arg
        elif opt == '-o':
            json_output = arg
        elif opt == '-i':
            json_input = arg
        elif opt == '-f':
            first_cg = arg
        elif opt == '-s':
            second_cg = arg

    
    if deb_cg:
        
        if PATH != '':
            logging.basicConfig(filename=PATH+"output.log", filemode='w', level=logging.INFO, format='%(message)s')
        if PATH == '' :
            print("at least one of the args is empty!!")
            show_help()
            sys.exit(2)

        con = db_connect(PATH+"database.db")
        ## load the necessary files that was generated by the static analysis
        (methods, funcs, max_method, max_func, unres_funcs) = load_files(PATH)
        logging.warning("finish loading the files")

        ## process the static analysis files
        ## generate the call-graph
        cg, cg_rev,  data = gen_cg(methods, max_method, funcs, max_func, PATH, con)
        logging.warning("finish processing the calls.txt")
        
        with open(PATH+"fanout_"+json_output,'w') as outfile:
            json.dump(data, outfile, indent=4)

                
        entpoint, files = read_log(log_file,ip, uri_path)
        dynamic_trace_function = {}
        dynamic_trace_file = {}
        ## parse function trace files 
        dyn_trace_function, dyn_trace_file = parse_trace_files(PATH)
            
        ## dump the list of functions and scripts that needs to stay in the web app
        edges, num_calls = prune_cg(cg, entpoint, con, dynamic_trace_file)
        logging.warning("finish creating the call-graph based on the entry-points")
        
        # remove all PHP internal functions from set_funcs
        set_funcs = get_nodes(edges)
        newList = [x for x in set_funcs if ".php" in x or ".inc" in x or ".module" in x]

        List = add_magic_methods(newList, methods, cg)        
        newlist = remove_ns(List, methods,funcs)
        with open(PATH+"allowed.txt", 'w') as outfile:
            for i in sorted(newlist):
                outfile.write(i)
                outfile.write("\n")
    
    else:
        print("nothing is happening!!!")
     
if __name__ == "__main__":
    main(sys.argv[1:])

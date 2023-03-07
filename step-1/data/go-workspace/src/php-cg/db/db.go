package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

type Tx struct {
	*sql.Tx
}

type Include struct {
	File             sql.NullString
	Resolved_include sql.NullString
	Static           sql.NullInt64
	Count            sql.NullInt64
	Include_string   sql.NullString
}

type SyscallRequirement map[string]map[string][]string

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func OpenDb(filepath string) (*DB, error) {
	db, err := sql.Open("sqlite3", filepath)
	checkErr(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS file (path TEXT PRIMARY KEY, UNIQUE(path));")

	checkErr(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS class_definition (id INTEGER PRIMARY KEY AUTOINCREMENT, fqn TEXT, name TEXT, file TEXT, FOREIGN KEY(file) REFERENCES file(path), UNIQUE(fqn, file));")
	checkErr(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS class_instance(file STRING, fqn TEXT, name TEXT, FOREIGN KEY(file) REFERENCES file(path), UNIQUE(file, fqn, name))")
	checkErr(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS resolved_class_instance(file STRING, resolved_class_file STRING, class_name, FOREIGN KEY(file) REFERENCES file(path), FOREIGN KEY(resolved_class_file) REFERENCES class_definition(path), UNIQUE(file, class_name, resolved_class_file))")
	checkErr(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS function (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, line INT, len_param INT, req_param INT, UNIQUE(name))")
	checkErr(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS syscall (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, number INT, UNIQUE(name, number))")
	checkErr(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS syscall_requirement (syscall_id INTEGER, function_id INTEGER, FOREIGN KEY(syscall_id) REFERENCES syscall(id), FOREIGN KEY(function_id) REFERENCES function(id))")
	checkErr(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS function_call (file STRING, function_id INTEGER, name TEXT, FOREIGN KEY(file) REFERENCES file(path), FOREIGN KEY(function_id) REFERENCES function(id), UNIQUE(file, name))")
	checkErr(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS include (id INTEGER PRIMARY KEY AUTOINCREMENT, file TEXT, resolved_include TEXT, count INTEGER, fully_dynamic INTEGER, include_string TEXT, FOREIGN KEY(file) REFERENCES file(path), FOREIGN KEY(resolved_include) REFERENCES file(path), UNIQUE(file, include_string, resolved_include))")
	checkErr(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS include_component (include_id TEXT, count INTEGER, contents TEXT, type STRING, FOREIGN KEY(include_id) REFERENCES include(id))")
	checkErr(err)
	return &DB{db}, nil
}

func (db *DB) Begin() (*Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{tx}, nil
}

func (tx *Tx) CreateFile(path string) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO file (path) VALUES (?)`, path)
	return err
}

func (tx *Tx) CreateClass(fqn string, name string, file string) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO class_definition (fqn, name, file) VALUES (?, ?, ?)`, fqn, name, file)
	return err
}

func (tx *Tx) CreateClassInstance(file string, fqn string, name string) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO class_instance (file, fqn, name) VALUES (?, ?, ?)`, file, fqn, name)
	return err
}

// Check what namespacing does to this
func (tx *Tx) CreateFunction(name string, line int, len_param int, req_param int) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO function (name, line, len_param, req_param) VALUES (?, ?, ?, ?)`, name, line, len_param, req_param)
	return err
}

func (tx *Tx) CreateSyscall(name string, number int) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO syscall (name, number) VALUES (?, ?)`, name, number)
	return err
}

func (tx *Tx) CreateSyscallRequirement(function_name string, syscall_name string) error {
	query := `INSERT OR REPLACE INTO syscall_requirement (function_id, syscall_id) SELECT
	(SELECT id FROM function WHERE name=?) as function_id,
	(SELECT id FROM syscall WHERE name=?) as syscall_id`
	_, err := tx.Exec(query, function_name, syscall_name)
	return err
}
func (tx *Tx) CreateFunctionCall(file string, function_name string) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO function_call (file, name) VALUES (?, ?)`, file, function_name)
	return err
}

func (tx *Tx) CreateInclude(file string, resolved_include string, count int, fully_dynamic int, include_string string) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO include (file, resolved_include, count, fully_dynamic, include_string) VALUES (?, ?, ?, ?, ?)`, file, resolved_include, count, fully_dynamic, include_string)
	checkErr(err)
	return err
}
func (tx *Tx) CreateResolvedClassInstance(file string, resolvedClassFile string, className string) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO resolved_class_instance (file, resolved_class_file, class_name) VALUES (?, ?, ?)`, file, resolvedClassFile, className)
	checkErr(err)
	return err
}

func (Db *DB) GetFunctionCallsForFileImmediate(file string) []string {
	query := `
SELECT DISTINCT F.name 
FROM function F 
INNER JOIN function_call L ON L.name=F.name
WHERE L.file = ?
`
	rows, err := Db.Query(query, file)
	checkErr(err)
	var functions []string
	for rows.Next() {
		function := ""
		err := rows.Scan(&function)
		checkErr(err)
		functions = append(functions, function)
	}
	return functions
}

func (Db *DB) GetSyscallsForFileImmediate(file string) []string {
	query := `
SELECT DISTINCT SC.name FROM syscall SC
INNER JOIN syscall_requirement SCR ON SC.id=SCR.syscall_id
INNER JOIN function F ON SCR.function_id=F.id
INNER JOIN function_call L ON L.name=F.name
WHERE L.file = ?
`
	rows, err := Db.Query(query, file)
	checkErr(err)
	var syscalls []string
	for rows.Next() {
		syscall := ""
		err := rows.Scan(&syscall)
		checkErr(err)
		syscalls = append(syscalls, syscall)
	}
	return syscalls
}

func (Db *DB) GetSyscallsForFile(file string) SyscallRequirement {
	syscalls := make(SyscallRequirement)
	query := `
WITH RECURSIVE
includes(x) AS (
        SELECT ?
             UNION
        SELECT DISTINCT INC.resolved_include
        FROM includes, 
		(SELECT INC.file as file, INC.resolved_include as resolved_include
		FROM include INC
		UNION
		SELECT RCI.file as file, RCI.resolved_class_file as resolved_include
		FROM resolved_class_instance RCI) INC
        WHERE INC.file = includes.x AND INC.resolved_include != includes.x AND INC.resolved_include IS not null
)
SELECT DISTINCT SC.name, I.x, L.name FROM syscall SC
INNER JOIN syscall_requirement SCR ON SC.id=SCR.syscall_id
INNER JOIN function F ON SCR.function_id=F.id
INNER JOIN function_call L ON L.name=F.name
INNER JOIN includes I ON L.File=I.x;
`
	// union
	// SELECT DISTINCT path
	//       FROM file
	//       WHERE
	//       0 = (SELECT MIN(y) FROM includes)
	// ) fin ON L.file = fin.x
	rows, err := Db.Query(query, file)
	checkErr(err)
	for rows.Next() {
		syscall := ""
		file := ""
		functionName := ""
		err := rows.Scan(&syscall, &file, &functionName)
		if _, ok := syscalls[syscall]; !ok {
			syscalls[syscall] = make(map[string][]string)
		}
		syscalls[syscall][file] = append(syscalls[syscall][file], functionName)
		checkErr(err)
	}
	return syscalls
}

func (Db *DB) GetSyscallIdsForFile(file string) []int {
	var syscalls []int
	query := `
WITH RECURSIVE
includes(x, y) AS (
        SELECT ?, 1
                UNION
        SELECT DISTINCT INC.resolved_include, INC.static
        FROM include INC, includes
        WHERE INC.file = includes.x AND INC.processed = 1
)
, class_instances(x, y) AS (
        SELECT ?, 1
                UNION
        SELECT DISTINCT RCI.resolved_class_file, 1
        FROM resolved_class_instance RCI, class_instances
        WHERE RCI.file = class_instances.x
)
SELECT DISTINCT SC.name FROM syscall SC
INNER JOIN syscall_requirement SCR ON SC.id=SCR.syscall_id
INNER JOIN function F ON SCR.function_id=F.id
INNER JOIN function_call L ON L.name=F.name
INNER JOIN (
SELECT DISTINCT x from includes WHERE x is not NULL
union
SELECT DISTINCT x FROM class_instances
union
SELECT DISTINCT path
      FROM file
      WHERE
      0 = (SELECT MIN(y) FROM includes)
) fin ON L.file = fin.x
`
	rows, err := Db.Query(query, file, file)
	checkErr(err)
	for rows.Next() {
		syscall := 0
		err := rows.Scan(&syscall)
		checkErr(err)
		syscalls = append(syscalls, syscall)
	}
	return syscalls
}

func (Db *DB) GetIncludes(file string) []Include {
	rows, err := Db.Query(`SELECT resolved_include, static, include_string FROM include where file = ?`, file)
	checkErr(err)
	defer rows.Close()
	includes := []Include{}
	for rows.Next() {
		includes = append(includes, Include{})
		inc := &includes[len(includes)-1]
		err := rows.Scan(&inc.Resolved_include, &inc.Static, &inc.Include_string)
		checkErr(err)
	}
	return includes
}



func (Db *DB) GetIncludesOfFile(file string) []string {
	query := `
SELECT INC.resolved_include
FROM include INC
WHERE INC.file = ?1
union
SELECT RCI.resolved_class_file
FROM resolved_class_instance RCI
WHERE RCI.file = ?1
`

	rows, err := Db.Query(query, file)
	checkErr(err)
	defer rows.Close()
	includes := make([]string, 0)
	for rows.Next() {
		s := ""
		err := rows.Scan(&s)
		includes = append(includes, s)
		checkErr(err)
	}
	return includes
}


func (Db *DB) GetIncludesForFile(file string) map[string]string {
	query := `
SELECT INC.resolved_include, INC.include_string
FROM include INC
WHERE INC.file = ?1
union
SELECT RCI.resolved_class_file, RCI.class_name
FROM resolved_class_instance RCI
WHERE RCI.file = ?1
`

	rows, err := Db.Query(query, file)
	checkErr(err)
	defer rows.Close()
	includes := make(map[string]string)
	for rows.Next() {
		f := ""
		s := ""
		err := rows.Scan(&f, &s)
		includes[f] = s
		checkErr(err)
	}
	return includes
}
func (Db *DB) GetIncludesForFileRecursive(file string) []string {
	query := `
WITH RECURSIVE
includes(x) AS (
        SELECT ?
                UNION
        SELECT DISTINCT INC.resolved_include
        FROM include INC, includes
        WHERE INC.file = includes.x
)
, class_instances(x) AS (
        SELECT ?
                UNION
        SELECT DISTINCT RCI.resolved_class_file
        FROM resolved_class_instance RCI, class_instances
        WHERE RCI.file = class_instances.x
)

SELECT DISTINCT x FROM includes WHERE x is not NULL
union
SELECT DISTINCT x FROM class_instances
`

	rows, err := Db.Query(query, file, file)
	checkErr(err)
	defer rows.Close()
	includes := make([]string, 0)
	for rows.Next() {
		s := ""
		err := rows.Scan(&s)
		includes = append(includes, s)
		checkErr(err)
	}
	return includes
}
func (Db *DB) GetAllIncludes() []Include {
	rows, err := Db.Query(`SELECT file, resolved_include, count, include_string FROM include`)
	checkErr(err)
	defer rows.Close()
	includes := []Include{}
	for rows.Next() {
		includes = append(includes, Include{})
		inc := &includes[len(includes)-1]
		err := rows.Scan(&inc.File, &inc.Resolved_include, &inc.Count, &inc.Include_string)
		checkErr(err)
	}
	return includes
}
func (Db *DB) GetFiles() []string {
	files := []string{}
	rows, err := Db.Query(`SELECT * FROM file`)
	checkErr(err)
	defer rows.Close()
	for rows.Next() {
		file := ""
		err := rows.Scan(&file)
		files = append(files, file)
		checkErr(err)
	}
	return files
}
func (Db *DB) GetNonReferencedFiles() []string {
	files := []string{}
	rows, err := Db.Query(`
SELECT F.path FROM file F
LEFT JOIN include I ON F.path = I.resolved_include
LEFT JOIN resolved_class_instance CI ON F.path = CI.resolved_class_file
WHERE I.resolved_include IS NULL AND CI.resolved_class_file IS NULL
`)
	checkErr(err)
	defer rows.Close()
	for rows.Next() {
		file := ""
		err := rows.Scan(&file)
		files = append(files, file)
		checkErr(err)
	}
	return files
}

func (Db *DB) GetSyscallNames() []string {
	syscalls := []string{}
	rows, err := Db.Query(`
	SELECT name FROM syscall
`)
	checkErr(err)
	defer rows.Close()
	for rows.Next() {
		sc := ""
		err := rows.Scan(&sc)
		syscalls = append(syscalls, sc)
		checkErr(err)
	}
	return syscalls
}

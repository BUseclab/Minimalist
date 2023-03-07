package main

import (
	"php-cg/db"
)

var ignoreStrings = []string{
	"__LINE__",
	"__FILE__",
	"__DIR__",
	"__FUNCTION__",
	"__CLASS__",
	"__TRAIT__",
	"__METHOD__",
	"__NAMESPACE__",
	"'",
	"`",
	"\"",
}

func ResolveClassInstances(Db *db.DB, fqnClassDefinitions *map[string][]string, classDefinitions *map[string][]string, classInstances *map[string][]string) error {
	tx, _ := Db.Begin()
	for k, v := range *classInstances {
		resolutions := []string{}
		if _, ok := (*fqnClassDefinitions)[k]; ok {
			resolutions = append(resolutions, (*fqnClassDefinitions)[k]...)

		} else if _, ok := (*classDefinitions)[k]; ok {
			resolutions = append(resolutions, (*classDefinitions)[k]...)
		}
		for _, m := range resolutions {
			for _, f := range v {
				tx.CreateResolvedClassInstance(f, m, k)
			}
		}
	}
	return tx.Commit()
}

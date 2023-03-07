package includestring

import (
	l "php-cg/scan-project/logger"

	"github.com/jinzhu/copier"
)

type StringTrie struct {
	Dynamic  bool
	Content  string
	Constant string
	Children []StringTrie
}

func (st *StringTrie) Consolidate(params ...int) {
	for !st.Dynamic && len(st.Children) == 1 && st.Children[0].Dynamic && st.Content == "" && st.Constant == "" {
		st.Dynamic = st.Children[0].Dynamic
		st.Content = st.Children[0].Content
		st.Constant = st.Children[0].Constant
		newChildren := st.Children[0].Children
		st.Children = newChildren
	}
	for !st.Dynamic && len(st.Children) == 1 && !st.Children[0].Dynamic && st.Constant == "" && st.Children[0].Constant == "" {
		st.Dynamic = st.Children[0].Dynamic
		st.Content += st.Children[0].Content
		newChildren := st.Children[0].Children
		st.Children = newChildren
	}
	for st.Dynamic && len(st.Children) == 1 && st.Children[0].Dynamic {
		newChildren := st.Children[0].Children
		st.Children = newChildren
	}
	for _, c := range st.Children {
		c.Consolidate(0)
	}
}

func (st *StringTrie) AddLeaf(leaf StringTrie) {
	leafcopy := StringTrie{}
	copier.Copy(&leafcopy, &leaf)
	if len(st.Children) == 0 {
		st.Children = append(st.Children, leafcopy)
	} else {
		for i, _ := range st.Children {
			st.Children[i].AddLeaf(leafcopy)
		}
	}
}

func (st *StringTrie) AddChild(child StringTrie) {
	childcopy := StringTrie{}
	copier.Copy(&childcopy, &child)
	st.Children = append(st.Children, childcopy)
}

func (st StringTrie) DfsPaths() []StringTrie {
	result := []StringTrie{}
	if len(st.Children) == 0 {
		return []StringTrie{st}
	}
	for _, c := range st.Children {
		for _, p := range c.DfsPaths() {
			flatNode := st
			flatNode.Children = []StringTrie{}
			result = append(result, StringTrie{Content: st.Content, Dynamic: st.Dynamic})
			result[len(result)-1].AddLeaf(p)
		}
	}
	return result
}

func (st *StringTrie) ResolveConstants(ct *map[string]StringTrie) {
	if st.Constant != "" {
		if c, ok := (*ct)[st.Constant]; ok {
			if c.IsSimpleString(ct) {
				st.Content = c.SimpleString()
				st.Constant = ""
			} else {
				l.Log(l.Info, "Could not resolve constant %s : %+v", st.Constant, c)
				c.Constant = ""
				c.Dynamic = true
			}
		} else {
		}
	}
	for i, _ := range st.Children {
		st.Children[i].ResolveConstants(ct)
	}
}

func (st *StringTrie) IsSimpleString(ct ...*map[string]StringTrie) bool {
	if len(ct) == 1 {
		st.ResolveConstants(ct[0])
	}
	st.Consolidate()
	if len(st.Children) == 0 && !st.Dynamic {
		return true
	}
	return false
}

func (st StringTrie) SimpleString() string {
	st.Consolidate()
	if len(st.Children) == 0 && !st.Dynamic {
		return st.Content
	}
	return ""
}

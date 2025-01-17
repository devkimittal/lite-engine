package common

import "github.com/mattn/go-zglob"

type NodeType int32

const (
	NodeType_SOURCE   NodeType = 0 //nolint:revive,stylecheck
	NodeType_TEST     NodeType = 1 //nolint:revive,stylecheck
	NodeType_CONF     NodeType = 2 //nolint:revive,stylecheck
	NodeType_RESOURCE NodeType = 3 //nolint:revive,stylecheck
	NodeType_OTHER    NodeType = 4 //nolint:revive,stylecheck
)

type LangType int32

const (
	LangType_JAVA    LangType = 0 //nolint:revive,stylecheck
	LangType_CSHARP  LangType = 1 //nolint:revive,stylecheck
	LangType_PYTHON  LangType = 2 //nolint:revive,stylecheck
	LangType_UNKNOWN LangType = 3 //nolint:revive,stylecheck
)

// Node holds data about a source code
type Node struct {
	Pkg    string
	Class  string
	Method string
	File   string
	Lang   LangType
	Type   NodeType
}

// GetFiles gets list of all file paths matching a provided regex
func GetFiles(path string) ([]string, error) {
	matches, err := zglob.Glob(path)
	if err != nil {
		return []string{}, err
	}
	return matches, err
}

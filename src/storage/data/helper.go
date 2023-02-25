package data

import (
	"fmt"
	"strings"
)

// WhereMap represents a map of fields and values to be used in a WHERE clause

type WhereMap struct {
	clauses map[FieldName]any
}

func (m *WhereMap) String() string {
	if len(m.clauses) < 1 {
		return ""
	}
	if len(m.clauses) == 1 {
		for k, v := range m.clauses {
			return fmt.Sprintf("%s = '%s'", strings.ToLower(string(k)), v)
		}
	}
	sb := strings.Builder{}
	counter := 0
	for k, v := range m.clauses {
		sb.WriteString(fmt.Sprintf("%s = '%s' AND", strings.ToLower(string(k)), v))
		counter++
	}
	return sb.String()
}

func NewWhereMap(rawMap map[FieldName]interface{}) *WhereMap {
	return &WhereMap{
		clauses: rawMap,
	}
}

func (m *WhereMap) Set(field FieldName, value DBValue) {
	if m.clauses == nil {
		m.clauses = make(map[FieldName]any)
	}
	m.clauses[field] = value
}

func (m *WhereMap) Get(field FieldName) DBValue {
	return m.clauses[field]
}

// DBField represents a field in a database table

type DBField struct {
	name       FieldName
	typ        string
	len        int    // optional
	constraint string // optional
}

func (f *DBField) String() string {
	sb := strings.Builder{}
	sb.WriteString(strings.ToLower(string(f.name)))
	sb.WriteString(" ")
	sb.WriteString(f.typ)
	if f.len > 0 {
		sb.WriteString(fmt.Sprintf("(%d)", f.len))
	}
	if f.constraint != "" {
		sb.WriteString(" ")
		sb.WriteString(f.constraint)
	}
	return sb.String()
}

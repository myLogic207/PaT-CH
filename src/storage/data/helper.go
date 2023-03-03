package data

import (
	"fmt"
	"strings"
)

func join[K any](a []K, sep string) string {
	sb := strings.Builder{}
	for i, v := range a {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(fmt.Sprint(v))
	}
	return sb.String()
}

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
			return fmt.Sprintf("%s = '%s'", strings.ToLower(fmt.Sprint(k)), fmt.Sprint(v))
		}
	}
	sb := strings.Builder{}
	counter := 0
	for k, v := range m.clauses {
		sb.WriteString(fmt.Sprintf("%s = '%s' AND", strings.ToLower(fmt.Sprint(k)), fmt.Sprint(v)))
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

type DBInit struct {
	Name  string    `json:"name" yaml:"name"`     // e.g. "system"
	Table []DBTable `json:"tables" yaml:"tables"` // e.g. "users"
}

func (i *DBInit) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s", i.Table[0].String()))
	if len(i.Table) < 2 {
		return sb.String()
	}
	for _, table := range i.Table[1:] {
		sb.WriteString(table.String())
		sb.WriteString("\\ ")
	}
	return sb.String()
}

type DBTable struct {
	Name   string    `json:"name" yaml:"name"` // e.g. "users"
	Fields []DBField `json:"fields" yaml:"fields"`
}

func (t *DBTable) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("%s (", strings.ToLower(t.Name)))
	fLen := len(t.Fields) - 1
	for i, field := range t.Fields {
		sb.WriteString(field.String())
		if i < fLen {
			sb.WriteString(", ")
		}
	}
	return sb.String() + ")"
}

// DBField represents a field in a database table

type DBField struct {
	Name       FieldName `json:"name" yaml:"name"` // e.g. "id"
	Typ        string    `json:"type" yaml:"type"` // e.g. "INT"
	Len        int       `json:"length" yaml:"length"`
	Constraint string    `json:"constraint" yaml:"constraint"`
}

func (f *DBField) String() string {
	sb := strings.Builder{}
	sb.WriteString(strings.ToLower(string(f.Name)))
	sb.WriteString(" ")
	sb.WriteString(f.Typ)
	if f.Len > 0 {
		sb.WriteString(fmt.Sprintf("(%d)", f.Len))
	}
	if f.Constraint != "" {
		sb.WriteString(" ")
		sb.WriteString(f.Constraint)
	}
	return sb.String()
}

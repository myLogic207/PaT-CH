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

type DBInit struct {
	Table []DBTable `json:"table"`
}

func (i *DBInit) String() string {
	sb := strings.Builder{}
	for _, table := range i.Table {
		sb.WriteString(table.String())
		sb.WriteString("; ")
	}
	return sb.String()
}

type DBTable struct {
	Name   string    `json:"name"` // e.g. "users"
	Fields []DBField `json:"fields"`
}

func (t *DBTable) String() string {
	sb := strings.Builder{}
	sb.WriteString("CREATE TABLE IF NOT EXISTS")
	sb.WriteString(t.Name)
	sb.WriteString(" (")
	for _, field := range t.Fields {
		sb.WriteString(field.String())
		sb.WriteString(", ")
	}
	sb.WriteString(");")
	return sb.String()
}

// DBField represents a field in a database table

type DBField struct {
	Name       FieldName `json:"name"`
	Typ        string    `json:"type"`
	Len        int       `json:"length"`
	Constraint string    `json:"constraint"` // e.g. NOT NULL
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

package main

type Map map[string]interface{}

type Property interface {
	ToMap() map[string]interface{}
}

type Schema struct {
	Title       string
	Description string
	Root        *ObjectProperty
}

func (self *Schema) ToMap() map[string]interface{} {
	m := self.Root.ToMap()
	m["$schema"] = "http://json-schema.org/draft-04/schema#"

	return m
}

type ObjectProperty struct {
	Title       string
	Description string
	Properties  map[string]Property
	Required    []string
}

func (self *ObjectProperty) ToMap() map[string]interface{} {
	m := make(map[string]interface{})
	m["type"] = "object"
	m["title"] = self.Title
	m["description"] = self.Description
	m["required"] = self.Required

	p := make(map[string]interface{})

	for k, v := range self.Properties {
		p[k] = v.ToMap()
	}
	m["properties"] = p
	return m
}

type ArrayProperty struct {
	Description string
	Items       Property
	MinItems    int
	UniqueItems bool
}

func (self *ArrayProperty) ToMap() map[string]interface{} {
	return nil
}

type StringProperty struct {
	Description string
	Format      string
}

func (self *StringProperty) ToMap() map[string]interface{} {
	m := Map{
		"type":        "string",
		"description": self.Description,
	}

	if self.Format != "" {
		m["format"] = self.Format
	}

	return m
}

type NumberProperty struct {
	Description      string
	Minimum          int
	ExclusiveMinimum bool
}

func (self *NumberProperty) ToMap() map[string]interface{} {
	m := Map{
		"type":        "number",
		"description": self.Description,
		"minimum":     self.Minimum,
	}
	return m
}

type BooleanProperty struct {
}

func (self *BooleanProperty) ToMap() map[string]interface{} {
	return nil
}

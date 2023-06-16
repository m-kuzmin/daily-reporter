/*
Package template provides a Template. It is usually stored in a YAML file with 2 top-level keys: `vars` and `templates`.

The templates key has any number of named keys (not a list), each key is a template group. These groups can be used to
organize template strings. Each group has any number of key value pairs where the value is an array, even if its one
value. All elements of this array are treated as strings.

The vars key is used to do variable substitution using fmt.Sprintf. Here are 2 code blocks that do the same thing:

	var (
		foo = "Foo"
		bar = "Bar"
	)

	whatAreThese := fmt.Sprintf("%s is not a %s", foo, bar)

This is the yaml version:

	vars:
	  foo: Foo
	  bar: Bar
	templates:
	  firstTemplate:
	    whatAreThese: ["%s is not a %s", "foo", "bar"]

The names "foo" and "bar" are looked up in the vars map and their values are passed into Sprintf.
*/
package template

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// A template generated from a YAML file
type Template struct {
	/*
		vars:
		  var1: one
	*/
	Vars map[string]any `yaml:"vars"`

	/*
		templates:
		  template1:
			someString: ["%s", var1]
	*/
	Templates map[string]map[string][]string `yaml:"templates"`
}

/*
LoadYAMLTemplate reads and parses a YAML file into a template

Returned error value is either because the file could not be read or it could not be parsed as YAML
*/
func LoadYAMLTemplate(filename string) (Template, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return Template{}, fmt.Errorf("while reading template file: %w", err)
	}

	return NewTemplate(file)
}

/*
NewTemplate creates a new template from bytes

Returns an error if template source could not be parsed
*/
func NewTemplate(source []byte) (Template, error) {
	template := Template{}

	err := yaml.Unmarshal(source, &template)
	if err != nil {
		return Template{}, fmt.Errorf("while parsing YAML file: %w", err)
	}

	return template, nil
}

/*
Get returns a template group. You can call Group.Get() to get the specific string you're looking for.

Returned error indicates the group with this name doesn't exist in this template
*/
func (t Template) Get(group string) (Group, error) {
	if _, found := t.Templates[group]; !found {
		return Group{}, GroupNotFoundError{Name: group}
	}

	return Group{name: group, wrapped: &t}, nil
}

// Group holds a name of the group name passed into Template.Get() and a pointer to the template
type Group struct {
	name    string
	wrapped *Template
}

/*
Get returns a string from template group. The returned string could be "" (empty) if the key exists, but it's value is
an empty array.

Returned error could either be a group lookup error (the group was deleted from the template) or this key doesn't exist.
*/
func (g Group) Get(key string) (string, error) {
	group, exists := g.wrapped.Templates[g.name]
	if !exists {
		return "", fmt.Errorf("while looking up key %s: %w", key, GroupNotFoundError{Name: g.name})
	}

	fmtParams, found := group[key]
	if !found {
		return "", KeyNotFoundError{Group: g.name, Key: key}
	}

	switch len(fmtParams) {
	case 0:
		return "", nil
	case 1:
		return fmtParams[0], nil
	} // At this point len() is at least 2

	values := make([]any, len(fmtParams)-1)

	for i, varName := range fmtParams[1:] {
		values[i] = g.wrapped.Vars[varName]
	}

	return fmt.Sprintf(fmtParams[0], values[1:]...), nil
}

type GroupNotFoundError struct {
	Name string
}

func (e GroupNotFoundError) Error() string {
	return fmt.Sprintf("requested group with name %q was not found in template", e.Name)
}

type KeyNotFoundError struct {
	Group, Key string
}

func (e KeyNotFoundError) Error() string {
	return fmt.Sprintf("requested key %q from group %q was not found in template", e.Key, e.Group)
}

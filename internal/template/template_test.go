package template_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/m-kuzmin/daily-reporter/internal/template"
)

func TestGroupGet(t *testing.T) {
	t.Parallel()

	const yaml = `---
vars:
  foo: Foo
templates:
  foo:
    bar: ["%s", foo]
...
`

	templ, err := template.NewTemplate([]byte(yaml))
	if err != nil {
		t.Errorf("While parsing yaml template: %s", err)
	}

	foo, err := templ.Get("foo")
	if err != nil {
		t.Errorf("While getting foo group: %s", err)
	}

	bar, err := foo.Get("bar")
	if err != nil {
		t.Errorf("While getting bar from foo: %s", err)
	}

	if bar != "Foo" {
		t.Errorf("bar is not Foo, but %q", bar)
	}
}

func TestPercentPercent(t *testing.T) {
	t.Parallel()

	const yaml = `---
templates:
  percent:
    string: ["%%s"]
...
`

	templ, err := template.NewTemplate([]byte(yaml))
	if err != nil {
		t.Errorf("While parsing yaml template: %s", err)
	}

	percent, err := templ.Get("percent")
	if err != nil {
		t.Errorf("While getting percent group: %s", err)
	}

	str, err := percent.Get("string")
	if err != nil {
		t.Errorf("While getting string from percent: %s", err)
	}

	if str != "%s" {
		t.Errorf("percent.string is not %%s, but %s", str)
	}

	if foo := "foo"; fmt.Sprintf(str, foo) != foo {
		t.Errorf("percent.string is not foo, but %s", str)
	}
}

func TestPopulateGroupNilPtr(t *testing.T) {
	t.Parallel()

	const yaml = `---
templates:
  foo:
...
`

	templ, err := template.NewTemplate([]byte(yaml))
	if err != nil {
		t.Errorf("While parsing yaml template: %s", err)
	}

	group, err := templ.Get("foo")
	if err != nil {
		t.Errorf("While getting group foo: %s", err)
	}

	type Foo struct{}

	var (
		sPtr    *Foo
		errType *template.InvalidTypeError
	)

	err = group.Populate(sPtr)
	if errors.As(err, &errType) {
		return
	}

	t.Errorf("Expected InvalidTypeError for nil pointer, but got: %v", err)
}

func TestTemplatePopulate(t *testing.T) {
	t.Parallel()

	const yaml = `---
templates:
  foo:
    bar: [foobar]
...`

	var repsonses struct {
		Foo struct {
			Bar string `template:"bar"`
		} `template:"foo"`
	}

	templ, err := template.NewTemplate([]byte(yaml))
	if err != nil {
		t.Fatalf("While parsing template YAML: %s", err)
	}

	err = templ.Populate(&repsonses)
	if err != nil {
		t.Fatalf("While populating responses: %s", err)
	}

	if repsonses.Foo.Bar != "foobar" {
		t.Fatalf(`responses.Foo.Bar != "foobar" // but instead: %q`, repsonses.Foo.Bar)
	}
}

func TestPopulateTemplateNilPtr(t *testing.T) {
	t.Parallel()

	const yaml = `---
templates:
  foo:
    bar: [foobar]
...`

	templ, err := template.NewTemplate([]byte(yaml))
	if err != nil {
		t.Errorf("While parsing yaml template: %s", err)
	}

	var (
		parsed *struct { // nil pointer
			Foo struct {
				Bar string `template:"bar"`
			} `template:"foo"`
		}
		errType *template.InvalidTypeError
	)

	err = templ.Populate(parsed)
	if errors.As(err, &errType) {
		return
	}

	t.Errorf("Expected InvalidTypeError for nil pointer, but got: %v", err)
}

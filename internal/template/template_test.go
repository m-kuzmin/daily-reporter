package template_test

import (
	"fmt"
	"testing"

	"github.com/m-kuzmin/daily-reporter/internal/template"
)

const yaml = `---
vars:
  foo: Foo
templates:
  foo:
    bar: ["%s", foo]
  percent:
    string: ["%%s"]
...
`

func TestGroupGet(t *testing.T) {
	t.Parallel()

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

	foo := "foo"

	if fmt.Sprintf(str, foo) != foo {
		t.Errorf("percent.string is not foo, but %s", str)
	}
}

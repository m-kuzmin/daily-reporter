package slashcmd_test

import (
	"testing"

	"github.com/m-kuzmin/daily-reporter/internal/util/slashcmd"
)

func TestParseNoArgs(t *testing.T) {
	t.Parallel()

	const source = "/foo"

	cmd, ok := slashcmd.Parse(source)

	if cmd.Method != "foo" && ok {
		t.Fatalf("cmd.Method is not foo, but %q", cmd.Method)
	}
}

func TestParseSlashThenSpace(t *testing.T) {
	t.Parallel()

	if cmd, ok := slashcmd.Parse("/ foo"); ok {
		t.Fatalf("%#v was produced with no error, but a space between / and method name is an invalid command!", cmd)
	}
}

func TestParseSlashThenSpaceWithArgs(t *testing.T) {
	t.Parallel()

	if cmd, ok := slashcmd.Parse("/ foo bar"); ok {
		t.Fatalf("%#v was produced with no error, but a space between / and method name is an invalid command!", cmd)
	}
}

func TestParseNotACommand(t *testing.T) {
	t.Parallel()

	if cmd, ok := slashcmd.Parse("bar"); ok {
		t.Fatalf("%#v was produced for a string that doesnt start with a /", cmd)
	}
}

func TestQuotedArgs(t *testing.T) {
	t.Parallel()

	const (
		arg    = "bar is not foo"
		source = `/foo "` + arg + `" and then some`
	)

	cmd, _ := slashcmd.Parse(source)
	if cmd.Args[0] != arg {
		t.Fatal(cmd.Args[0])
	}
}

func TestArgsAreParsed(t *testing.T) {
	t.Parallel()

	source := "/foo"
	args := []string{
		`bar`,
		`"not foo"`,
		`\"`,
		`\\`,
		`\ `,
		`not\ foo`,
		`not\"foo`,
		`"escaped\""`,
		`"quoted\ and escaped"`,
	}

	parsedArgs := []string{
		`bar`,
		`not foo`,
		`"`,
		`\`,
		` `,
		`not foo`,
		`not"foo`,
		`escaped"`,
		`quoted and escaped`,
	}

	for _, arg := range args {
		source += " " + arg
	}

	cmd, _ := slashcmd.Parse(source)

	if cmd.Method != "foo" {
		t.Logf("Method is not foo: %q", cmd.Method)
		t.Fail()
	}

	for i, arg := range parsedArgs {
		if cmd.Args[i] == arg {
			t.Logf("Args[%d] is correct (%s)", i, arg)
		} else {
			t.Logf("Args[%d] is not `%s`, but `%s`", i, arg, cmd.Args[i]) // Using %s to reduce \escape complexity
			t.Fail()
		}
	}
}

func TestTypicalCommand(t *testing.T) {
	t.Parallel()

	const source = `/foo after bar "but not"     with \"foobar\" and\ foo\` // \ at the end is a test if it will be omitted

	args := []string{
		"after",
		"bar",
		"but not",
		"with",
		`"foobar"`,
		"and foo",
	}

	cmd, _ := slashcmd.Parse(source)

	if cmd.Method != "foo" {
		t.Logf("Method is not foo, but %q", cmd.Method)
		t.Fail()
	}

	for i, arg := range cmd.Args {
		if arg != args[i] {
			t.Logf("Args are parsed incorrectly: %#v", cmd.Args)
			t.Fail()
		}
	}
}

func TestNamedArg(t *testing.T) {
	t.Parallel()

	const source = "/list page 10"

	cmd, _ := slashcmd.Parse(source)

	if page, found := cmd.NextAfter("page"); !(page == "10" && found) {
		t.Fatal(page)
	}
}

func TestNamedArgBeforeEnd(t *testing.T) {
	t.Parallel()

	const souce = "/list last"

	cmd, _ := slashcmd.Parse(souce)

	if _, found := cmd.NextAfter("last"); found {
		t.Fatal()
	}
}

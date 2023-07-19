/*
slashcmd parses /command strings and returns a method (/something == "something") and a list of args to the method.

See slashcmd.Parse()
*/
package slashcmd

import (
	"strings"
	"unicode"
)

/*
Parse parses a string into a command. If bool is false the string was not a command.

The parser splits the arguments by space. To have an argument contain space you can surround it with `""`(surround with
double quotes) or use `\ `(backslash followed by space). For simplicity even inside the quoted region a single backslash
has to be written as two backslashes (`\\`). A literal double quote is `\"`(backslash followed by double quote). Quotes
by themselves dont indicate an argument boundary (i.e `foo"bar"` is one argument).
*/
func Parse(source string) (Command, bool) {
	parts := strings.SplitN(source, " ", 2) //nolint:gomnd
	parse := func(firstWord, wordsAfter string) (Command, bool) {
		method, ok := strings.CutPrefix(firstWord, "/")
		if !ok || method == "" {
			return Command{}, false
		}

		return Command{
			Method: method,
			Args:   splitArgs(wordsAfter),
		}, true
	}

	switch len(parts) {
	case 0:
		return Command{}, false
	case 1:
		return parse(parts[0], "")
	default:
		return parse(parts[0], parts[1])
	}
}

func splitArgs(source string) []string {
	var (
		args       = []string{}
		currentArg = ""
		isQuoted   = false
		isEscaped  = false

		appendToCurrent = func(char rune) {
			currentArg += string(char)
		}

		finalizeArg = func() {
			if currentArg != "" {
				args = append(args, currentArg)
			}

			currentArg = ""
		}
	)

	for _, char := range source {
		if char == '"' && !isEscaped {
			isQuoted = !isQuoted

			continue
		}

		if char == '\\' && !isEscaped {
			isEscaped = true

			continue
		}

		if unicode.IsSpace(char) && !(isQuoted || isEscaped) {
			finalizeArg()
		} else {
			appendToCurrent(char)
		}

		isEscaped = false
	}

	finalizeArg()

	return args
}

type Command struct {
	Method string
	Args   []string
}

// NextAfter finds `key` in Args and returns the next string (`Args[posOfKey+1]`)
func (c Command) NextAfter(key string) (string, bool) {
	for i, arg := range c.Args {
		if arg == key && len(c.Args) > i+1 {
			return c.Args[i+1], true
		}
	}

	return "", false
}

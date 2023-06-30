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

# Parsing rules

- The first character should be '/' followed by non-space characters which are the method string value.
- After the method there can be any amount of space characters (`unicode.IsSpace()`)
- After those spaces between the method and args, the args are split using these rules
  - Generally a space will indicate a boundary between args
  - `\` is an escape character which means that the next character is placed into the current argument regardless of
    other rules. So `\"\ \\` produces `" \`. If a `\` is the last character it is simply ignored.
  - Unescaped `"` will enable space escaping until the next unescaped quote.
  - If a space character is not preceded by `\` or inside a quoted region it indicates the end of the current arg and
    the beginning of a new one.

Here are a few examples:

	`doesnt start with slash`   => {}, false
	`/foo`                      => {"foo", []}
	`/foo bar`                  => {"foo", [ "bar" ]}
	`/foo "not bar"`            => {"foo", [ "not bar" ]}
	`/foo not\ bar`             => {"foo", [ "not bar" ]}
	`/foo not" "bar`            => {"foo", [ "not bar" ]}
	`/foo bar \`                => {"foo", [ "bar"]}
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
		args      = []string{}
		curArg    = ""
		isQuoted  = false
		isEscaped = false

		appendToCurrent = func(char rune) {
			curArg += string(char)
		}

		finalizeArg = func() {
			if curArg != "" {
				args = append(args, curArg)
			}

			curArg = ""
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

// NextAfter finds `key` in Args and returns the next string (Args[posOfKey+1])
func (c Command) NextAfter(key string) (string, bool) {
	for i, arg := range c.Args {
		if arg == key && len(c.Args) > i+1 {
			return c.Args[i+1], true
		}
	}

	return "", false
}

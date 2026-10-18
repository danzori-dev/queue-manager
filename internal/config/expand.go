package config

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
)

var refPattern = regexp.MustCompile(`\$\$|\$\{[^{}]*\}`)

type expander struct {
	errs map[string]error
}

func expand(tree map[string]any) error {
	e := &expander{
		errs: map[string]error{},
	}
	e.node(tree)

	errs := slices.SortedFunc(
		maps.Values(e.errs), func(a, b error) int {
			return strings.Compare(a.Error(), b.Error())
		},
	)

	return errors.Join(errs...)
}

func (e *expander) node(node any) any {
	switch v := node.(type) {
	case string:
		return e.str(v)

	case map[string]any:
		for key, child := range v {
			v[key] = e.node(child)
		}

		return v

	case []any:
		for i, child := range v {
			v[i] = e.node(child)
		}

		return v

	default:
		return node
	}
}

func (e *expander) str(s string) string {
	return refPattern.ReplaceAllStringFunc(
		s, func(match string) string {
			if match == "$$" {
				return "$"
			}

			return e.ref(match)
		},
	)
}

func (e *expander) ref(match string) string {
	name, def, hasDefault := strings.Cut(match[2:len(match)-1], ":-")
	if name == "" {
		e.errs[match] = fmt.Errorf("%w: %s: empty variable name", ErrInvalidReference, match)

		return ""
	}

	if v, ok := os.LookupEnv(name); ok {
		return v
	}

	if hasDefault {
		return def
	}

	e.errs[match] = fmt.Errorf("%w: %s", ErrMissingReference, match)

	return ""
}

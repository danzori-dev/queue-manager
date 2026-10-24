package command

import (
	"strconv"
	"strings"
	"unicode"

	tu "github.com/mymmrac/telego/telegoutil"
)

func Args(text string) string {
	i := strings.IndexFunc(text, unicode.IsSpace)
	if i < 0 {
		return ""
	}

	return strings.TrimSpace(text[i:])
}

func IDAndText(text string) (id int64, rest string, ok bool) {
	args := Args(text)

	i := strings.IndexFunc(args, unicode.IsSpace)
	if i < 0 {
		return 0, "", false
	}

	id, err := strconv.ParseInt(args[:i], 10, 64)
	if err != nil || id <= 0 {
		return 0, "", false
	}

	rest = strings.TrimSpace(args[i:])
	if rest == "" {
		return 0, "", false
	}

	return id, rest, true
}

func UserID(text string) (int64, bool) {
	_, _, args := tu.ParseCommand(text)
	if len(args) != 1 {
		return 0, false
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}

	return id, true
}

func StartPayload(text string) (string, bool) {
	_, _, args := tu.ParseCommand(text)
	if len(args) != 1 {
		return "", false
	}

	return args[0], true
}

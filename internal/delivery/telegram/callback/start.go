package callback

import (
	"strconv"
	"strings"
)

const swapStartPrefix = "swap_"

func SwapStart(eventID int64) string {
	return swapStartPrefix + strconv.FormatInt(eventID, 10)
}

func ParseSwapStart(payload string) (int64, bool) {
	raw, ok := strings.CutPrefix(payload, swapStartPrefix)
	if !ok {
		return 0, false
	}

	eventID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}

	return eventID, true
}

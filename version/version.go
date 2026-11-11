package version

import (
	"strconv"
	"strings"
)

func NewerThan(left string, right string) bool {
	leftParts, rightParts := strings.Split(left, "."), strings.Split(strings.SplitN(right, "-", 2)[0], ".")
	for index := range min(len(leftParts), len(rightParts)) {
		a, _ := strconv.Atoi(leftParts[index])
		b, _ := strconv.Atoi(rightParts[index])
		if a != b {
			return a > b
		}
	}

	return false
}

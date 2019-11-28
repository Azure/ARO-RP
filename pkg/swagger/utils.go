package swagger

import (
	"fmt"
	"regexp"
	"strings"
)

var shortVersion = regexp.MustCompile(`^v(\d{1,4}\d{1,2}\d{1,2})(preview)?$`)

// ValidateVersion validates if version was provided
// in short format - v19890211[preview]
func ValidateVersion(input string) error {
	if !shortVersion.MatchString(input) {
		return fmt.Errorf("wrong version format %s", input)
	}
	return nil
}

// longVersion return long version format from short version
func longVersion(input string) (short string, err error) {
	input = strings.Replace(input, "v", "", 1)
	yearIndex := 4
	monthIntex := 7
	dayIndex := 10
	q := input[:yearIndex] + "-" + input[yearIndex:]
	q = q[:monthIntex] + "-" + q[monthIntex:]
	if strings.Contains(input, "preview") {
		q = q[:dayIndex] + "-" + q[dayIndex:]
	}

	return q, nil
}

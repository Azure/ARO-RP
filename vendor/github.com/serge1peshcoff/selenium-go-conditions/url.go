package conditions

import (
	"strings"

	"github.com/tebeka/selenium"
)

// URLIs returns a condition that checks if the page's URL matches the expectedURL.
func URLIs(expectedURL string) selenium.Condition {
	return func(wd selenium.WebDriver) (bool, error) {
		url, err := wd.CurrentURL()
		if err != nil {
			return false, err
		}

		return url == expectedURL, nil
	}
}

// URLIsNot returns a condition that checks if the page's URL doesn't match the expectedURL.
func URLIsNot(expectedURL string) selenium.Condition {
	return func(wd selenium.WebDriver) (bool, error) {
		url, err := wd.CurrentURL()
		if err != nil {
			return false, err
		}

		return url != expectedURL, nil
	}
}

// URLContains returns a condition that checks if the page's URL includes the substring.
func URLContains(substring string) selenium.Condition {
	return func(wd selenium.WebDriver) (bool, error) {
		url, err := wd.CurrentURL()
		if err != nil {
			return false, err
		}

		return strings.Contains(url, substring), nil
	}
}

// URLNotContains returns a condition that checks if the page's URL doesn't include the substring.
func URLNotContains(substring string) selenium.Condition {
	return func(wd selenium.WebDriver) (bool, error) {
		url, err := wd.CurrentURL()
		if err != nil {
			return false, err
		}

		return !strings.Contains(url, substring), nil
	}
}

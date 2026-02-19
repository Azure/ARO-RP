package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

type calendar struct {
	Weekday []time.Weekday

	Year  []int
	Month []int
	Day   []int

	Hour   []int
	Minute []int
	Second []int
}

var calRegex = regexp.MustCompile(`(?:((?:Sun|Mon|Tue|Wed|Thu|Fri|Sat|,)+) )?([0-9,\*]+)-([0-9,\*]+)-([0-9,\*]+) ([0-9,\*]+):([0-9,\*]+):([0-9,\*]+)`)

func ParseCalendar(s string) (cal calendar, reterr error) {
	p := calRegex.FindStringSubmatch(s)
	if len(p) == 0 {
		return calendar{}, fmt.Errorf("failed to match '%s'", s)
	}

	if len(p) > 1 {
		p = p[1:]
	}

	// add the day if it's not on there
	if len(p) == 6 {
		p = append([]string{"*"}, p...)
	}

	if len(p) != 7 {
		reterr = fmt.Errorf("failed parsing, only got %#v", p)
		return
	}

	c := calendar{
		Weekday: []time.Weekday{},
		Year:    []int{},
		Month:   []int{},
		Day:     []int{},
		Hour:    []int{},
		Minute:  []int{},
		Second:  []int{},
	}

	// calc weekdays
	if p[0] != "*" {
		for weekday := range strings.SplitSeq(p[0], ",") {
			switch weekday {
			case "Mon":
				c.Weekday = append(c.Weekday, time.Monday)
			case "Tue":
				c.Weekday = append(c.Weekday, time.Tuesday)
			case "Wed":
				c.Weekday = append(c.Weekday, time.Wednesday)
			case "Thu":
				c.Weekday = append(c.Weekday, time.Thursday)
			case "Fri":
				c.Weekday = append(c.Weekday, time.Friday)
			case "Sat":
				c.Weekday = append(c.Weekday, time.Saturday)
			case "Sun":
				c.Weekday = append(c.Weekday, time.Sunday)
			}
		}
	}

	if p[1] != "*" {
		for day := range strings.SplitSeq(p[1], ",") {
			v, err := strconv.Atoi(day)
			if err != nil {
				reterr = fmt.Errorf("error parsing year: %w", err)
				return
			}
			if v < 2026 {
				reterr = fmt.Errorf("error parsing year: %d is out of range (before 2026)", v)
				return
			}
			c.Year = append(c.Year, v)
		}
	}
	if p[2] != "*" {
		for day := range strings.SplitSeq(p[2], ",") {
			v, err := strconv.Atoi(day)
			if err != nil {
				reterr = fmt.Errorf("error parsing month: %w", err)
				return
			}
			if v > 12 || v < 1 {
				reterr = fmt.Errorf("error parsing month: %d is out of range", v)
				return
			}
			c.Month = append(c.Month, v)
		}
	}
	if p[3] != "*" {
		for day := range strings.SplitSeq(p[3], ",") {
			v, err := strconv.Atoi(day)
			if err != nil {
				reterr = fmt.Errorf("error parsing day: %w", err)
				return
			}
			if v > 31 || v < 1 {
				reterr = fmt.Errorf("error parsing day: %d is out of range", v)
				return
			}
			c.Day = append(c.Day, v)
		}
	}

	if p[4] != "*" {
		for day := range strings.SplitSeq(p[4], ",") {
			v, err := strconv.Atoi(day)
			if err != nil {
				reterr = fmt.Errorf("error parsing hour: %w", err)
				return
			}
			if v > 23 || v < 0 {
				reterr = fmt.Errorf("error parsing hour: %d is out of range", v)
				return
			}
			c.Hour = append(c.Hour, v)
		}
	}

	if p[5] != "*" {
		for day := range strings.SplitSeq(p[5], ",") {
			v, err := strconv.Atoi(day)
			if err != nil {
				reterr = fmt.Errorf("error parsing minute: %w", err)
				return
			}
			if v > 59 || v < 0 {
				reterr = fmt.Errorf("error parsing minute: %d is out of range", v)
				return
			}
			c.Minute = append(c.Minute, v)
		}
	}
	if p[6] != "*" {
		for day := range strings.SplitSeq(p[6], ",") {
			v, err := strconv.Atoi(day)
			if err != nil {
				reterr = fmt.Errorf("error parsing seconds: %w", err)
				return
			}
			if v > 59 || v < 0 {
				reterr = fmt.Errorf("error parsing seconds: %d is out of range", v)
				return
			}
			c.Second = append(c.Second, v)
		}
	}

	slices.Sort(c.Weekday)
	slices.Sort(c.Year)
	slices.Sort(c.Month)
	slices.Sort(c.Day)
	slices.Sort(c.Hour)
	slices.Sort(c.Minute)
	slices.Sort(c.Second)

	return c, nil
}

func IsValid(t time.Time, sched calendar) bool {
	if len(sched.Year) != 0 {
		yearOK := false
		for _, s := range sched.Year {
			if s >= t.Year() {
				yearOK = true
				break
			}
		}
		if !yearOK {
			return false
		}
	}

	if len(sched.Month) != 0 {
		monthOK := slices.Contains(sched.Month, int(t.Month()))
		if !monthOK {
			return false
		}
	}

	if len(sched.Day) != 0 {
		dayOK := slices.Contains(sched.Day, t.Day())
		if !dayOK {
			return false
		}
	}

	if len(sched.Weekday) != 0 {
		weekdayOK := slices.Contains(sched.Weekday, t.Weekday())
		if !weekdayOK {
			return false
		}
	}

	if len(sched.Hour) != 0 {
		hourOK := slices.Contains(sched.Hour, t.Hour())
		if !hourOK {
			return false
		}
	}

	if len(sched.Minute) != 0 {
		MinuteOK := slices.Contains(sched.Minute, t.Minute())
		if !MinuteOK {
			return false
		}
	}

	if len(sched.Second) != 0 {
		SecondOK := slices.Contains(sched.Second, t.Second())
		if !SecondOK {
			return false
		}
	}

	return true
}

const nextiterlimit = 1000

func Next(now time.Time, sched calendar) (time.Time, bool) {
	nextPossible := now.Round(time.Second).Add(time.Second)

	for v := range nextiterlimit {
		// no next found
		if v == nextiterlimit {
			return nextPossible, false
		}

		if len(sched.Year) > 0 {
			for _, s := range sched.Year {
				if s == nextPossible.Year() {
					break
				}
				if s > nextPossible.Year() {
					nextPossible = time.Date(s, time.January, 1, 0, 0, 0, 0, nextPossible.Location())
					break
				}
			}
		}

		if len(sched.Month) > 0 {
			hasMonth := false
			for _, s := range sched.Month {
				if s == int(nextPossible.Month()) {
					hasMonth = true
					break
				}
				if s > int(nextPossible.Month()) {
					hasMonth = true
					nextPossible = time.Date(nextPossible.Year(), time.Month(s), 1, 0, 0, 0, 0, nextPossible.Location())
					break
				}
			}
			if !hasMonth {
				targetMonth := time.Month(sched.Month[0])
				nextPossible = time.Date(nextPossible.Year()+1, targetMonth, 1, 0, 0, 0, 0, nextPossible.Location())
			}
		}

		if len(sched.Weekday) > 0 {
			hasWeekday := false
			for _, s := range sched.Weekday {
				if s == nextPossible.Weekday() {
					hasWeekday = true
					break
				}
				if int(s) > int(nextPossible.Weekday()) {
					for {
						nextPossible = nextPossible.AddDate(0, 0, 1)
						if s == nextPossible.Weekday() {
							nextPossible = time.Date(nextPossible.Year(), nextPossible.Month(), nextPossible.Day(), 0, 0, 0, 0, nextPossible.Location())
							break
						}
					}
					hasWeekday = true
					break
				}
			}

			if !hasWeekday {
				for {
					nextPossible = nextPossible.AddDate(0, 0, 1)
					if sched.Weekday[0] == nextPossible.Weekday() {
						nextPossible = time.Date(nextPossible.Year(), nextPossible.Month(), nextPossible.Day(), 0, 0, 0, 0, nextPossible.Location())
						break
					}
				}
			}
		}

		if len(sched.Day) > 0 {
			hasDay := false
			for _, s := range sched.Day {
				if s == nextPossible.Day() {
					hasDay = true
					break
				}
				if s > nextPossible.Day() {
					inRange := time.Date(nextPossible.Year(), nextPossible.Month(), s, 0, 0, 0, 0, nextPossible.Location()).Month() == nextPossible.Month()

					if inRange {
						hasDay = true
						nextPossible = time.Date(nextPossible.Year(), nextPossible.Month(), s, 0, 0, 0, 0, nextPossible.Location())
						break
					} else {
						break
					}
				}
			}
			if !hasDay {
				if nextPossible.Month() == time.December {
					nextPossible = time.Date(nextPossible.Year()+1, time.January, sched.Day[0], 0, 0, 0, 0, nextPossible.Location())
				} else {
					nextPossible = time.Date(nextPossible.Year(), time.Month(int(nextPossible.Month())+1), sched.Day[0], 0, 0, 0, 0, nextPossible.Location())
				}
			}
		}

		if len(sched.Hour) > 0 {
			hasHour := false
			for _, s := range sched.Hour {
				if s == nextPossible.Hour() {
					hasHour = true
					break
				}
				if s > nextPossible.Hour() {
					hasHour = true
					nextPossible = time.Date(nextPossible.Year(), nextPossible.Month(), nextPossible.Day(), s, 0, 0, 0, nextPossible.Location())
					break
				}
			}
			if !hasHour {
				nextPossiblePlusDay := nextPossible.Add(time.Hour * 24)
				nextPossible = time.Date(nextPossiblePlusDay.Year(), nextPossiblePlusDay.Month(), nextPossiblePlusDay.Day(), sched.Hour[0], 0, 0, 0, nextPossible.Location())
			}
		}

		if len(sched.Minute) > 0 {
			hasMinute := false
			for _, s := range sched.Minute {
				if s == nextPossible.Minute() {
					hasMinute = true
					break
				}
				if s > nextPossible.Minute() {
					nextPossible = time.Date(nextPossible.Year(), nextPossible.Month(), nextPossible.Day(), nextPossible.Hour(), s, 0, 0, nextPossible.Location())
					hasMinute = true
					break
				}
			}
			if !hasMinute {
				nextPossiblePlus := nextPossible.Add(time.Minute * 60)
				nextPossible = time.Date(nextPossiblePlus.Year(), nextPossiblePlus.Month(), nextPossiblePlus.Day(), nextPossiblePlus.Hour(), sched.Minute[0], 0, 0, nextPossible.Location())
			}
		}

		if len(sched.Second) > 0 {
			hasSecond := false
			for _, s := range sched.Second {
				if s == nextPossible.Second() {
					hasSecond = true
					break
				}
				if s > nextPossible.Second() {
					nextPossible = time.Date(nextPossible.Year(), nextPossible.Month(), nextPossible.Day(), nextPossible.Hour(), nextPossible.Minute(), s, 0, nextPossible.Location())
					hasSecond = true
					break
				}
			}
			if !hasSecond {
				nextPossiblePlus := nextPossible.Add(time.Second * 60)
				nextPossible = time.Date(nextPossiblePlus.Year(), nextPossiblePlus.Month(), nextPossiblePlus.Day(), nextPossiblePlus.Hour(), nextPossiblePlus.Minute(), sched.Second[0], 0, nextPossible.Location())
			}
		}

		if IsValid(nextPossible, sched) {
			return nextPossible, true
		}
	}

	return nextPossible, false
}

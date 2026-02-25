package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
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
}

var calRegex = regexp.MustCompile(`^(?:(?P<weekday>[a-zA-Z,]+) )?(?P<year>[0-9,\*]+)-(?P<month>[0-9,\*]+)-(?P<day>[0-9,\*]+) (?P<hour>[0-9,\*]+):(?P<minute>[0-9,\*]+)(?::(?P<second>[0-9,\*]+))?$`)

func ParseCalendar(s string) (cal calendar, reterr error) {
	names := calRegex.SubexpNames()
	p := calRegex.FindStringSubmatch(s)
	if len(p) == 0 {
		return calendar{}, fmt.Errorf("failed to match '%s'", s)
	}

	c := calendar{
		Weekday: []time.Weekday{},
		Year:    []int{},
		Month:   []int{},
		Day:     []int{},
		Hour:    []int{},
		Minute:  []int{},
	}

	for x := range len(p) {
		switch names[x] {
		case "weekday":
			// calc weekdays -- missing counts as all
			if p[x] != "*" && p[x] != "" {
				for weekday := range strings.SplitSeq(strings.ToLower(p[x]), ",") {
					switch weekday {
					case "mon", "monday":
						c.Weekday = append(c.Weekday, time.Monday)
					case "tue", "tuesday":
						c.Weekday = append(c.Weekday, time.Tuesday)
					case "wed", "wednesday":
						c.Weekday = append(c.Weekday, time.Wednesday)
					case "thu", "thursday":
						c.Weekday = append(c.Weekday, time.Thursday)
					case "fri", "friday":
						c.Weekday = append(c.Weekday, time.Friday)
					case "sat", "saturday":
						c.Weekday = append(c.Weekday, time.Saturday)
					case "sun", "sunday":
						c.Weekday = append(c.Weekday, time.Sunday)
					default:
						reterr = fmt.Errorf("error parsing weekday: unknown weekday '%s'", weekday)
						return
					}
				}
			}

		case "year":
			if p[x] != "*" {
				for year := range strings.SplitSeq(p[x], ",") {
					v, err := strconv.Atoi(year)
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
		case "month":
			if p[x] != "*" {
				for month := range strings.SplitSeq(p[x], ",") {
					v, err := strconv.Atoi(month)
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
		case "day":
			if p[x] != "*" {
				for day := range strings.SplitSeq(p[x], ",") {
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
		case "hour":
			if p[x] != "*" {
				for hour := range strings.SplitSeq(p[x], ",") {
					v, err := strconv.Atoi(hour)
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
		case "minute":
			if p[x] != "*" {
				for minute := range strings.SplitSeq(p[x], ",") {
					v, err := strconv.Atoi(minute)
					if err != nil {
						reterr = fmt.Errorf("error parsing minute: %w", err)
						return
					}

					if !slices.Contains([]int{0, 15, 30, 45}, v) {
						reterr = fmt.Errorf("error parsing minute: '%d' is not one of 0/15/30/45", v)
						return
					}
					c.Minute = append(c.Minute, v)
				}
			}
		case "second":
			if p[x] != "00" && p[x] != "0" && p[x] != "" {
				reterr = errors.New("error parsing seconds: per-second granularity is unsupported")
				return
			}
		}
	}

	slices.Sort(c.Weekday)
	slices.Sort(c.Year)
	slices.Sort(c.Month)
	slices.Sort(c.Day)
	slices.Sort(c.Hour)
	slices.Sort(c.Minute)

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

	if t.Second() != 0 {
		return false
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

		if nextPossible.Second() != 0 {
			nextPossiblePlus := nextPossible.Add(time.Second * 60)
			nextPossible = time.Date(nextPossiblePlus.Year(), nextPossiblePlus.Month(), nextPossiblePlus.Day(), nextPossiblePlus.Hour(), nextPossiblePlus.Minute(), 0, 0, nextPossible.Location())
		}

		if IsValid(nextPossible, sched) {
			return nextPossible, true
		}
	}

	return nextPossible, false
}

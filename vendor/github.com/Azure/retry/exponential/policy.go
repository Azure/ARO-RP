package exponential

import (
	"errors"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/sanity-io/litter"
)

// Policy is the configuration for the backoff policy. Generally speaking you should use the
// default policy, but you can create your own if you want to customize it. But think long and
// hard about it before you do, as the default policy is a good mechanism for avoiding thundering
// herd problems, which are always remote calls. If not doing remote calls, you should question the use
// of this package. Note that a Policy is ignored if the service returns a delay in the error message.
type Policy struct {
	// InitialInterval is how long to wait after the first failure before retrying. Must be
	// greater than 0.
	// Defaults to 100ms.
	InitialInterval time.Duration
	// Multiplier is used to increase the delay after each failure. Must be greater than 1.
	// Defaults to 2.0.
	Multiplier float64
	// RandomizationFactor is used to randomize the delay. This prevents problems where multiple
	// clients are all retrying at the same intervals, and thus all hammering the server at the same time.
	// This is a value between 0 and 1. Zero(0) means no randomization, 1 means randomize by the entire interval.
	// The randomization factor sets a range of randomness in the positive and negative direction with a maximum
	// window of +/= RandomizationFactor * Interval. For example, if the RandomizationFactor is 0.5, the interval
	// will be randomized by up to 50% in the positive and negative direction. If the interval is 1s, the randomization
	// window is 0.5s to 1.5s.
	// Randomization can push the interval above the MaxInterval. The factor can be both positive and negative.
	// Defaults to 0.5
	RandomizationFactor float64
	// MaxInterval is the maximum amount of time to wait between retries. Must be > 0.
	// Defaults to 60s.
	MaxInterval time.Duration
}

func (p Policy) validate() error {
	if p.InitialInterval <= 0 {
		return errors.New("Policy.InitialInterval must be greater than 0")
	}
	if p.Multiplier <= 1 {
		return errors.New("Policy.Multiplier must be greater than 1")
	}
	if p.RandomizationFactor < 0 || p.RandomizationFactor > 1 {
		return errors.New("Policy.RandomizationFactor must be between 0 and 1")
	}
	if p.MaxInterval <= 0 {
		return errors.New("Policy.MaxInterval must be greater than 0")
	}
	if p.InitialInterval > p.MaxInterval {
		return errors.New("Policy.InitialInterval must be less than or equal to Policy.MaxInterval")
	}
	return nil
}

// TimeTableEntry is an entry in the time table.
type TimeTableEntry struct {
	// Attempt is the attempt number that this entry is for.
	Attempt int
	// Interval is the interval to wait before the next attempt. However, this is
	// not the actual interval. The actual interval is the Interval plus or minus
	// the RandomizationFactor.
	Interval time.Duration
	// MinInterval is the minimum interval to wait before the next attempt. This is
	// Interval minus the maximum randomization factor.
	MinInterval time.Duration
	// MaxInterval is the maximum interval to wait before the next attempt. This is
	// Interval plus the maximum randomization factor.
	MaxInterval time.Duration
}

// TimeTable is a table of intervals describing the wait time between retries. This is useful for
// both testing and understanding what a policy will do.
type TimeTable struct {
	// MinTime is the minimum time a program will have to wait if every attempt gets the minimum interval
	// when calculating the RandomizationFactor. This value changes depending
	// on if Policy.TimeTable() had attempts set >= 0 or < 0. If attempts is >= 0, then MinTime
	// is the sum of all the MinInterval values up through the attempts. If attempts is < 0, then
	// MinTime is the sum of all the MinInterval values until we reach our maximum interval setting.
	MinTime time.Duration
	// MaxTime is the maximum time a program will have to wait if every attempt gets the maximum interval
	// when calculating the RandomizationFactor. This value changes depending on
	// if Policy.TimeTable() had attempts set >= 0 or < 0. If attempts is >= 0, then MaxTime
	// is the sum of all the MaxInterval values up through the attempts. If attempts is < 0, then
	// MaxTime is the sum of all the MaxInterval values until we reach our maximum interval setting.
	MaxTime time.Duration
	// Entries is the list of minimum and maximum intervals for each attempt.
	Entries []TimeTableEntry
}

// String implements fmt.Stringer.
func (t TimeTable) String() string {
	var b strings.Builder
	w := table.NewWriter()
	w.SetOutputMirror(&b)

	b.WriteString("=============\n")
	b.WriteString("= TimeTable =\n")
	b.WriteString("=============\n")

	w.AppendHeader(table.Row{"Attempt", "Interval", "MinInterval", "MaxInterval"})
	for _, e := range t.Entries {
		w.AppendRow(table.Row{e.Attempt, e.Interval, e.MinInterval, e.MaxInterval})
	}
	w.AppendFooter(table.Row{"", "MinTime", "MaxTime"})
	w.AppendFooter(table.Row{"", "", t.MinTime, t.MaxTime})
	w.Render()

	return b.String()
}

var litterConf = litter.Options{
	StripPackageNames: true,
	HidePrivateFields: true,
	Separator:         "\t",
	StrictGo:          true,
}

// Litter writes the TimeTable as a Go struct that can be used to recreate the TimeTable.
// For use in internal testing only.
func (t TimeTable) Litter() string {
	return litterConf.Sdump(t)
}

// TimeTable will return a TimeTable for the Policy. If attempts is >= 0, then the TimeTable will
// be for that number of attempts. If attempts is < 0, then the TimeTable will be for all entries
// until the maximum interval is reached. This should only be used in tools and testing.
func (p Policy) TimeTable(attempts int) TimeTable {
	if attempts >= 0 {
		return p.timeTableWithAttempts(attempts)
	}
	return p.timeTable()
}

// timeTableWithAttempts creates a TimeTable with the given number of attempts which must be >= 0.
func (p Policy) timeTableWithAttempts(attempts int) TimeTable {
	if attempts < 0 {
		panic("BUG: attempts must be >= 0")
	}

	tt := TimeTable{
		Entries: []TimeTableEntry{
			{
				Attempt:     1,
				Interval:    0,
				MinInterval: 0,
				MaxInterval: 0,
			},
		},
	}

	interval := p.InitialInterval

	for i := 2; i <= attempts; i++ {
		minInterval := interval - time.Duration(float64(interval)*p.RandomizationFactor)
		maxInterval := interval + time.Duration(float64(interval)*p.RandomizationFactor)

		entry := TimeTableEntry{
			Attempt:     i,
			Interval:    interval,
			MinInterval: minInterval,
			MaxInterval: maxInterval,
		}
		tt.MinTime += minInterval
		tt.MaxTime += maxInterval
		tt.Entries = append(tt.Entries, entry)

		interval = time.Duration(float64(interval) * p.Multiplier)
		if interval > p.MaxInterval {
			interval = p.MaxInterval
		}
	}
	return tt
}

// timeTable creates a TimeTable for the Policy. This is for all attempts until the maximum interval
// is reached.
func (p Policy) timeTable() TimeTable {
	tt := TimeTable{
		Entries: []TimeTableEntry{
			{
				Attempt:     1,
				Interval:    0,
				MinInterval: 0,
				MaxInterval: 0,
			},
		},
	}

	interval := p.InitialInterval

	var i int
	for i = 2; interval != p.MaxInterval; i++ {
		minInterval := interval - time.Duration(float64(interval)*p.RandomizationFactor)
		maxInterval := interval + time.Duration(float64(interval)*p.RandomizationFactor)

		entry := TimeTableEntry{
			Attempt:     i,
			Interval:    interval,
			MinInterval: minInterval,
			MaxInterval: maxInterval,
		}
		tt.MinTime += minInterval
		tt.MaxTime += maxInterval
		tt.Entries = append(tt.Entries, entry)

		interval = time.Duration(float64(interval) * p.Multiplier)
		if interval > p.MaxInterval {
			interval = p.MaxInterval
		}
	}

	// This is the final entry at the maximum interval.
	entry := TimeTableEntry{
		Attempt:     i,
		Interval:    interval,
		MinInterval: interval - time.Duration(float64(interval)*p.RandomizationFactor),
		MaxInterval: interval + time.Duration(float64(interval)*p.RandomizationFactor),
	}
	tt.MinTime += entry.MinInterval
	tt.MaxTime += entry.MaxInterval
	tt.Entries = append(tt.Entries, entry)

	return tt
}

// defaults creates a new Policy with the default values.
func defaults() Policy {
	// progression will be:
	// 100ms, 200ms, 400ms, 800ms, 1.6s, 3.2s, 6.4s, 12.8s, 25.6s, 51.2s, 60s
	// Not counting a randomization factor which will be +/- up to 50% of the interval.
	return Policy{
		InitialInterval:     100 * time.Millisecond,
		Multiplier:          2,
		RandomizationFactor: 0.5,
		MaxInterval:         60 * time.Second,
	}
}

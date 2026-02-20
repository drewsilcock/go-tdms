package tdms

import (
	"fmt"
	"iter"
	"time"
)

type TimePreference int

const (
	TimePreferenceNone TimePreference = iota
	TimePreferenceAbsolute
	TimePreferenceRelative
)

type Waveform struct {
	XAxisName      string
	XAxisUnit      string
	NumSamples     uint
	StartOffset    float64
	StartTime      *time.Time
	TimePreference TimePreference

	increment float64
}

// Value returns the value at the given index.
//
// If the index is outside the waveform range, this will return an error.
//
// If this waveform is in time space, this will return number of nanoseconds
// since Unix epoch, which can then be converted to a [time.Time] using
// [time.Unix].
//
// If the waveform has a start time set, this will be added to the offset.
//
// If you know that your data is in time space and uses seconds, you can use
// [AsTimeWaveform] instead to produce [time.Time] values instead of floats.
func (w Waveform) Value(index uint) (float64, error) {
	if index >= w.NumSamples {
		return 0, fmt.Errorf("index out of range")
	}

	offset := w.StartOffset + (w.increment * float64(index))
	start := 0.0
	if w.StartTime != nil {
		start = float64(w.StartTime.UnixNano()) / float64(time.Second)
	}
	return start + offset, nil
}

// AsTimeWaveform validates that this waveform is in the time domain with an
// absolute start time and returns a [TimeWaveform] that is guaranteed to
// satisfy those constraints.
//
// This allows [TimeWaveform.Time] to skip the repeated nil-check and unit
// validation on every call, making it suitable for tight loops over millions of
// data points.
//
// Returns an error if the waveform has no start time or if its x-axis unit is
// not "s" (seconds).
func (w Waveform) AsTimeWaveform() (TimeWaveform, error) {
	if w.StartTime == nil {
		return TimeWaveform{}, fmt.Errorf("start time not set")
	}

	if w.XAxisUnit != "s" {
		return TimeWaveform{}, fmt.Errorf("unsupported unit: %s", w.XAxisUnit)
	}

	return TimeWaveform{
		startTime:   *w.StartTime,
		startOffset: w.StartOffset,
		increment:   w.increment,
		numSamples:  w.NumSamples,
	}, nil
}

// TimeWaveform is a pre-validated waveform that is guaranteed to be in the time
// domain with an absolute start time and second-based units. It is created from
// a [Waveform] using [Waveform.AsTimeWaveform].
//
// Because all invariants are checked at construction time, [TimeWaveform.Time]
// only performs a bounds check, and [TimeWaveform.Times] requires no error
// handling at all. This makes TimeWaveform ideal for performance-critical code
// that needs to resolve timestamps for millions of data points.
type TimeWaveform struct {
	startTime   time.Time
	startOffset float64
	increment   float64
	numSamples  uint
}

// Time returns the absolute time at the given sample index.
//
// Unlike [Waveform.Time], this method only performs a bounds check — the start
// time and unit validations were already done when the TimeWaveform was created
// via [Waveform.AsTimeWaveform].
func (tw TimeWaveform) Time(index uint) (time.Time, error) {
	if index >= tw.numSamples {
		return time.Time{}, fmt.Errorf("index out of range")
	}

	offset := tw.startOffset + (tw.increment * float64(index))
	return tw.startTime.Add(time.Duration(offset * float64(time.Second))), nil
}

// Times returns an iterator over all timestamps in the waveform, one per
// sample, in order.
//
// Because the TimeWaveform has already been validated and the iterator controls
// its own bounds, no errors are possible during iteration. This makes it safe
// to use in a simple for-range loop without error handling:
//
//	tw, err := waveform.AsTimeWaveform()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	for t := range tw.Times() {
//		fmt.Println(t)
//	}
func (tw TimeWaveform) Times() iter.Seq[time.Time] {
	return func(yield func(time.Time) bool) {
		for i := uint(0); i < tw.numSamples; i++ {
			offset := tw.startOffset + (tw.increment * float64(i))
			t := tw.startTime.Add(time.Duration(offset * float64(time.Second)))
			if !yield(t) {
				return
			}
		}
	}
}

// AllTimes returns all timestamps in the waveform as a slice.
//
// This is a convenience method equivalent to collecting all values from
// [TimeWaveform.Times] into a slice.
func (tw TimeWaveform) AllTimes() []time.Time {
	times := make([]time.Time, tw.numSamples)
	for i := uint(0); i < tw.numSamples; i++ {
		offset := tw.startOffset + (tw.increment * float64(i))
		times[i] = tw.startTime.Add(time.Duration(offset * float64(time.Second)))
	}
	return times
}

// NumSamples returns the number of samples in the waveform.
func (tw TimeWaveform) NumSamples() uint {
	return tw.numSamples
}

// StartTime returns the absolute start time of the waveform.
func (tw TimeWaveform) StartTime() time.Time {
	return tw.startTime
}

// Increment returns the time increment between successive samples, in seconds.
func (tw TimeWaveform) Increment() float64 {
	return tw.increment
}

// StartOffset returns the start offset of the waveform, in seconds.
func (tw TimeWaveform) StartOffset() float64 {
	return tw.startOffset
}

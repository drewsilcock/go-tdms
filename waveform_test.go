package tdms

import (
	"slices"
	"testing"
	"time"
)

func TestWaveform_Value(t *testing.T) {
	w := Waveform{
		NumSamples:  4,
		StartOffset: 10.0,
		increment:   0.5,
	}

	tests := []struct {
		name  string
		index uint
		want  float64
	}{
		{"first sample", 0, 10.0},
		{"second sample", 1, 10.5},
		{"last sample", 3, 11.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := w.Value(tt.index)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !almostEqual(got, tt.want, 1e-12) {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestWaveform_Value_OutOfRange(t *testing.T) {
	w := Waveform{NumSamples: 3, increment: 1.0}

	_, err := w.Value(3)
	if err == nil {
		t.Fatal("expected error for out-of-range index, got nil")
	}
}

func TestWaveform_Value_WithStartTime(t *testing.T) {
	startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	w := Waveform{
		NumSamples:  2,
		StartOffset: 0.0,
		StartTime:   &startTime,
		increment:   1.0,
	}

	got, err := w.Value(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Value should include the start time as seconds since epoch.
	wantStart := float64(startTime.UnixNano()) / float64(time.Second)
	if !almostEqual(got, wantStart, 1e-6) {
		t.Errorf("expected %v, got %v", wantStart, got)
	}

	got1, err := w.Value(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !almostEqual(got1-got, 1.0, 1e-6) {
		t.Errorf("expected increment of 1.0 between samples, got %v", got1-got)
	}
}

func TestWaveform_AsTimeWaveform(t *testing.T) {
	startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	w := Waveform{
		XAxisName:   "Time",
		XAxisUnit:   "s",
		NumSamples:  5,
		StartOffset: 0.5,
		StartTime:   &startTime,
		increment:   0.001,
	}

	tw, err := w.AsTimeWaveform()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tw.NumSamples() != 5 {
		t.Errorf("expected NumSamples 5, got %d", tw.NumSamples())
	}
	if !tw.StartTime().Equal(startTime) {
		t.Errorf("expected StartTime %v, got %v", startTime, tw.StartTime())
	}
	if tw.Increment() != 0.001 {
		t.Errorf("expected Increment 0.001, got %v", tw.Increment())
	}
	if tw.StartOffset() != 0.5 {
		t.Errorf("expected StartOffset 0.5, got %v", tw.StartOffset())
	}
}

func TestWaveform_AsTimeWaveform_NoStartTime(t *testing.T) {
	w := Waveform{
		XAxisUnit:  "s",
		NumSamples: 3,
		increment:  0.001,
	}

	_, err := w.AsTimeWaveform()
	if err == nil {
		t.Fatal("expected error for nil start time, got nil")
	}
}

func TestWaveform_AsTimeWaveform_UnsupportedUnit(t *testing.T) {
	startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	w := Waveform{
		XAxisUnit:  "Hz",
		NumSamples: 3,
		StartTime:  &startTime,
		increment:  1.0,
	}

	_, err := w.AsTimeWaveform()
	if err == nil {
		t.Fatal("expected error for unsupported unit, got nil")
	}
}

func TestTimeWaveform_Time(t *testing.T) {
	startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	w := Waveform{
		XAxisName:   "Time",
		XAxisUnit:   "s",
		NumSamples:  5,
		StartOffset: 0.0,
		StartTime:   &startTime,
		increment:   0.001,
	}

	tw, err := w.AsTimeWaveform()
	if err != nil {
		t.Fatalf("unexpected error creating TimeWaveform: %v", err)
	}

	tests := []struct {
		name  string
		index uint
		want  time.Time
	}{
		{
			name:  "first sample",
			index: 0,
			want:  startTime,
		},
		{
			name:  "third sample",
			index: 2,
			want:  startTime.Add(2 * time.Millisecond),
		},
		{
			name:  "last sample",
			index: 4,
			want:  startTime.Add(4 * time.Millisecond),
		},
		{
			name:  "beyond range extrapolates",
			index: 10,
			want:  startTime.Add(10 * time.Millisecond),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tw.Time(tt.index)
			if !got.Equal(tt.want) {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestTimeWaveform_Time_WithStartOffset(t *testing.T) {
	startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	w := Waveform{
		XAxisUnit:   "s",
		NumSamples:  3,
		StartOffset: 0.5,
		StartTime:   &startTime,
		increment:   1.0,
	}

	tw, err := w.AsTimeWaveform()
	if err != nil {
		t.Fatalf("unexpected error creating TimeWaveform: %v", err)
	}

	got := tw.Time(0)

	// First sample should be at startTime + 0.5s offset.
	want := startTime.Add(500 * time.Millisecond)
	if !got.Equal(want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestTimeWaveform_Times(t *testing.T) {
	startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	w := Waveform{
		XAxisUnit:   "s",
		NumSamples:  4,
		StartOffset: 0.0,
		StartTime:   &startTime,
		increment:   0.5,
	}

	tw, err := w.AsTimeWaveform()
	if err != nil {
		t.Fatalf("unexpected error creating TimeWaveform: %v", err)
	}

	want := []time.Time{
		startTime,
		startTime.Add(500 * time.Millisecond),
		startTime.Add(1 * time.Second),
		startTime.Add(1500 * time.Millisecond),
	}

	var got []time.Time
	for ts := range tw.Times() {
		got = append(got, ts)
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d timestamps, got %d", len(want), len(got))
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Errorf("index %d: expected %v, got %v", i, want[i], got[i])
		}
	}
}

func TestTimeWaveform_Times_EarlyBreak(t *testing.T) {
	startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	w := Waveform{
		XAxisUnit:   "s",
		NumSamples:  1000,
		StartOffset: 0.0,
		StartTime:   &startTime,
		increment:   0.001,
	}

	tw, err := w.AsTimeWaveform()
	if err != nil {
		t.Fatalf("unexpected error creating TimeWaveform: %v", err)
	}

	// Break after 3 iterations — should not panic or iterate all 1000.
	count := 0
	for range tw.Times() {
		count++
		if count == 3 {
			break
		}
	}

	if count != 3 {
		t.Errorf("expected 3 iterations before break, got %d", count)
	}
}

func TestTimeWaveform_Times_Empty(t *testing.T) {
	startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	w := Waveform{
		XAxisUnit:   "s",
		NumSamples:  0,
		StartOffset: 0.0,
		StartTime:   &startTime,
		increment:   1.0,
	}

	tw, err := w.AsTimeWaveform()
	if err != nil {
		t.Fatalf("unexpected error creating TimeWaveform: %v", err)
	}

	count := 0
	for range tw.Times() {
		count++
	}

	if count != 0 {
		t.Errorf("expected 0 iterations for empty waveform, got %d", count)
	}
}

func TestTimeWaveform_AllTimes(t *testing.T) {
	startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	w := Waveform{
		XAxisUnit:   "s",
		NumSamples:  3,
		StartOffset: 0.0,
		StartTime:   &startTime,
		increment:   1.0,
	}

	tw, err := w.AsTimeWaveform()
	if err != nil {
		t.Fatalf("unexpected error creating TimeWaveform: %v", err)
	}

	got := tw.AllTimes()

	want := []time.Time{
		startTime,
		startTime.Add(1 * time.Second),
		startTime.Add(2 * time.Second),
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d timestamps, got %d", len(want), len(got))
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Errorf("index %d: expected %v, got %v", i, want[i], got[i])
		}
	}
}

func TestTimeWaveform_AllTimes_MatchesTimes(t *testing.T) {
	startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	w := Waveform{
		XAxisUnit:   "s",
		NumSamples:  5,
		StartOffset: 0.25,
		StartTime:   &startTime,
		increment:   0.1,
	}

	tw, err := w.AsTimeWaveform()
	if err != nil {
		t.Fatalf("unexpected error creating TimeWaveform: %v", err)
	}

	allTimes := tw.AllTimes()
	iterTimes := slices.Collect(tw.Times())

	if len(allTimes) != len(iterTimes) {
		t.Fatalf("AllTimes returned %d items, Times iterator returned %d", len(allTimes), len(iterTimes))
	}
	for i := range allTimes {
		if !allTimes[i].Equal(iterTimes[i]) {
			t.Errorf("index %d: AllTimes=%v, Times=%v", i, allTimes[i], iterTimes[i])
		}
	}
}

func TestTimeWaveform_AllTimes_Empty(t *testing.T) {
	startTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	w := Waveform{
		XAxisUnit:   "s",
		NumSamples:  0,
		StartOffset: 0.0,
		StartTime:   &startTime,
		increment:   1.0,
	}

	tw, err := w.AsTimeWaveform()
	if err != nil {
		t.Fatalf("unexpected error creating TimeWaveform: %v", err)
	}

	got := tw.AllTimes()
	if len(got) != 0 {
		t.Errorf("expected empty slice for empty waveform, got %d items", len(got))
	}
}

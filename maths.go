package tdms

import "sort"

// interp performs a 1-D linear interpolation similar to numpy.interp.
//
// Panics if xp has a different size to fp or if either xp and fp are empty.
//
// x: x co-ordinate at which to evaluate the interpolated values.
// xp: x co-ordinates of the data points – expected to be monotonically increasing.
// fp: y co-ordinates of the data points.
// left: optional value to return when x < xp[0] (defaults to fp[0])
// right: optional value to return when x > xp[-1] (defaults to fp[-1])
func interp[T Numeric](x T, xp, fp []float64, left, right *float64) float64 {
	if len(xp) != len(fp) {
		panic("xp and fp must have the same length")
	}

	if len(xp) == 0 {
		panic("xp and fp must not be empty")
	}

	leftVal := fp[0]
	if left != nil {
		leftVal = *left
	}
	rightVal := fp[len(fp)-1]
	if right != nil {
		rightVal = *right
	}

	xi := float64(x)

	if xi < xp[0] {
		return leftVal
	}

	if xi > xp[len(xp)-1] {
		return rightVal
	}

	j := sort.SearchFloat64s(xp, xi)

	x0, x1 := xp[j-1], xp[j]
	y0, y1 := fp[j-1], fp[j]

	return y0 + (y1-y0)*(xi-x0)/(x1-x0)
}

func isMonotonicInc(x []float64) bool {
	if len(x) < 2 {
		return true
	}

	for i := 1; i < len(x); i++ {
		if x[i] < x[i-1] {
			return false
		}
	}

	return true
}

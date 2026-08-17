package executor

import (
	"errors"
	"math"
)

func asNonNegativeInt(v any) (int, error) {
	switch n := v.(type) {
	case float64:
		if n < 0 || n != math.Trunc(n) {
			return 0, errors.New("must be a non-negative integer")
		}
		return int(n), nil
	case int:
		if n < 0 {
			return 0, errors.New("must be a non-negative integer")
		}
		return n, nil
	default:
		return 0, errors.New("must be a non-negative integer")
	}
}
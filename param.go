package yuna

import (
	"fmt"
	"strconv"

	"github.com/google/uuid"
)

// ParamValue represents a parameter value, either query parameter or path parameter.
type ParamValue struct {
	vals    []string
	present bool
}

func (pv ParamValue) Present() bool {
	return pv.present
}

func (pv ParamValue) Values() []string {
	return pv.vals
}

func (pv ParamValue) Len() int {
	return len(pv.vals)
}

func (pv ParamValue) Empty() bool {
	return len(pv.vals) == 0
}

func (pv ParamValue) String() string {
	return pv.vals[0]
}

func (pv ParamValue) StringOrDefault(defaultVal string) string {
	if len(pv.vals) == 0 || pv.vals[0] == "" {
		return defaultVal
	}
	return pv.vals[0]
}

func (pv ParamValue) Strings() []string {
	return pv.vals
}

func (pv ParamValue) Bool() (bool, error) {
	if len(pv.vals) == 0 {
		return false, fmt.Errorf("missing value")
	}
	return strconv.ParseBool(pv.vals[0])
}

func (pv ParamValue) BoolOrDefault(defaultVal bool) bool {
	if len(pv.vals) == 0 {
		return defaultVal
	}
	val, err := strconv.ParseBool(pv.vals[0])
	if err != nil {
		return defaultVal
	}
	return val
}

func (pv ParamValue) Int() (int, error) {
	if len(pv.vals) == 0 {
		return 0, fmt.Errorf("missing value")
	}
	return strconv.Atoi(pv.vals[0])
}

func (pv ParamValue) IntOrDefault(defaultVal int) int {
	if len(pv.vals) == 0 {
		return defaultVal
	}
	val, err := strconv.Atoi(pv.vals[0])
	if err != nil {
		return defaultVal
	}
	return val
}

func (pv ParamValue) Int64() (int64, error) {
	if len(pv.vals) == 0 {
		return 0, fmt.Errorf("missing value")
	}
	return strconv.ParseInt(pv.vals[0], 10, 64)
}

func (pv ParamValue) Int64OrDefault(defaultVal int64) int64 {
	if len(pv.vals) == 0 {
		return defaultVal
	}
	val, err := strconv.ParseInt(pv.vals[0], 10, 64)
	if err != nil {
		return defaultVal
	}
	return val
}

func (pv ParamValue) Float64() (float64, error) {
	if len(pv.vals) == 0 {
		return 0, fmt.Errorf("missing value")
	}
	return strconv.ParseFloat(pv.vals[0], 64)
}

func (pv ParamValue) Float64OrDefault(defaultVal float64) float64 {
	if len(pv.vals) == 0 {
		return defaultVal
	}
	val, err := strconv.ParseFloat(pv.vals[0], 64)
	if err != nil {
		return defaultVal
	}
	return val
}

func (pv ParamValue) Float32() (float32, error) {
	if len(pv.vals) == 0 {
		return 0, fmt.Errorf("missing value")
	}
	val, err := strconv.ParseFloat(pv.vals[0], 32)
	if err != nil {
		return 0, err
	}
	return float32(val), nil
}

func (pv ParamValue) Float32OrDefault(defaultVal float32) float32 {
	if len(pv.vals) == 0 {
		return defaultVal
	}
	val, err := strconv.ParseFloat(pv.vals[0], 32)
	if err != nil {
		return defaultVal
	}
	return float32(val)
}

func (pv ParamValue) UUID() (uuid.UUID, error) {
	if len(pv.vals) == 0 {
		return uuid.Nil, fmt.Errorf("missing value")
	}
	return uuid.Parse(pv.vals[0])
}

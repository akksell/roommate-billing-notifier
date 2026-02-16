package bill

import (
	"strconv"
	"strings"
)

func parseDecimal(s string, out *float64) error {
	s = strings.ReplaceAll(s, ",", "")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*out = v
	return nil
}

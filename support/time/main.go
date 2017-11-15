package time

import (
	"strconv"
	goTime "time"
)

//ToInt64 represents time as milliseconds since epoch without any timezone adjustments
type Millis int64

//MillisFromString generates a ToInt64 struct from a string representing an int64
func MillisFromString(s string) (Millis, error) {
	millis, err := strconv.ParseInt(s, 10, 64)
	return Millis(int64(millis)), err
}

//MillisFromInt64 generates a ToInt64 struct from given millis int64
func MillisFromInt64(millis int64) Millis {
	return Millis(millis)
}

func (t Millis) increment(millisToAdd int64) Millis {
	return Millis(int64(t)+ millisToAdd)
}

//IsNil returns true if the timeMillis has not been initialized to a date other then 0 from epoch
func (t Millis) IsNil() bool {
	return t == 0
}

//RoundUp returns a new ToInt64 instance with a rounded up to d millis
func (t Millis) RoundUp(d int64) Millis {
	if int64(t)%d != 0 {
		return t.RoundDown(d).increment(d)
	}
	return t
}

//RoundUp returns a new ToInt64 instance with a down to d millis
func (t Millis) RoundDown(d int64) Millis {
	//round down to the nearest d
	return Millis(int64(int64(t)/d) * d)
}

//ToInt64 returns the actual int64 millis since epoch
func (t Millis) ToInt64() int64 {
	return int64(t)
}

//ToTime returns a go time.Time timestamp, UTC adjusted
func (t Millis) ToTime() goTime.Time {
	return goTime.Unix(int64(t)/1000, 0).UTC()
}

//Now returns current time in millis
func Now() Millis {
	return Millis(goTime.Now().UTC().UnixNano() / int64(goTime.Millisecond))
}

func (t Millis) String() string {
	return strconv.FormatInt(t.ToInt64(), 10)
}
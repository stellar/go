package randxdr

import (
	"math"
	"regexp"
	"strings"

	goxdr "github.com/xdrpp/goxdr/xdr"
)

// Selector is function used to match fields of a goxdr.XdrType
type Selector func(string, goxdr.XdrType) bool

// Setter is a function used to set field values for a goxdr.XdrType
type Setter func(*randMarshaller, string, goxdr.XdrType)

// Preset can be used to restrict values for specific fields of a goxdr.XdrType.
type Preset struct {
	Selector Selector
	Setter   Setter
}

// FieldEquals returns a Selector which matches on a field name by equality
func FieldEquals(toMatch string) Selector {
	return func(name string, xdrType goxdr.XdrType) bool {
		return name == toMatch
	}
}

// FieldMatches returns a Selector which matches on a field name by regexp
func FieldMatches(r *regexp.Regexp) Selector {
	return func(name string, xdrType goxdr.XdrType) bool {
		return r.MatchString(name)
	}
}

// IsPtr is a Selector which matches on all XDR pointer fields
var IsPtr Selector = func(name string, xdrType goxdr.XdrType) bool {
	_, ok := goxdr.XdrBaseType(xdrType).(goxdr.XdrPtr)
	return ok
}

// IsNestedInnerSet is a Selector which identifies nesting for the following xdr type:
//	struct SCPQuorumSet
//	{
//		uint32 threshold;
//		PublicKey validators<>;
//		SCPQuorumSet innerSets<>;
//	};
// supports things like: A,B,C,(D,E,F),(G,H,(I,J,K,L))
// only allows 2 levels of nesting
var IsNestedInnerSet Selector = func(name string, xdrType goxdr.XdrType) bool {
	if strings.HasSuffix(name, ".innerSets") && strings.Count(name, ".innerSets[") > 0 {
		_, ok := goxdr.XdrBaseType(xdrType).(goxdr.XdrVec)
		return ok
	}
	return false
}

// SetPtrToPresent is a Setter which ensures that a given XDR pointer field is not nil
var SetPtrToPresent Setter = func(m *randMarshaller, name string, xdrType goxdr.XdrType) {
	p := goxdr.XdrBaseType(xdrType).(goxdr.XdrPtr)
	p.SetPresent(true)
	p.XdrMarshalValue(m, name)
}

// SetVecLen returns a Setter which sets the length of a variable length
// array ( https://tools.ietf.org/html/rfc4506#section-4.13 ) to a fixed value
func SetVecLen(vecLen uint32) Setter {
	return func(x *randMarshaller, field string, xdrType goxdr.XdrType) {
		v := goxdr.XdrBaseType(xdrType).(goxdr.XdrVec)
		v.SetVecLen(vecLen)
		v.XdrMarshalN(x, field, vecLen)
	}
}

// SetU32 returns a Setter which sets a uint32 XDR field to a fixed value
func SetU32(val uint32) Setter {
	return func(x *randMarshaller, field string, xdrType goxdr.XdrType) {
		f := goxdr.XdrBaseType(xdrType).(goxdr.XdrNum32)
		f.SetU32(val)
	}
}

// SetPositiveNum64 returns a Setter which sets a uint64 XDR field to a random positive value
func SetPositiveNum64() Setter {
	return func(x *randMarshaller, field string, xdrType goxdr.XdrType) {
		f := goxdr.XdrBaseType(xdrType).(goxdr.XdrNum64)
		f.SetU64(uint64(x.rand.Int63n(math.MaxInt64)))
	}
}

// SetPositiveNum32 returns a Setter which sets a uint32 XDR field to a random positive value
func SetPositiveNum32() Setter {
	return func(x *randMarshaller, field string, xdrType goxdr.XdrType) {
		f := goxdr.XdrBaseType(xdrType).(goxdr.XdrNum32)
		f.SetU32(uint32(x.rand.Int31n(math.MaxInt32)))
	}
}

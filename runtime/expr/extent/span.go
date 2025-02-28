package extent

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zson"
)

// For now, we do slow-path stuff here but the interface will allow us
// to optimize with type-specific implementations.  It would be trivial here
// to create a Time range that embeds nano.Span instead of Lower/Upper
// and implements Range.

// Span represents the closed interval [first, last] where first is "less than"
// last with respect to the Span's order.Which.
type Span interface {
	First() *zed.Value
	Last() *zed.Value
	Before(*zed.Value) bool
	After(*zed.Value) bool
	In(*zed.Value) bool
	Overlaps(*zed.Value, *zed.Value) bool
	Crop(Span) bool
	Extend(*zed.Value)
	String() string
}

type Generic struct {
	first zed.Value
	last  zed.Value
	cmp   expr.CompareFn
}

// Create a new Range from generic range of zed.Values according
// to lower and upper.  The range is not sensitive to the absolute order
// of lower and upper.
func NewGeneric(lower, upper zed.Value, cmp expr.CompareFn) *Generic {
	if cmp(&lower, &upper) > 0 {
		lower, upper = upper, lower
	}
	return &Generic{
		first: lower,
		last:  upper,
		cmp:   cmp,
	}
}

func NewGenericFromOrder(first, last zed.Value, o order.Which) *Generic {
	return NewGeneric(first, last, expr.NewValueCompareFn(o, o == order.Asc))
}

func (g *Generic) In(val *zed.Value) bool {
	return g.cmp(val, &g.first) >= 0 && g.cmp(val, &g.last) <= 0
}

func (g *Generic) First() *zed.Value {
	return &g.first
}

func (g *Generic) Last() *zed.Value {
	return &g.last
}

func (g *Generic) After(val *zed.Value) bool {
	return g.cmp(val, &g.last) > 0
}

func (g *Generic) Before(val *zed.Value) bool {
	return g.cmp(val, &g.first) < 0
}

func (g *Generic) Overlaps(first, last *zed.Value) bool {
	if g.cmp(first, &g.first) >= 0 {
		return g.cmp(first, &g.last) <= 0
	}
	return g.cmp(last, &g.first) >= 0
}

func (g *Generic) Crop(s Span) bool {
	if first := s.First(); g.cmp(first, &g.first) > 0 {
		g.first = *first
	}
	if last := s.Last(); g.cmp(last, &g.last) < 0 {
		g.last = *last
	}
	return g.cmp(&g.first, &g.last) <= 0
}

func (g *Generic) Extend(val *zed.Value) {
	if g.cmp(val, &g.first) < 0 {
		g.first = *val.Copy()
	} else if g.cmp(val, &g.last) > 0 {
		g.last = *val.Copy()
	}
}

func (g *Generic) String() string {
	return Format(g)
}

func Format(s Span) string {
	first := zson.FormatValue(s.First())
	last := zson.FormatValue(s.Last())
	return fmt.Sprintf("first %s last %s", first, last)
}

func Overlaps(a, b Span) bool {
	return !b.Before(a.Last()) && !b.After(a.First())
}

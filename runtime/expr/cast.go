package expr

import (
	"math"
	"net/netip"
	"unicode/utf8"

	"github.com/araddon/dateparse"
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/byteconv"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/runtime/expr/coerce"
	"github.com/brimdata/zed/zson"
	"github.com/x448/float16"
)

func LookupPrimitiveCaster(zctx *zed.Context, typ zed.Type) Evaluator {
	switch typ {
	case zed.TypeBool:
		return &casterBool{zctx}
	case zed.TypeInt8:
		return &casterIntN{zctx, zed.TypeInt8, math.MinInt8, math.MaxInt8}
	case zed.TypeInt16:
		return &casterIntN{zctx, zed.TypeInt16, math.MinInt16, math.MaxInt16}
	case zed.TypeInt32:
		return &casterIntN{zctx, zed.TypeInt32, math.MinInt32, math.MaxInt32}
	case zed.TypeInt64:
		return &casterIntN{zctx, zed.TypeInt64, 0, 0}
	case zed.TypeUint8:
		return &casterUintN{zctx, zed.TypeUint8, math.MaxUint8}
	case zed.TypeUint16:
		return &casterUintN{zctx, zed.TypeUint16, math.MaxUint16}
	case zed.TypeUint32:
		return &casterUintN{zctx, zed.TypeUint32, math.MaxUint32}
	case zed.TypeUint64:
		return &casterUintN{zctx, zed.TypeUint64, 0}
	case zed.TypeFloat16:
		return &casterFloat16{zctx}
	case zed.TypeFloat32:
		return &casterFloat32{zctx}
	case zed.TypeFloat64:
		return &casterFloat64{zctx}
	case zed.TypeIP:
		return &casterIP{zctx}
	case zed.TypeNet:
		return &casterNet{zctx}
	case zed.TypeDuration:
		return &casterDuration{zctx}
	case zed.TypeTime:
		return &casterTime{zctx}
	case zed.TypeString:
		return &casterString{zctx}
	case zed.TypeBytes:
		return &casterBytes{}
	default:
		return nil
	}
}

type casterIntN struct {
	zctx *zed.Context
	typ  zed.Type
	min  int64
	max  int64
}

func (c *casterIntN) Eval(ectx Context, val *zed.Value) *zed.Value {
	v, ok := coerce.ToInt(val)
	if !ok || (c.min != 0 && (v < c.min || v > c.max)) {
		return c.zctx.WrapError("cannot cast to "+zson.FormatType(c.typ), val)
	}
	return ectx.CopyValue(*zed.NewInt(c.typ, v))
}

type casterUintN struct {
	zctx *zed.Context
	typ  zed.Type
	max  uint64
}

func (c *casterUintN) Eval(ectx Context, val *zed.Value) *zed.Value {
	v, ok := coerce.ToUint(val)
	if !ok || (c.max != 0 && v > c.max) {
		return c.zctx.WrapError("cannot cast to "+zson.FormatType(c.typ), val)
	}
	return ectx.CopyValue(*zed.NewUint(c.typ, v))
}

type casterBool struct {
	zctx *zed.Context
}

func (c *casterBool) Eval(ectx Context, val *zed.Value) *zed.Value {
	b, ok := coerce.ToBool(val)
	if !ok {
		return c.zctx.WrapError("cannot cast to bool", val)
	}
	return ectx.CopyValue(*zed.NewBool(b))
}

type casterFloat16 struct {
	zctx *zed.Context
}

func (c *casterFloat16) Eval(ectx Context, val *zed.Value) *zed.Value {
	f, ok := coerce.ToFloat(val)
	if !ok {
		return c.zctx.WrapError("cannot cast to float16", val)
	}
	f16 := float16.Fromfloat32(float32(f))
	return ectx.CopyValue(*zed.NewFloat16(f16.Float32()))
}

type casterFloat32 struct {
	zctx *zed.Context
}

func (c *casterFloat32) Eval(ectx Context, val *zed.Value) *zed.Value {
	f, ok := coerce.ToFloat(val)
	if !ok {
		return c.zctx.WrapError("cannot cast to float32", val)
	}
	return ectx.CopyValue(*zed.NewFloat32(float32(f)))
}

type casterFloat64 struct {
	zctx *zed.Context
}

func (c *casterFloat64) Eval(ectx Context, val *zed.Value) *zed.Value {
	f, ok := coerce.ToFloat(val)
	if !ok {
		return c.zctx.WrapError("cannot cast to float64", val)
	}
	return ectx.CopyValue(*zed.NewFloat64(f))
}

type casterIP struct {
	zctx *zed.Context
}

func (c *casterIP) Eval(ectx Context, val *zed.Value) *zed.Value {
	if _, ok := zed.TypeUnder(val.Type).(*zed.TypeOfIP); ok {
		return val
	}
	if !val.IsString() {
		return c.zctx.WrapError("cannot cast to ip", val)
	}
	ip, err := byteconv.ParseIP(val.Bytes())
	if err != nil {
		return c.zctx.WrapError("cannot cast to ip", val)
	}
	return ectx.NewValue(zed.TypeIP, zed.EncodeIP(ip))
}

type casterNet struct {
	zctx *zed.Context
}

func (c *casterNet) Eval(ectx Context, val *zed.Value) *zed.Value {
	if val.Type.ID() == zed.IDNet {
		return val
	}
	if !val.IsString() {
		return c.zctx.WrapError("cannot cast to net", val)
	}
	net, err := netip.ParsePrefix(string(val.Bytes()))
	if err != nil {
		return c.zctx.WrapError("cannot cast to net", val)
	}
	return ectx.NewValue(zed.TypeNet, zed.EncodeNet(net))
}

type casterDuration struct {
	zctx *zed.Context
}

func (c *casterDuration) Eval(ectx Context, val *zed.Value) *zed.Value {
	id := val.Type.ID()
	if id == zed.IDDuration {
		return val
	}
	if id == zed.IDString {
		d, err := nano.ParseDuration(byteconv.UnsafeString(val.Bytes()))
		if err != nil {
			f, ferr := byteconv.ParseFloat64(val.Bytes())
			if ferr != nil {
				return c.zctx.WrapError("cannot cast to duration", val)
			}
			d = nano.Duration(f)
		}
		return ectx.CopyValue(*zed.NewDuration(d))
	}
	if zed.IsFloat(id) {
		return ectx.CopyValue(*zed.NewDuration(nano.Duration(val.Float())))
	}
	v, ok := coerce.ToInt(val)
	if !ok {
		return c.zctx.WrapError("cannot cast to duration", val)
	}
	return ectx.CopyValue(*zed.NewDuration(nano.Duration(v)))
}

type casterTime struct {
	zctx *zed.Context
}

func (c *casterTime) Eval(ectx Context, val *zed.Value) *zed.Value {
	id := val.Type.ID()
	var ts nano.Ts
	switch {
	case id == zed.IDTime:
		return val
	case val.IsNull():
		// Do nothing. Any nil value is cast to a zero time.
	case id == zed.IDString:
		gotime, err := dateparse.ParseAny(byteconv.UnsafeString(val.Bytes()))
		if err != nil {
			v, err := byteconv.ParseFloat64(val.Bytes())
			if err != nil {
				return c.zctx.WrapError("cannot cast to time", val)
			}
			ts = nano.Ts(v)
		} else {
			ts = nano.Ts(gotime.UnixNano())
		}
	case zed.IsNumber(id):
		//XXX we call coerce on integers here to avoid unsigned/signed decode
		v, ok := coerce.ToInt(val)
		if !ok {
			return c.zctx.WrapError("cannot cast to time: coerce to int failed", val)
		}
		ts = nano.Ts(v)
	default:
		return c.zctx.WrapError("cannot cast to time", val)
	}
	return ectx.CopyValue(*zed.NewTime(ts))
}

type casterString struct {
	zctx *zed.Context
}

func (c *casterString) Eval(ectx Context, val *zed.Value) *zed.Value {
	id := val.Type.ID()
	if id == zed.IDBytes {
		if !utf8.Valid(val.Bytes()) {
			return c.zctx.WrapError("cannot cast to string: invalid UTF-8", val)
		}
		return ectx.NewValue(zed.TypeString, val.Bytes())
	}
	if enum, ok := val.Type.(*zed.TypeEnum); ok {
		selector := zed.DecodeUint(val.Bytes())
		symbol, err := enum.Symbol(int(selector))
		if err != nil {
			return ectx.CopyValue(*c.zctx.NewError(err))
		}
		return ectx.NewValue(zed.TypeString, zed.EncodeString(symbol))
	}
	if id == zed.IDString {
		// If it's already stringy, then the Zed encoding can stay
		// the same and we just update the stringy type.
		return ectx.NewValue(zed.TypeString, val.Bytes())
	}
	// Otherwise, we'll use a canonical ZSON value for the string rep
	// of an arbitrary value cast to a string.
	return ectx.NewValue(zed.TypeString, zed.EncodeString(zson.FormatValue(val)))
}

type casterBytes struct{}

func (c *casterBytes) Eval(ectx Context, val *zed.Value) *zed.Value {
	return ectx.NewValue(zed.TypeBytes, val.Bytes())
}

type casterNamedType struct {
	zctx *zed.Context
	expr Evaluator
	name string
}

func (c *casterNamedType) Eval(ectx Context, this *zed.Value) *zed.Value {
	val := c.expr.Eval(ectx, this)
	if val.IsError() {
		return val
	}
	typ, err := c.zctx.LookupTypeNamed(c.name, zed.TypeUnder(val.Type))
	if err != nil {
		return ectx.CopyValue(*c.zctx.NewError(err))
	}
	return ectx.NewValue(typ, val.Bytes())
}

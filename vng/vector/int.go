package vector

import (
	"io"

	"github.com/brimdata/zed"
)

type Int64Writer struct {
	PrimitiveWriter
}

func NewInt64Writer(spiller *Spiller) *Int64Writer {
	return &Int64Writer{*NewPrimitiveWriter(zed.TypeInt64, spiller, false)}
}

func (p *Int64Writer) Write(v int64) error {
	return p.PrimitiveWriter.Write(zed.EncodeInt(v))
}

type Int64Reader struct {
	PrimitiveReader
}

func NewInt64Reader(segmap []Segment, r io.ReaderAt) *Int64Reader {
	return &Int64Reader{*NewPrimitiveReader(&Primitive{Typ: zed.TypeInt64, Segmap: segmap}, r)}
}

func (p *Int64Reader) Read() (int64, error) {
	zv, err := p.ReadBytes()
	if err != nil {
		return 0, err
	}
	return zed.DecodeInt(zv), err
}

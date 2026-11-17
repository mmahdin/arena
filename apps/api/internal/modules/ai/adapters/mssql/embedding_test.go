package mssql

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestEmbeddingStorageCodec(t *testing.T) {
	raw, err := EncodeEmbedding("synthetic", []float32{3, 4})
	if err != nil {
		t.Fatal(err)
	}
	v, err := DecodeEmbedding(1, "synthetic", "synthetic", 2, 2, raw)
	if err != nil || math.Abs(float64(v[0])-.6) > 1e-6 || math.Abs(float64(v[1])-.8) > 1e-6 {
		t.Fatal(v, err)
	}
	for _, tc := range []struct {
		version, dim, want int
		model              string
		raw                []byte
	}{{2, 2, 2, "synthetic", raw}, {1, 2, 3, "synthetic", raw}, {1, 2, 2, "other", raw}, {1, 2, 2, "synthetic", raw[:4]}, {1, 1, 1, "synthetic", make([]byte, 4)}} {
		if _, err = DecodeEmbedding(tc.version, tc.model, "synthetic", tc.dim, tc.want, tc.raw); err == nil {
			t.Fatal("bad metadata/vector accepted")
		}
	}
	binary.LittleEndian.PutUint32(raw, math.Float32bits(float32(math.NaN())))
	if _, err = DecodeEmbedding(1, "synthetic", "synthetic", 2, 2, raw); err == nil {
		t.Fatal("NaN accepted")
	}
	for _, v := range [][]float32{nil, {0, 0}, {float32(math.Inf(1))}, {float32(math.NaN())}, make([]float32, 8193)} {
		if _, err = EncodeEmbedding("synthetic", v); err == nil {
			t.Fatal("invalid vector encoded")
		}
	}
}

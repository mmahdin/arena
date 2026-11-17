package mssql

import (
	"encoding/binary"
	"errors"
	"math"
)

// EmbeddingV1 is the SQL storage codec: little-endian IEEE754 float32,
// unit-normalized, with model ID and dimensions stored beside the bytes.
const EmbeddingV1 = 1

var errEmbedding = errors.New("invalid embedding metadata or vector")

func EncodeEmbedding(model string, vector []float32) ([]byte, error) {
	if model == "" || len(model) > 120 || len(vector) < 1 || len(vector) > 8192 {
		return nil, errEmbedding
	}
	var sum float64
	for _, v := range vector {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return nil, errEmbedding
		}
		sum += float64(v) * float64(v)
	}
	if sum == 0 {
		return nil, errEmbedding
	}
	scale := math.Sqrt(sum)
	out := make([]byte, len(vector)*4)
	for i, v := range vector {
		binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(float32(float64(v)/scale)))
	}
	return out, nil
}
func DecodeEmbedding(version int, model, wantModel string, dim, wantDim int, raw []byte) ([]float32, error) {
	if version != EmbeddingV1 || model == "" || model != wantModel || dim != wantDim || dim < 1 || dim > 8192 || len(raw) != dim*4 {
		return nil, errEmbedding
	}
	out := make([]float32, dim)
	var sum float64
	for i := range out {
		v := math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return nil, errEmbedding
		}
		sum += float64(v) * float64(v)
		out[i] = v
	}
	if math.Abs(sum-1) > 0.0001 {
		return nil, errEmbedding
	}
	return out, nil
}

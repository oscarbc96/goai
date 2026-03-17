package bench

import (
	"testing"

	"github.com/zendev-sh/goai"
)

type BenchmarkProduct struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	InStock     bool     `json:"in_stock"`
}

type BenchmarkOrder struct {
	ID       string             `json:"id"`
	Customer BenchmarkCustomer  `json:"customer"`
	Items    []BenchmarkProduct `json:"items"`
	Total    float64            `json:"total"`
	Status   string             `json:"status"`
}

type BenchmarkCustomer struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Address string `json:"address"`
}

// BenchmarkSchemaSimple measures schema generation for a simple struct.
func BenchmarkSchemaSimple(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = goai.SchemaFrom[BenchmarkProduct]()
	}
}

// BenchmarkSchemaComplex measures schema generation for a nested struct.
func BenchmarkSchemaComplex(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = goai.SchemaFrom[BenchmarkOrder]()
	}
}

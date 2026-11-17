package mymiddleware

import (
	"testing"
)

func TestValidatorValidateAppliesDefaults(t *testing.T) {
	p := &struct {
		Name   string `default:"haha"`
		Age    int    `default:"17"`
		Weight int    `default:"50"`
	}{}

	v := &Validator{}
	if err := v.Validate(p); err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if p.Name != "haha" || p.Age != 17 || p.Weight != 50 {
		t.Error("绑定默认值失败")
	}
}

//func BenchmarkBrotliWithConfig(b *testing.B) {
//	for n := 0; n < b.N; n++ {
//		brotli.NewWriterOptions(nil, brotli.WriterOptions{Quality: 11})
//	}
//}

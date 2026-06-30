package fisheryates

import "testing"

// BenchmarkGetLogger measures the cost of the lock-free atomic read path
// used by GetLogger.
func BenchmarkGetLogger(b *testing.B) {
	SetLogger(nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetLogger()
	}
}

// BenchmarkGetTranslator measures the cost of the lock-free atomic read
// path used by GetTranslator.
func BenchmarkGetTranslator(b *testing.B) {
	SetTranslator(nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetTranslator()
	}
}

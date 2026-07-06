package fourway

import "testing"

func BenchmarkFrameworkPlaintext(b *testing.B) {
	b.Run("kern", benchmarkKernPlaintext)
	b.Run("mach", benchmarkMachPlaintext)
	b.Run("chi", benchmarkChiPlaintext)
	b.Run("gin", benchmarkGinPlaintext)
	b.Run("fiber", benchmarkFiberPlaintext)
	b.Run("fasthttp", benchmarkFastHTTPPlaintext)
}

func BenchmarkFrameworkPlaintextMiddleware(b *testing.B) {
	b.Run("kern", benchmarkKernPlaintextMiddleware)
	b.Run("mach", benchmarkMachPlaintextMiddleware)
	b.Run("chi", benchmarkChiPlaintextMiddleware)
	b.Run("gin", benchmarkGinPlaintextMiddleware)
	b.Run("fiber", benchmarkFiberPlaintextMiddleware)
	b.Run("fasthttp", benchmarkFastHTTPPlaintextMiddleware)
}

func BenchmarkFrameworkQueryAccess(b *testing.B) {
	b.Run("kern", benchmarkKernQueryAccess)
	b.Run("mach", benchmarkMachQueryAccess)
	b.Run("chi", benchmarkChiQueryAccess)
	b.Run("gin", benchmarkGinQueryAccess)
	b.Run("fiber", benchmarkFiberQueryAccess)
	b.Run("fasthttp", benchmarkFastHTTPQueryAccess)
}

func BenchmarkFrameworkDecodeJSON(b *testing.B) {
	b.Run("kern", benchmarkKernDecodeJSON)
	b.Run("mach", benchmarkMachDecodeJSON)
	b.Run("chi", benchmarkChiDecodeJSON)
	b.Run("gin", benchmarkGinDecodeJSON)
	b.Run("fiber", benchmarkFiberDecodeJSON)
	b.Run("fasthttp", benchmarkFastHTTPDecodeJSON)
}

func BenchmarkFrameworkPathParams(b *testing.B) {
	b.Run("kern", benchmarkKernPathParams)
	b.Run("mach", benchmarkMachPathParams)
	b.Run("chi", benchmarkChiPathParams)
	b.Run("gin", benchmarkGinPathParams)
	b.Run("fiber", benchmarkFiberPathParams)
	b.Run("fasthttp", benchmarkFastHTTPPathParams)
}

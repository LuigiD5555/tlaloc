package groundingautomaton

import "testing"

func BenchmarkVerify(b *testing.B) {
	in := VerifyInput{
		ModelAnswer: "The system does not distribute the load between three agents.",
		PageContent: "The system distributes the load between three agents. A second subsystem coordinates two workers locally.",
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Verify(in)
	}
}

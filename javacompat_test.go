package fisheryates

import (
	"sort"
	"testing"

	"github.com/0verkilll/f5prng"
	"github.com/0verkilll/sha1"
)

// newSeededSHA1PRNG builds a fresh SecureRandom seeded with `seed`. Tests
// share this so every assertion runs against the same Java-compat byte
// stream that production code uses.
func newSeededSHA1PRNG(t *testing.T, seed []byte) f5prng.RandomSource {
	t.Helper()
	hasher := sha1.NewSHA1(sha1.NewBigEndian())
	prng := f5prng.NewSecureRandom(hasher)
	if err := prng.Seed(seed); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return prng
}

// TestJavaCompat_IsValidPermutation: output must contain each value in [0, size)
// exactly once. This is the structural correctness check — independent of
// what byte stream Java produced, the result still has to be a permutation.
func TestJavaCompat_IsValidPermutation(t *testing.T) {
	jc := NewJavaCompat()
	for _, size := range []int{1, 2, 8, 64, 1000} {
		prng := newSeededSHA1PRNG(t, []byte("test-seed"))
		perm, err := jc.Generate(size, prng)
		if err != nil {
			t.Fatalf("size=%d: %v", size, err)
		}
		if len(perm) != size {
			t.Fatalf("size=%d: got %d elements", size, len(perm))
		}
		sorted := append([]int(nil), perm...)
		sort.Ints(sorted)
		for i, v := range sorted {
			if v != i {
				t.Fatalf("size=%d: element %d is %d (not a valid permutation)", size, i, v)
			}
		}
	}
}

// TestJavaCompat_Deterministic: same seed must always produce the same
// permutation. This is the F5 invariant — embed and extract must agree on
// the shuffle order, and that only works if the permutation is a pure
// function of the seed.
func TestJavaCompat_Deterministic(t *testing.T) {
	jc := NewJavaCompat()
	const size = 500

	a, err := jc.Generate(size, newSeededSHA1PRNG(t, []byte("23")))
	if err != nil {
		t.Fatal(err)
	}
	b, err := jc.Generate(size, newSeededSHA1PRNG(t, []byte("23")))
	if err != nil {
		t.Fatal(err)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("non-deterministic at %d: %d vs %d", i, a[i], b[i])
		}
	}
}

// TestJavaCompat_DiffersFromUnbiased: the whole point of having both
// permutators is that they produce different orderings on the same seed.
// If this test ever fails we've broken the divergence and should re-check
// the algorithm. (Pinning byte-identical output against a Java oracle is
// what the integration test against sample.jpg does — we don't try to
// reproduce that oracle here in unit tests.)
//
// Since v2 [JavaCompat] is an alias for [FisherYates], so we compare
// against [Unbiased] (the rejection-sampled implementation).
func TestJavaCompat_DiffersFromUnbiased(t *testing.T) {
	const size = 1000
	javaPerm, err := NewJavaCompat().Generate(size, newSeededSHA1PRNG(t, []byte("compat-test")))
	if err != nil {
		t.Fatal(err)
	}
	unbiasedPerm, err := NewUnbiased().Generate(size, newSeededSHA1PRNG(t, []byte("compat-test")))
	if err != nil {
		t.Fatal(err)
	}

	same := 0
	for i := range javaPerm {
		if javaPerm[i] == unbiasedPerm[i] {
			same++
		}
	}
	if same == size {
		t.Fatal("JavaCompat and Unbiased produced identical permutations; one of them is not implementing its documented algorithm")
	}
}

// TestJavaCompat_GenerateInto_MatchesGenerate: GenerateInto with an empty
// buffer must produce the same permutation as Generate. This is the same
// contract the unbiased FisherYates honors (see fisheryates_test.go's
// TestFisherYates_GenerateInto_MatchesGenerate).
func TestJavaCompat_GenerateInto_MatchesGenerate(t *testing.T) {
	jc := NewJavaCompat()
	const size = 100

	want, err := jc.Generate(size, newSeededSHA1PRNG(t, []byte("seed")))
	if err != nil {
		t.Fatal(err)
	}
	got, err := jc.GenerateInto(nil, size, newSeededSHA1PRNG(t, []byte("seed")))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("length mismatch: %d vs %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mismatch at %d: %d vs %d", i, got[i], want[i])
		}
	}
}

// TestJavaCompat_GenerateInto_ReusesBuffer: GenerateInto with a sufficiently-
// large buffer should reuse it (no fresh allocation). We check this by
// comparing the underlying array address before and after.
func TestJavaCompat_GenerateInto_ReusesBuffer(t *testing.T) {
	jc := NewJavaCompat()
	buf := make([]int, 0, 256)
	bufAddr := &buf[:1][0]

	out, err := jc.GenerateInto(buf, 200, newSeededSHA1PRNG(t, []byte("buf-test")))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 200 {
		t.Fatalf("got %d, want 200", len(out))
	}
	if &out[:1][0] != bufAddr {
		t.Fatalf("GenerateInto allocated a new buffer instead of reusing the supplied one")
	}
}

// TestJavaCompat_RejectsInvalidSize: out-of-range sizes are rejected
// the same way as the unbiased implementation.
func TestJavaCompat_RejectsInvalidSize(t *testing.T) {
	jc := NewJavaCompat()
	prng := newSeededSHA1PRNG(t, []byte("x"))

	if _, err := jc.Generate(-1, prng); err == nil {
		t.Fatal("size=-1 should error")
	}
	if _, err := jc.Generate(MaxPermutationSize+1, prng); err == nil {
		t.Fatal("size=MaxPermutationSize+1 should error")
	}
	if perm, err := jc.Generate(0, prng); err != nil || len(perm) != 0 {
		t.Fatalf("size=0 should return empty permutation, got len=%d err=%v", len(perm), err)
	}
}

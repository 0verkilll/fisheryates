package fisheryates

// JavaCompat is a deprecated alias for [FisherYates], retained for source
// compatibility with v1.x code that explicitly opted into Java-biased
// shuffling via [NewJavaCompat] when the default was rejection-sampled.
//
// As of v2, [FisherYates] is itself the Java-biased default, so JavaCompat
// is now redundant. New code should use [FisherYates] / [NewFisherYates]
// directly.
//
// Deprecated: use [FisherYates] (the package default since v2).
type JavaCompat = FisherYates

// NewJavaCompat returns a Java-bias-compatible permutator. It is identical
// to [NewFisherYates] in v2 and later.
//
// Deprecated: use [NewFisherYates]. JavaCompat is retained only so existing
// call sites continue to compile.
func NewJavaCompat() Permutator {
	return NewFisherYates()
}

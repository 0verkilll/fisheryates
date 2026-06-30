// Package testing provides test utilities, mocks, and helpers for testing
// Fisher-Yates implementations.
package testing

// MockRandomSource provides a deterministic random source for testing.
// It cycles through a predefined set of int32 values, making tests reproducible.
//
// This mock implements the f5prng.RandomSource interface and is useful for:
//   - Testing specific permutation sequences
//   - Verifying negative int32 handling
//   - Creating deterministic test scenarios
//
// Example usage:
//
//	mock := NewMockRandomSource(5, -10, 3, -1, 0)
//	fy := fisheryates.NewFisherYates()
//	perm := fy.Generate(5, mock)
//	// perm will be generated using values [5, -10, 3, -1, 0] in sequence
type MockRandomSource struct {
	values []int32 // Predefined values to return
	index  int     // Current position in values slice
}

// NewMockRandomSource creates a new MockRandomSource with the specified values.
// The mock will cycle through these values repeatedly when NextInt() is called.
//
// Parameters:
//   - values: The int32 values to return in sequence (cycles if exhausted)
//
// Returns:
//   - A MockRandomSource ready for use in tests
//
// Example:
//
//	// Create mock that returns specific values
//	mock := NewMockRandomSource(-2147483648, 0, 2147483647)
//
//	// Use in permutation generation
//	perm := fy.Generate(10, mock)
func NewMockRandomSource(values ...int32) *MockRandomSource {
	if len(values) == 0 {
		values = []int32{0} // Default to zero if no values provided
	}
	return &MockRandomSource{
		values: values,
		index:  0,
	}
}

// NextInt returns the next int32 value from the predefined sequence.
// The sequence cycles back to the beginning when exhausted.
//
// Returns:
//   - The next int32 value in the sequence
func (m *MockRandomSource) NextInt() int32 {
	val := m.values[m.index]
	m.index = (m.index + 1) % len(m.values)
	return val
}

// Seed is a no-op for MockRandomSource since it provides deterministic values.
// This method exists to satisfy the RandomSource interface.
//
// Parameters:
//   - seed: Ignored (not used in deterministic mock)
//
// Returns:
//   - Always nil (mock cannot fail to seed)
func (m *MockRandomSource) Seed(_ []byte) error {
	// No-op: mock provides deterministic values regardless of seed
	_ = m // Explicitly mark as no-op for coverage
	return nil
}

// NextBytes returns a byte slice of the specified length.
// All bytes are zero for simplicity in this mock implementation.
//
// Parameters:
//   - n: Number of bytes to generate
//
// Returns:
//   - A byte slice of length n filled with zeros
func (m *MockRandomSource) NextBytes(n int) []byte {
	return make([]byte, n)
}

// Clear is a no-op for MockRandomSource since it holds no sensitive state.
// This method exists to satisfy the f5prng.RandomSource interface.
func (m *MockRandomSource) Clear() {
	// No-op: mock has no sensitive state to clear
	// Reset index for convenience (allows reuse after Clear)
	m.index = 0
}

// Reset resets the mock's internal index to the beginning of the value sequence.
// This is useful when you want to repeat the same sequence in multiple tests.
//
// Example:
//
//	mock := NewMockRandomSource(1, 2, 3)
//	perm1 := fy.Generate(5, mock) // Uses 1, 2, 3, 1, 2
//	mock.Reset()
//	perm2 := fy.Generate(5, mock) // Uses 1, 2, 3, 1, 2 (same sequence)
func (m *MockRandomSource) Reset() {
	m.index = 0
}

// SetValues replaces the current value sequence with a new one and resets the index.
// This allows reusing the same mock instance with different test data.
//
// Parameters:
//   - values: New int32 values to use in the sequence
//
// Example:
//
//	mock := NewMockRandomSource(1, 2, 3)
//	perm1 := fy.Generate(3, mock)
//
//	mock.SetValues(10, 20, 30) // Change values for next test
//	perm2 := fy.Generate(3, mock)
func (m *MockRandomSource) SetValues(values ...int32) {
	if len(values) == 0 {
		values = []int32{0}
	}
	m.values = values
	m.index = 0
}

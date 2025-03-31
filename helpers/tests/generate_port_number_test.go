package tests

import (
	"testing"

	"github.com/LajnaLegenden/transpiler4/helpers"
)

func TestGeneratePortNumber(t *testing.T) {
	// Test that the port number is within the expected range
	port := helpers.GeneratePortNumber()

	// The port should be between 1000 and 99999 (inclusive)
	if port < 1000 || port > 99999 {
		t.Errorf("Generated port %d is outside the expected range [1000, 99999]", port)
	}

	// Test that the function generates different numbers on consecutive calls
	// This test could theoretically fail if we get the same random number twice,
	// but the probability of that is extremely low
	anotherPort := helpers.GeneratePortNumber()
	if port == anotherPort {
		t.Errorf("Generated the same port number (%d) on consecutive calls, which is highly unlikely", port)
	}
}

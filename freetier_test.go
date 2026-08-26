package main

import "testing"

func TestIsCacheHit(t *testing.T) {
	tests := []struct {
		cacheStatus string
		expected    bool
	}{
		{"hit", true},
		{"stale", true},
		{"updating", true},
		{"revalidated", true},
		{"ignored", false},
		{"deferred_hit", true},
		{"revalidated_hit", true},
		{"miss", false},
		{"expired", false},
		{"dynamic", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.cacheStatus, func(t *testing.T) {
			result := isCacheHit(tt.cacheStatus)
			if result != tt.expected {
				t.Errorf("isCacheHit(%q) = %v, want %v", tt.cacheStatus, result, tt.expected)
			}
		})
	}
}

func TestIsSSLEncrypted(t *testing.T) {
	tests := []struct {
		clientSSLProtocol string
		expected          bool
	}{
		{"TLSv1.3", true},
		{"TLSv1.2", true},
		{"none", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.clientSSLProtocol, func(t *testing.T) {
			result := isSSLEncrypted(tt.clientSSLProtocol)
			if result != tt.expected {
				t.Errorf("isSSLEncrypted(%q) = %v, want %v", tt.clientSSLProtocol, result, tt.expected)
			}
		})
	}
}

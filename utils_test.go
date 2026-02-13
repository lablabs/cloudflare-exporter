package main

import "testing"

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"/", "/"},
		{"/health", "/health"},
		{"/api/v1/users", "/api/v1/users"},
		{"/users/123", "/users/:id"},
		{"/users/123/orders/456", "/users/:id/orders/:id"},
		{"/orders/550e8400-e29b-41d4-a716-446655440000", "/orders/:uuid"},
		{"/items/5f3a2b1c9d", "/items/:id"},
		{"/search?q=test&page=1", "/search"},
		{"/api/v1/users/123?include=orders", "/api/v1/users/:id"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

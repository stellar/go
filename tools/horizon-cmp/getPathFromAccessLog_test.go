package main

import "testing"

func TestGetPathFromAccessLog(t *testing.T) {
	validLine := "127.0.0.1 - - [10/Oct/2000:13:55:36 -0700] \"GET /page.html HTTP/1.0\" 200 2326"
	invalidLengthLine := "127.0.0.1 - - [10/Oct/2000:13:55:36 -0700] \"GET /page.html HTTP/1.0\" 200"
	invalidPathLine := "127.0.0.1 - - [10/Oct/2000:13:55:36 -0700] \"GET //page.html HTTP/1.0\" 200 2326"

	expectedPath := "/page.html"

	// Test valid line
	path, err := getPathFromAccessLog(validLine)
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
	if path != expectedPath {
		t.Errorf("Expected path %q, but got %q", expectedPath, path)
	}

	// Test invalid length line
	_, err = getPathFromAccessLog(invalidLengthLine)
	if err == nil {
		t.Error("Expected error, but got nil")
	}

	// Test invalid path line
	_, err = getPathFromAccessLog(invalidPathLine)
	if err == nil {
		t.Error("Expected error, but got nil")
	}
}

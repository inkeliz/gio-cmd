package main

import (
	"reflect"
	"testing"
)

func TestParseLibraryNameVersion(t *testing.T) {
	tests := []struct {
		filename        string
		expectedName    string
		expectedVersion string
	}{
		{"lib.aar", "lib", ""},
		{"lib-1.0.aar", "lib", "1.0"},
		{"lib-1.0.0.aar", "lib", "1.0.0"},
		{"lib-1.0.0-rc1.aar", "lib", "1.0.0-rc1"},
		{"library-name-1.2.3.aar", "library-name", "1.2.3"},
		{"my-lib-v1.0.aar", "my-lib", "v1.0"},
		{"complex-name-with-dashes-1.0.aar", "complex-name-with-dashes", "1.0"},
		{"noversion.jar", "noversion", ""},
		{"lib-2.aar", "lib", "2"},
		// Edge cases
		{"lib-.aar", "lib-", ""},
		// Path-based identity for classes.jar
		{"play-services-base-18.1.0/classes.jar", "play-services-base", "18.1.0"},
		{"androidx.annotation/classes.jar", "androidx.annotation", ""},
		{"vendors/android/jar/classes.jar", "classes", ""},
	}

	for _, tt := range tests {
		name, version := parseLibraryNameVersion(tt.filename)
		if name != tt.expectedName || version != tt.expectedVersion {
			t.Errorf("parseLibraryNameVersion(%q) = (%q, %q), want (%q, %q)",
				tt.filename, name, version, tt.expectedName, tt.expectedVersion)
		}
	}
}

func TestDeduplicateLibraries(t *testing.T) {
	tests := []struct {
		name     string
		libs     []string
		expected []string
	}{
		{
			name:     "No duplicates",
			libs:     []string{"libA-1.0.aar", "libB-2.0.aar"},
			expected: []string{"libA-1.0.aar", "libB-2.0.aar"},
		},
		{
			name:     "Simple duplicates",
			libs:     []string{"libA-1.0.aar", "libA-2.0.aar"},
			expected: []string{"libA-2.0.aar"},
		},
		{
			name:     "Duplicates with mixed extensions",
			libs:     []string{"libA-1.0.aar", "libA-1.0.jar"},
			expected: []string{"libA-1.0.jar"}, // Should be deterministic, probably sort by filename if versions equal
		},
		{
			name:     "Complex versions",
			libs:     []string{"libA-1.0.0.aar", "libA-1.0.1.aar", "libA-1.0.0-rc1.aar"},
			expected: []string{"libA-1.0.1.aar"},
		},
		{
			name:     "Version vs No Version",
			libs:     []string{"libA.aar", "libA-1.0.aar"},
			expected: []string{"libA-1.0.aar"}, // Versioned should generally win over non-versioned if we treat "" as 0.0.0 or similar, or just lexically
		},
		{
			name:     "Multiple groups",
			libs:     []string{"libA-1.0.aar", "libB-1.0.aar", "libA-2.0.aar", "libB-0.5.aar"},
			expected: []string{"libA-2.0.aar", "libB-1.0.aar"},
		},
		{
			name:     "Annotation vs Annotation-JVM (User report)",
			libs:     []string{"annotation-1.5.0.jar", "annotation-jvm-1.7.0.jar"},
			expected: []string{"annotation-jvm-1.7.0.jar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deduplicateLibraries(tt.libs)
			// check set equality or sorted equality
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("deduplicateLibraries(%v) = %v, want %v", tt.libs, got, tt.expected)
			}
		})
	}
}

package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
)

// parseLibraryNameVersion parses a path to a library and returns the library name and version.
// If the filename is generic (like 'classes.jar'), it attempts to use the parent directory as the library name.
// Examples:
//
//		lib-1.0.aar -> name: lib, version: 1.0
//		foo-bar-1.2.3.jar -> name: foo-bar, version: 1.2.3
//		mylib.aar -> name: mylib, version: ""
//		lib-1.0.0-rc1.aar -> name: lib, version: 1.0.0-rc1
//	 .../play-services-base/classes.jar -> name: play-services-base, version: ""
func parseLibraryNameVersion(path string) (name, version string) {
	filename := filepath.Base(path)
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	if base == "classes" && ext == ".jar" {
		// Generic Android library name. Try to use the parent directory.
		parent := filepath.Dir(path)
		parentName := filepath.Base(parent)
		if parentName != "." && parentName != "jar" && parentName != "libs" && parentName != "lib" {
			base = parentName
		}
	}

	// We look for the longest suffix that is a valid semver.
	// Split by hyphens.
	parts := strings.Split(base, "-")

	bestSplit := -1

	// Iterate from the first hyphen to the last.
	for i := 1; i < len(parts); i++ {
		suffix := strings.Join(parts[i:], "-")

		check := suffix
		if !strings.HasPrefix(check, "v") {
			check = "v" + check
		}

		if semver.IsValid(check) {
			if bestSplit == -1 {
				bestSplit = i
			}
			break
		}
	}

	if bestSplit != -1 {
		name = strings.Join(parts[:bestSplit], "-")
		version = strings.Join(parts[bestSplit:], "-")
		return name, version
	}

	return base, ""
}

// normalizeLibraryName strips known platform/variant suffixes from the library name.
// This allows grouping variants like 'annotation' and 'annotation-jvm' together.
func normalizeLibraryName(name string) string {
	suffixes := []string{
		"-jvm",
		"-android",
		"-jvm-linux-x64",
		"-jvm-linux-arm64",
		"-jvm-macos-x64",
		"-jvm-macos-arm64",
		"-jvm-windows-x64",
		"-jvm-windows-arm64",
	}
	for _, s := range suffixes {
		if strings.HasSuffix(name, s) {
			return strings.TrimSuffix(name, s)
		}
	}
	return name
}

// deduplicateLibraries takes a list of library paths, detects duplicates based on name,
// selects the highest version, and returns a sorted list of unique libraries.
func deduplicateLibraries(libs []string) []string {
	type libInfo struct {
		path    string
		version string
		base    string // filename
	}

	grouped := make(map[string][]libInfo)

	for _, lib := range libs {
		name, version := parseLibraryNameVersion(lib)
		normalizedName := normalizeLibraryName(name)
		grouped[normalizedName] = append(grouped[normalizedName], libInfo{
			path:    lib,
			version: version,
			base:    filepath.Base(lib),
		})
	}

	var result []string

	for name, infos := range grouped {
		if len(infos) == 1 {
			result = append(result, infos[0].path)
			continue
		}

		// Sort by version descending.
		sort.Slice(infos, func(i, j int) bool {
			v1 := infos[i].version
			v2 := infos[j].version

			// Normalize for semver comparison
			sv1 := v1
			if !strings.HasPrefix(sv1, "v") {
				sv1 = "v" + sv1
			}
			sv2 := v2
			if !strings.HasPrefix(sv2, "v") {
				sv2 = "v" + sv2
			}

			// Compare returns 0 if both invalid, which is fine, we fall back to string comparison below?
			// Actually semver.Compare puts invalid versions less than valid ones.
			// If both are invalid, result is 0.

			cmp := semver.Compare(sv1, sv2)
			if cmp != 0 {
				return cmp > 0 // Descending
			}

			// If semver compare is equal (or both invalid), fall back to string comparison of raw version
			if v1 != v2 {
				return v1 > v2
			}

			// If versions are identical (e.g. same version but different extension or path),
			// sort by filename to be deterministic.
			return infos[i].base > infos[j].base
		})

		// Warn about duplicates
		best := infos[0]
		fmt.Printf("WARNING: Duplicate libraries found for %q. Choosing %s.\n", name, best.base)
		for _, info := range infos[1:] {
			fmt.Printf("         Ignored: %s\n", info.base)
		}

		result = append(result, best.path)
	}

	// Sort result to be deterministic
	sort.Strings(result)
	return result
}

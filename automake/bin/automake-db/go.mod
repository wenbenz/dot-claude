module automake-db

go 1.23

require modernc.org/sqlite v1.34.4

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/sys v0.22.0 // indirect
	modernc.org/libc v1.55.3 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
)

// The replace directives below pin modernc.org/sqlite's dependency tree
// straight to its git hosts (gitlab.com/cznic, github.com/golang) instead of
// resolving through each package's modernc.org/golang.org vanity redirect.
// Harmless either way; needed in network environments that block vanity
// go-get lookups or module-proxy blob storage but allow direct git access
// to GitHub/GitLab. `go mod tidy` will happily collapse these away on a
// machine with unrestricted access to modernc.org.
replace modernc.org/sqlite => gitlab.com/cznic/sqlite v1.34.4

replace golang.org/x/sys => github.com/golang/sys v0.22.0

replace modernc.org/gc/v3 => gitlab.com/cznic/gc/v3 v3.0.0-20240107210532-573471604cb6

replace modernc.org/libc => gitlab.com/cznic/libc v1.55.3

replace modernc.org/fileutil => gitlab.com/cznic/fileutil v1.3.0

replace modernc.org/mathutil => gitlab.com/cznic/mathutil v1.6.0

replace modernc.org/strutil => gitlab.com/cznic/strutil v1.2.0

replace modernc.org/token => gitlab.com/cznic/token v1.1.0

replace modernc.org/memory => gitlab.com/cznic/memory v1.8.0

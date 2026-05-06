module github.com/anchore/go-logger/adapter/logrus

go 1.25.0

require (
	github.com/anchore/go-logger v0.0.0-00010101000000-000000000000
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d
	github.com/sirupsen/logrus v1.9.4
	github.com/stretchr/testify v1.11.1
	golang.org/x/term v0.42.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Local development: resolve the parent module from the working tree so
// changes to root and adapter advance together. The release process replaces
// this with a normal tagged require before tagging adapter releases.
replace github.com/anchore/go-logger => ../..

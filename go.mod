module github.com/stkali/utility

go 1.18

require github.com/stretchr/testify v1.7.0

require (
	github.com/davecgh/go-spew v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)

retract (
	v1.2.2  // not compatible with go versions 1.18, 1.19, and 1.20
)

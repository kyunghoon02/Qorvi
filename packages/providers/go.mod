module github.com/flowintel/flowintel/packages/providers

go 1.24.4

require (
	github.com/flowintel/flowintel/packages/config v0.0.0
	github.com/flowintel/flowintel/packages/domain v0.0.0
)

replace github.com/flowintel/flowintel/packages/config => ../config

replace github.com/flowintel/flowintel/packages/domain => ../domain

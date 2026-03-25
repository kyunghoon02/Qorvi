module github.com/flowintel/flowintel/apps/workers

go 1.24.4

require (
	github.com/flowintel/flowintel/packages/config v0.0.0
	github.com/flowintel/flowintel/packages/db v0.0.0
	github.com/flowintel/flowintel/packages/providers v0.0.0
)

replace github.com/flowintel/flowintel/packages/config => ../../packages/config

replace github.com/flowintel/flowintel/packages/db => ../../packages/db

replace github.com/flowintel/flowintel/packages/providers => ../../packages/providers

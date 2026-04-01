module github.com/qorvi/qorvi/apps/workers

go 1.24.4

require (
	github.com/qorvi/qorvi/packages/config v0.0.0
	github.com/qorvi/qorvi/packages/db v0.0.0
	github.com/qorvi/qorvi/packages/providers v0.0.0
)

replace github.com/qorvi/qorvi/packages/config => ../../packages/config

replace github.com/qorvi/qorvi/packages/db => ../../packages/db

replace github.com/qorvi/qorvi/packages/providers => ../../packages/providers

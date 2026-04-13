module github.com/qorvi/qorvi/packages/providers

go 1.24.4

require (
	github.com/qorvi/qorvi/packages/config v0.0.0
	github.com/qorvi/qorvi/packages/domain v0.0.0
)

replace github.com/qorvi/qorvi/packages/config => ../config

replace github.com/qorvi/qorvi/packages/domain => ../domain

module github.com/whalegraph/whalegraph/packages/providers

go 1.24.4

require (
	github.com/whalegraph/whalegraph/packages/config v0.0.0
	github.com/whalegraph/whalegraph/packages/domain v0.0.0
)

replace github.com/whalegraph/whalegraph/packages/config => ../config

replace github.com/whalegraph/whalegraph/packages/domain => ../domain

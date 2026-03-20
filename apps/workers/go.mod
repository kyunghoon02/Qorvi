module github.com/whalegraph/whalegraph/apps/workers

go 1.24.4

require (
	github.com/whalegraph/whalegraph/packages/config v0.0.0
	github.com/whalegraph/whalegraph/packages/db v0.0.0
	github.com/whalegraph/whalegraph/packages/providers v0.0.0
)

replace github.com/whalegraph/whalegraph/packages/config => ../../packages/config

replace github.com/whalegraph/whalegraph/packages/db => ../../packages/db

replace github.com/whalegraph/whalegraph/packages/providers => ../../packages/providers

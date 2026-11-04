module github.com/fuvito/api-workbench/apps/cli

go 1.26.0

require (
	github.com/fuvito/api-workbench/packages/core v0.0.0
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/PaesslerAG/gval v1.0.0 // indirect
	github.com/PaesslerAG/jsonpath v0.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/fuvito/api-workbench/packages/core => ../../packages/core

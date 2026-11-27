module github.com/stevetosak/pult

go 1.27.0

toolchain go1.27.4

require (
	go.yaml.in/yaml/v4 v4.0.0
	golang.org/x/sys v0.50.0
)

require (
	golang.org/x/mobile v0.0.0-20261117202452-47699113848a // indirect
	golang.org/x/mod v0.43.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/tools v0.53.0 // indirect
)

tool (
	golang.org/x/mobile/cmd/gobind
	golang.org/x/mobile/cmd/gomobile
)

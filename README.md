# Proteus Consul Key-Value Provider
![Coverage](https://img.shields.io/badge/Coverage-58.2%25-yellow)
[![Go Report Card](https://goreportcard.com/badge/github.com/simplesurance/proteus-consul)](https://goreportcard.com/report/github.com/simplesurance/proteus-consul)

## About

Proteus Consul Key-Value Provider is a _source_ for parameters for
[proteus](https://github.com/simplesurance/proteus). With it, is possible
to read parameter values from consul. This provider supports updating
values without restarting the application.

## Project Status

This project is in pre-release stage and backwards compatibility is not
guaranteed.

## How to Use

Since this provider itself needs to be configured to be able to connect to
consul. For that:

1. register one parameter that will receive the consul URL
2. configure one provider before the _Proteus Consul Key-Value Provider_
3. configure the _Proteus Consul Key-Value Provider_ while providing a
   reference to the parameter that holds the consul URL

```go
params := struct {
	ConsulURL string
	Server    *xtypes.String          `param:",optional"`
	Port      *xtypes.Integer[uint16] `param:",optional"`
	LogLevel  *xtypes.OneOf           `param:",optional"`
}{
	Server: &xtypes.String{
		UpdateFn: func(s string) {
			t.Logf("New Server: %s", s)
		},
	},
	Port: &xtypes.Integer[uint16]{
		UpdateFn: func(v uint16) {
			t.Logf("Port Number: %d", v)
		},
	},
	LogLevel: &xtypes.OneOf{
		Choices:      []string{"off", "debug", "info", "error"},
		DefaultValue: "info",
		UpdateFn: func(s string) {
			t.Logf("New Log Level: %s", s)
		},
	},
}
```

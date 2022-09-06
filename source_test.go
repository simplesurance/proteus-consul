//go:build integration

package cfgconsul_test

import (
	"testing"
	"time"

	"github.com/simplesurance/proteus"
	cfgconsul "github.com/simplesurance/proteus-consul"
	"github.com/simplesurance/proteus/plog"
	"github.com/simplesurance/proteus/sources/cfgtest"
	"github.com/simplesurance/proteus/types"
	"github.com/simplesurance/proteus/xtypes"
)

func TestConsulProvider(t *testing.T) {
	noerr := func(err error) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

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

	testSource := cfgtest.New(types.ParamValues{
		"": map[string]string{
			"consulurl": "http://localhost:8500",
		},
	})

	parsed, err := proteus.MustParse(&params,
		proteus.WithLogger(plog.TestLogger(t)),
		proteus.WithProviders(
			testSource,
			cfgconsul.NewFromReference(cfgconsul.ParameterReferences{
				ConsulURI: cfgconsul.Reference{ParamName: "consulurl"},
			}, "proteus-consul")))
	noerr(err)

	defer parsed.Stop()

	t.Logf("Server: %s", params.Server.Value())
	t.Logf("Port: %d", params.Port.Value())
	t.Logf("Log: %s", params.LogLevel.Value())

	time.Sleep(2 * 60 * time.Second)
}

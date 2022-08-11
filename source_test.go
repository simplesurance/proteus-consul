package cfgconsul_test

import (
	"testing"

	"github.com/simplesurance/proteus"
	cfgconsul "github.com/simplesurance/proteus-consul"
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
		TestValue *xtypes.String
	}{}

	testSource := cfgtest.New(types.ParamValues{
		"": map[string]string{
			"consulurl": "http://localhost:8500",
		},
	})

	_, err := proteus.MustParse(&params,
		proteus.WithLogger(cfgtest.LoggerFor(t)),
		proteus.WithProviders(
			testSource,
			cfgconsul.NewFromReference(cfgconsul.ParameterReferences{
				ConsulURI: cfgconsul.Reference{ParamName: "consulurl"},
			}, "/proteus-consul")))
	noerr(err)

	t.Logf("%s", params.TestValue.Value())
}

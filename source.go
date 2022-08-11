// Package cfgconsul is a configuration provider for proteus that reads
// configuration from consul key-value.
package cfgconsul

import (
	"path"
	"sync"

	consul "github.com/hashicorp/consul/api"
	"github.com/simplesurance/proteus/sources"
	"github.com/simplesurance/proteus/types"
)

// New creates a new Consul KV configuration provider for proteus
// with configuration provided by another provider.
//
// Example:
//
//	params := struct{
//		TheConsulURL string
//		LogLevel     string
//	}{}
//
//	proteus.MustParse(&params, proteus.WithProviders(
//		cfgflags.New(),
//		cfgconsul.New(ParameterReferences{
//			ConsulURI: Reference{"", "theconsulurl"},
//		}),
//	))
func New(chainedParams ParameterReferences, prefix string) sources.Provider {
	ret := &provider{
		prefix: prefix,
	}

	ret.consulURLFn = ret.parametersFromChain(chainedParams)

	return ret
}

// TestProvider is an application configuration provider designed to be used on
// tests.
type provider struct {
	consulURLFn func() (*parameters, error)
	updater     sources.Updater
	prefix      string
	client      *consul.Client

	protected struct {
		mutex  sync.Mutex
		values consulValues
	}
}

type consulValues map[string]map[string]consulValue

type consulValue struct {
	value     string
	lastIndex uint64 // used for blocking GET on consul (watches)
}

var _ sources.Provider = &provider{}

// IsCommandLineFlag returns true, to allow tests to handle special flags that
// only command-line flags are allowed to process.
func (r *provider) IsCommandLineFlag() bool {
	return false
}

// Stop does nothing.
func (r *provider) Stop() {
}

// Watch reads parameters from environment variables. Since environment
// variables never change, we only read once, and we don't have to watch
// for changes.
func (r *provider) Watch(
	paramIDs sources.Parameters,
	updater sources.Updater,
) (initial types.ParamValues, err error) {
	params, err := r.consulURLFn()
	if err != nil {
		return nil, err
	}

	client, err := consul.NewClient(&consul.Config{
		Address: params.consulURI,
	})
	if err != nil {
		return nil, err
	}

	r.client = client

	ret := types.ParamValues{}
	for setName, set := range paramIDs {
		ret[setName] = map[string]string{}
		for paramName := range set {
			val, err := r.get(setName, paramName)
			if err != nil {
				continue
			}

			if val != nil {
				ret[setName][paramName] = *val
			}
		}
	}

	return ret, nil
}

func (r *provider) get(setName, paramName string) (*string, error) {
	kv := r.client.KV()

	// TODO: retries

	var waitIndex uint64
	if set, ok := r.protected.values[setName]; ok {
		if param, ok := set[paramName]; ok {
			waitIndex = param.lastIndex
		}
	}

	consulPath := path.Join(r.prefix, setName, paramName)
	kvPair, meta, err := kv.Get(consulPath, &consul.QueryOptions{
		WaitIndex: waitIndex,
	})
	if err != nil {
		return nil, err
	}

	val := string(kvPair.Value)

	r.set(setName, paramName, val, meta.LastIndex)

	return &val, nil
}

func (r *provider) set(setName, paramName, value string, waitIndex uint64) {
	set, ok := r.protected.values[setName]
	if !ok {
		set = map[string]consulValue{}
		r.protected.values[setName] = set
	}

	set[paramName] = consulValue{
		value:     value,
		lastIndex: waitIndex,
	}
}

func (r *provider) parametersFromChain(
	chainedParams ParameterReferences,
) func() (*parameters, error) {
	return func() (*parameters, error) {
		consulURI, err := r.updater.Peek(chainedParams.ConsulURI.SetName, chainedParams.ConsulURI.ParamName)
		if err != nil {
			return nil, err
		}

		return &parameters{
			consulURI: consulURI,
		}, nil
	}
}

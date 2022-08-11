// Package cfgconsul is a configuration provider for proteus that reads
// configuration from consul key-value.
package cfgconsul

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	consul "github.com/hashicorp/consul/api"
	"github.com/simplesurance/proteus/sources"
	"github.com/simplesurance/proteus/types"
)

// NewFromReference creates a new Consul KV Provider for proteus
// that is itself configured by another provider.
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
//		cfgconsul.NewFromReference(ParameterReferences{
//			ConsulURI: Reference{"", "theconsulurl"},
//		}),
//	))
//
// In this case, providing the consul URL provided using command-line flags
// is used to configure the URL used by the consul provider.
func NewFromReference(chainedParams ParameterReferences, prefix string) sources.Provider {
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	ret := &provider{
		prefix: prefix,
	}

	ret.consulURLFn = ret.parametersFromReference(chainedParams)

	return ret
}

// TestProvider is an application configuration provider designed to be used on
// tests.
type provider struct {
	consulURLFn func() (*parameters, error)
	updater     sources.Updater
	paramNames  sources.Parameters
	prefix      string
	client      *consul.Client
	stopFn      func()
	stopped     sync.WaitGroup

	protected struct {
		mutex  sync.Mutex
		waitIx uint64
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
	r.stopFn()
	r.stopped.Wait()
}

// Watch reads parameters from environment variables. Since environment
// variables never change, we only read once, and we don't have to watch
// for changes.
func (r *provider) Watch(
	paramIDs sources.Parameters,
	updater sources.Updater,
) (initial types.ParamValues, err error) {
	ctx := context.Background()

	r.updater = updater
	r.paramNames = paramIDs

	params, err := r.consulURLFn()
	if err != nil {
		return nil, err
	}

	updater.Log(fmt.Sprintf("Consul URL: %s KV Path: %s",
		params.consulURI, r.prefix))

	client, err := consul.NewClient(&consul.Config{
		Address: params.consulURI,
	})
	if err != nil {
		return nil, err
	}

	r.client = client

	ret, err := r.list(ctx)
	if err != nil {
		return nil, err
	}

	runnerCtx, runnerCancel := context.WithCancel(context.Background())
	r.stopFn = runnerCancel

	r.stopped.Add(1)
	go r.updateWorker(runnerCtx)

	return ret, nil
}

func (r *provider) updateWorker(ctx context.Context) {
	defer r.stopped.Done()

	for ctx.Err() == nil {
		ret, err := r.list(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				r.updater.Log("update worker stopped")
				continue
			}

			r.updater.Log("error getting updates from consul: " + err.Error())
			time.Sleep(time.Second)
			continue
		}

		r.updater.Update(ret)
	}
}

func (r *provider) list(ctx context.Context) (types.ParamValues, error) {
	kv := r.client.KV()

	opts := &consul.QueryOptions{
		WaitIndex: r.protected.waitIx,
		WaitTime:  time.Minute,
	}

	// TODO: retries
	kvPairs, meta, err := kv.List(r.prefix, opts.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	if meta.LastIndex < r.protected.waitIx {
		// according to consul api documentation, the wait index is usually
		// a monotonically increasing number; it might decrease, and in this
		// case we should make the next calls from wait index 0.
		r.protected.waitIx = 0
	}

	r.protected.waitIx = meta.LastIndex

	ret := types.ParamValues{}
	for _, pair := range kvPairs {
		k := strings.TrimPrefix(pair.Key, r.prefix)

		if k == "" {
			continue
		}

		keySplitted := strings.Split(k, "/")
		if len(keySplitted) > 2 {
			r.updater.Log("Ignoring " + pair.Key)
			continue
		}

		setName, paramName := "", keySplitted[0]
		if len(keySplitted) == 2 {
			setName, paramName = keySplitted[0], keySplitted[1]
		}

		if _, found := r.paramNames.Get(setName, paramName); !found {
			r.updater.Log(fmt.Sprintf("Key %q does not match to any application parameter", pair.Key))
			continue
		}

		set, ok := ret[setName]
		if !ok {
			set = map[string]string{}
			ret[setName] = set
		}

		set[paramName] = string(pair.Value)
	}

	j, _ := json.MarshalIndent(ret, "", "  ")
	r.updater.Log(string(j))

	return ret, nil
}

func (r *provider) parametersFromReference(
	chainedParams ParameterReferences,
) func() (*parameters, error) {
	return func() (*parameters, error) {
		consulURI, err := r.updater.Peek(chainedParams.ConsulURI.SetName, chainedParams.ConsulURI.ParamName)
		if err != nil {
			return nil, err
		}

		if consulURI == nil {
			p := chainedParams.ConsulURI.ParamName
			if chainedParams.ConsulURI.SetName != "" {
				p = chainedParams.ConsulURI.SetName + "." + p
			}

			return nil, fmt.Errorf("error initializing Consul KV provider: Consul URL is expected to be provided on parameter %q, but it wasn't",
				p)
		}

		return &parameters{
			consulURI: *consulURI,
		}, nil
	}
}

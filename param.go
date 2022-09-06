package cfgconsul

type parameters struct {
	consulURI string
}

// ParameterReferences specifies from where the Consul KV provider configuration
// (like consul URI) should be read, instead of giving the configuration
// directly.
//
// When this is used, another configuration provider is expected to be
// registered before cfgconsul, and the application is expected to register
// a parameter that contains the configuration.
type ParameterReferences struct {
	ConsulURI Reference
}

// Reference is the parameter set and parameter name where the
// configuration should be read from.
type Reference struct {
	SetName   string
	ParamName string
}

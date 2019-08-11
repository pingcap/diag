package envs

// Env represent the environment variables api passed to collector.
// The map key is the name of enviroment variable name, and the value
// is it's value.
type Env map[string]string

func (e *Env) Get(name string) string {
	return (*e)[name]
}

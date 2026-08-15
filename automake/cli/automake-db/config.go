package main

import "github.com/spf13/viper"

// initViper wires the environment-variable overrides used across the CLI:
// AUTOMAKE_DB (see dbPath) and AUTOMAKE_TOPOLOGY (see resolveTopologyPath).
// An explicit --config flag on the command being run still takes
// precedence -- callers check their own flag value before falling back to
// viper, so this only ever supplies the env-var/default tier.
func initViper() {
	viper.SetEnvPrefix("AUTOMAKE")
	viper.AutomaticEnv()
}

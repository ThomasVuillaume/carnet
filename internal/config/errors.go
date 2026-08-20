package config

// The typed error every rejection of this package carries, named after the key
// at fault. cmd/carnet detects it with errors.As to exit with code 2, so the
// exit code follows from a type rather than from a parsed message.

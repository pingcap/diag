package types

const (
	// UnspecifiedFsp is the unspecified fractional seconds part.
	UnspecifiedFsp = int8(-1)
	// MaxFsp is the maximum digit of fractional seconds part.
	MaxFsp = int8(6)
	// MinFsp is the minimum digit of fractional seconds part.
	MinFsp = int8(0)
	// DefaultFsp is the default digit of fractional seconds part.
	// MySQL use 0 as the default Fsp.
	DefaultFsp = int8(0)
)

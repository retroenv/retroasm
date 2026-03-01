package z80

import z80profile "github.com/retroenv/retroasm/pkg/arch/z80/profile"

// Option configures Z80 architecture behavior.
type Option func(*options)

// WithProfile sets the instruction profile used by the Z80 parser/resolver.
func WithProfile(profileKind z80profile.Kind) Option {
	return func(opts *options) {
		opts.profile = profileKind
	}
}

type options struct {
	profile z80profile.Kind
}

func defaultOptions() options {
	return options{
		profile: z80profile.Default,
	}
}

func resolveOptions(opts []Option) options {
	settings := defaultOptions()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&settings)
	}
	return settings
}

package retroasm

import "maps"

// DefaultConfiguration provides a basic configuration implementation.
// It implements the Configuration interface with standard memory layout,
// segment definitions, and symbol table management.
type DefaultConfiguration struct {
	memoryLayout MemoryLayout
	segments     []SegmentConfig
	symbols      map[string]uint64
}

// ConfigurationBuilder helps build configurations fluently.
// Provides a builder pattern for creating configurations with
// method chaining for improved readability.
type ConfigurationBuilder struct {
	config *DefaultConfiguration
}

// NewDefaultConfiguration creates a new default configuration.
// Returns a configuration with 16-bit little-endian addressing,
// empty segments list, and initialized symbol table.
func NewDefaultConfiguration() Configuration {
	return &DefaultConfiguration{
		memoryLayout: MemoryLayout{
			AddressSize: 16,
			Endianness:  LittleEndian,
		},
		segments: make([]SegmentConfig, 0, 8), // Pre-allocate for typical segment count
		symbols:  make(map[string]uint64, 16), // Pre-allocate for typical symbol count
	}
}

// NewConfigurationBuilder creates a new configuration builder.
// Returns a builder initialized with default 16-bit little-endian settings.
func NewConfigurationBuilder() *ConfigurationBuilder {
	return &ConfigurationBuilder{
		config: &DefaultConfiguration{
			memoryLayout: MemoryLayout{
				AddressSize: 16,
				Endianness:  LittleEndian,
			},
			segments: make([]SegmentConfig, 0, 8), // Pre-allocate for typical segment count
			symbols:  make(map[string]uint64, 16), // Pre-allocate for typical symbol count
		},
	}
}

// MemoryLayout returns the memory layout configuration.
func (c *DefaultConfiguration) MemoryLayout() MemoryLayout {
	return c.memoryLayout
}

// Segments returns the segment configurations.
func (c *DefaultConfiguration) Segments() []SegmentConfig {
	return c.segments
}

// Symbols returns the symbol table.
func (c *DefaultConfiguration) Symbols() map[string]uint64 {
	return c.symbols
}

// SetMemoryLayout sets the memory layout.
func (c *DefaultConfiguration) SetMemoryLayout(layout MemoryLayout) {
	c.memoryLayout = layout
}

// AddSegment adds a segment configuration.
func (c *DefaultConfiguration) AddSegment(segment SegmentConfig) {
	c.segments = append(c.segments, segment)
}

// SetSymbol sets a symbol value.
func (c *DefaultConfiguration) SetSymbol(name string, value uint64) {
	c.symbols[name] = value
}

// SetMemoryLayout sets the memory layout and returns the builder.
func (b *ConfigurationBuilder) SetMemoryLayout(layout MemoryLayout) *ConfigurationBuilder {
	b.config.memoryLayout = layout
	return b
}

// AddSegment adds a segment configuration and returns the builder.
func (b *ConfigurationBuilder) AddSegment(segment SegmentConfig) *ConfigurationBuilder {
	b.config.segments = append(b.config.segments, segment)
	return b
}

// SetSymbol sets a symbol value and returns the builder.
func (b *ConfigurationBuilder) SetSymbol(name string, value uint64) *ConfigurationBuilder {
	b.config.symbols[name] = value
	return b
}

// Build creates the final configuration as a deep copy to prevent external modification.
func (b *ConfigurationBuilder) Build() Configuration {
	symbols := make(map[string]uint64, len(b.config.symbols))
	maps.Copy(symbols, b.config.symbols)

	segments := make([]SegmentConfig, len(b.config.segments))
	copy(segments, b.config.segments)

	return &DefaultConfiguration{
		memoryLayout: b.config.memoryLayout,
		segments:     segments,
		symbols:      symbols,
	}
}

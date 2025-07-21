package config

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/retroenv/retroasm/lexer"
	"github.com/retroenv/retroasm/lexer/token"
	"github.com/retroenv/retroasm/number"
)

// TODO support FEATURES and SYMBOLS areas
// https://cc65.github.io/doc/ld65.html#toc5

const (
	yes = "yes"
	no  = "no"
)

// ReadCa65Config reads a ca65 configuration.
func (c *Config[T]) ReadCa65Config(reader io.Reader) error {
	lexerCfg := lexer.Config{
		CommentPrefixes: []string{"#"},
		DecimalPrefix:   0,
	}

	lex := lexer.New(lexerCfg, reader)
	return c.readCa65Config(lex)
}

type ca65Area struct {
	name       string
	attributes map[string]string
}

// readCa65Config reads a ca65 config from the given lexer.
func (c *Config[T]) readCa65Config(lex *lexer.Lexer) error {
	var memory, segments []*ca65Area

	for eof := false; !eof; {
		tok, err := lex.NextToken()
		if err != nil {
			return fmt.Errorf("reading next token: %w", err)
		}

		switch tok.Type {
		case token.EOF:
			eof = true
			continue

		case token.Illegal:
			return fmt.Errorf("illegal token '%s' found at line %d column %d",
				tok.Value, tok.Position.Line, tok.Position.Column)

		case token.EOL, token.Comment:
			continue

		case token.Identifier:
			identifier := strings.ToLower(tok.Value)
			switch identifier {
			case "memory":
				memory, err = readCa65ConfigSection(lex)
				if err != nil {
					return fmt.Errorf("reading memory section: %w", err)
				}

			case "segments":
				segments, err = readCa65ConfigSection(lex)
				if err != nil {
					return fmt.Errorf("reading segments section: %w", err)
				}
			default:
				return fmt.Errorf("unsupported identifier '%s' found at line %d column %d",
					tok.Value, tok.Position.Line, tok.Position.Column)
			}

		default:
			return fmt.Errorf("unsupported identifier '%s' found at line %d column %d",
				tok.Value, tok.Position.Line, tok.Position.Column)
		}
	}

	if err := c.readFromCa65Areas(memory, segments); err != nil {
		return fmt.Errorf("reading areas: %w", err)
	}
	return nil
}

// readFromCa65Areas reads the configuration from the ca65 config areas.
func (c *Config[T]) readFromCa65Areas(memories, segments []*ca65Area) error {
	memoryNames := map[string]*Memory{}
	for _, m := range memories {
		memory, err := convertCa65MemoryArea(m)
		if err != nil {
			return fmt.Errorf("processing memory area '%s': %w", m.name, err)
		}

		memoryNames[m.name] = memory
	}

	startup := &ca65Area{
		name: "STARTUP",
	}
	segments = append(segments, startup)

	c.Segments = map[string]*Segment{}
	for _, seg := range segments {
		segment, err := convertCa65SegmentArea(seg, memoryNames)
		if err != nil {
			return fmt.Errorf("processing segment area '%s': %w", seg.name, err)
		}
		segment.SegmentName = seg.name

		c.Segments[seg.name] = segment
		c.SegmentsOrdered = append(c.SegmentsOrdered, segment)
	}

	return nil
}

var errNoStartingBlock = errors.New("no { starting block found")

// readCa65ConfigSection reads a config section, this can either be MEMORY or SEGMENTS.
func readCa65ConfigSection(lex *lexer.Lexer) ([]*ca65Area, error) {
	leftBraceFound := false              // flag to detect multiple left braces
	identifiers := map[string]struct{}{} // area identifiers set to detect duplicates
	var areas []*ca65Area                // all read areas to return
	var ar *ca65Area                     // current area to be read

	for {
		tok, err := lex.NextToken()
		if err != nil {
			return nil, fmt.Errorf("reading next token: %w", err)
		}

		switch tok.Type {
		case token.EOL, token.Comment,
			token.Colon, // after area name
			token.Comma: // setting separator

		case token.LeftBrace:
			if leftBraceFound {
				return nil, errors.New("multiple { starting blocks found")
			}
			leftBraceFound = true

		case token.RightBrace:
			if !leftBraceFound {
				return nil, errNoStartingBlock
			}
			return areas, nil

		case token.Semicolon:
			ar = nil

		case token.Identifier:
			if !leftBraceFound {
				return nil, errNoStartingBlock
			}

			if ar != nil { // inside an area?
				if err := readCa65ConfigAttribute(lex, ar, tok.Value); err != nil {
					return nil, fmt.Errorf("reading attribute '%s': %w", tok.Value, err)
				}
				continue
			}

			// identifiers are case-sensitive
			if _, ok := identifiers[tok.Value]; ok {
				return nil, fmt.Errorf("multiple areas named '%s' found", tok.Value)
			}
			identifiers[tok.Value] = struct{}{}

			ar = &ca65Area{
				name:       tok.Value,
				attributes: map[string]string{},
			}
			areas = append(areas, ar)

		default:
			return nil, fmt.Errorf("unexpected token type found: '%s'", tok.Type.String())
		}
	}
}

// readCa65ConfigAttribute reads an attribute for an area, it supports the 2 styles:
// start = $8000, size = $4000
// and:
// start $0800
// size = $4000.
func readCa65ConfigAttribute(lex *lexer.Lexer, ar *ca65Area, attribute string) error {
	tok, err := lex.NextToken()
	if err != nil {
		return fmt.Errorf("reading next token: %w", err)
	}

	if tok.Type == token.Assign { // assign is optional
		tok, err = lex.NextToken()
		if err != nil {
			return fmt.Errorf("reading next token: %w", err)
		}
	}

	if tok.Value == "%" { // support %O references
		tok, err = lex.NextToken()
		if err != nil {
			return fmt.Errorf("reading next token: %w", err)
		}
		tok.Value = "%" + tok.Value
	}

	if tok.Type != token.Identifier && tok.Type != token.Number {
		return fmt.Errorf("unexpected token type found: '%s'", tok.Type.String())
	}

	ar.attributes[strings.ToLower(attribute)] = tok.Value
	return nil
}

// convertCa65MemoryArea converts a MEMORY area of the ca65 configuration to a memory type instance.
func convertCa65MemoryArea(ar *ca65Area) (*Memory, error) {
	mem := &Memory{
		Name: ar.name,
	}
	if err := parseCa65MemoryArea(ar, mem, false); err != nil {
		return nil, fmt.Errorf("parsing memory area: %w", err)
	}
	return mem, nil
}

// nolint:cyclop
func parseCa65MemoryArea(ar *ca65Area, mem *Memory, ignoreUnknownKeys bool) error {
	var err error

	for key, value := range ar.attributes {
		switch key {
		case "start":
			mem.Start, err = number.Parse(value)
			if err != nil {
				return fmt.Errorf("parsing number '%s': %w", value, err)
			}

		case "size":
			mem.Size, err = number.Parse(value)
			if err != nil {
				return fmt.Errorf("parsing number '%s': %w", value, err)
			}

		case "file":
			mem.File = strings.Trim(value, "\"'") // unescape string

		case "type":
			mem.Typ = value

		case "fillval":
			i, err := number.Parse(value)
			if err != nil {
				return fmt.Errorf("parsing number '%s': %w", value, err)
			}
			if i > math.MaxUint8 {
				return fmt.Errorf("fill value '%s' exceeds byte", value)
			}
			mem.FillValue = byte(i)

		case "fill":
			switch value {
			case yes:
				mem.Fill = true
			case no:
			default:
				return fmt.Errorf("unsupported fill value '%s'", value)
			}

		default:
			if !ignoreUnknownKeys {
				return fmt.Errorf("unsupported area key '%s'", key)
			}
		}
	}

	return nil
}

// convertCa65SegmentArea converts a SEGMENTS area of the ca65 configuration to a segment type instance.
// nolint: cyclop, funlen
func convertCa65SegmentArea(ar *ca65Area, memoryNames map[string]*Memory) (*Segment, error) {
	seg := &Segment{}
	var err error

	// load specified memory area and set as default values
	memoryName, ok := ar.attributes["load"]
	if ok {
		mem, ok := memoryNames[memoryName]
		if !ok {
			return nil, fmt.Errorf("memory area '%s' not found", memoryName)
		}
		seg.Memory = *mem
	}

	memoryStart := seg.Memory.Start

	// overload all specified memory related keys
	if err := parseCa65MemoryArea(ar, &seg.Memory, true); err != nil {
		return nil, fmt.Errorf("parsing memory area keys: %w", err)
	}

	seg.SegmentStart = seg.Start
	// restore memory start in case the memory loader overwrote it
	if memoryStart != seg.Start {
		seg.Start = memoryStart
	}

	// parse all segment specific keys
	for key, value := range ar.attributes {
		switch key {
		case "align":
			seg.Align, err = number.Parse(value)
			if err != nil {
				return nil, fmt.Errorf("parsing number '%s': %w", value, err)
			}

		case "offset":
			seg.Offset, err = number.Parse(value)
			return nil, fmt.Errorf("parsing number '%s': %w", value, err)

		case "define":
			switch value {
			case yes:
				seg.Define = true
			case no:
			default:
				return nil, fmt.Errorf("unsupported define value '%s'", value)
			}

		case "optional":
			switch value {
			case yes:
				seg.Optional = true
			case no:
			default:
				return nil, fmt.Errorf("unsupported optional value '%s'", value)
			}

		case "run":
			seg.Run = value
		}
	}

	return seg, nil
}

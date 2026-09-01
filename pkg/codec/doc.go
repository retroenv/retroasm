// Package codec provides an architecture-bound typed assembly stream boundary.
//
// A Codec parses source into RetroASM AST nodes, resolves one typed instruction,
// looks up architecture-scoped opcode identities, and assembles existing nodes
// without formatting and reparsing them.
package codec

package chroma

import (
	"bytes"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/muesli/reflow/wrap"
)

// NewCadenceLexer creates a Chroma lexer for the Cadence language.
// Patterns are based on the official TextMate grammar (cadence.tmGrammar.json)
// from the vscode extensoins repo for onflow.
func NewCadenceLexer() chroma.Lexer {
	return chroma.MustNewLexer(
		&chroma.Config{
			Name:      "Cadence",
			Aliases:   []string{"cadence", "cdc"},
			Filenames: []string{"*.cdc"},
			MimeTypes: []string{"text/x-cadence"},
		},
		func() chroma.Rules {
			return chroma.Rules{
				"root": {
					// Keywords - based on grammar "keywords" section
					{Pattern: `\b(import|from|transaction|prepare|execute|pre|post)\b`, Type: chroma.KeywordNamespace, Mutator: nil},
					{Pattern: `\b(contract|struct|resource|interface|event|enum|attachment|entitlement)\b`, Type: chroma.KeywordDeclaration, Mutator: nil},
					{Pattern: `\b(fun|let|var|init)\b`, Type: chroma.KeywordDeclaration, Mutator: nil},
					{Pattern: `\b(if|else|switch|case|default|while|for|in|break|continue|return)\b`, Type: chroma.Keyword, Mutator: nil},
					{Pattern: `\b(pub|priv|access|auth|view|all|self|mapping|include)\b`, Type: chroma.KeywordReserved, Mutator: nil},
					{Pattern: `\b(create|destroy|emit|attach|to|remove|from|as)\b`, Type: chroma.Keyword, Mutator: nil},

					// Types - Cadence built-in types
					{Pattern: `\b(Int|UInt|Int8|Int16|Int32|Int64|Int128|Int256|UInt8|UInt16|UInt32|UInt64|UInt128|UInt256)\b`, Type: chroma.KeywordType, Mutator: nil},
					{Pattern: `\b(Word8|Word16|Word32|Word64|Fix64|UFix64|String|Character|Bool|Address|Void)\b`, Type: chroma.KeywordType, Mutator: nil},
					{Pattern: `\b(AnyStruct|AnyResource|Any|Never|Type|Capability|Account|StoragePath|PublicPath|PrivatePath|CapabilityPath)\b`, Type: chroma.KeywordType, Mutator: nil},

					// Standard library types (common contracts/interfaces)
					{Pattern: `\b(FungibleToken|NonFungibleToken|MetadataViews|ViewResolver|Burner|Vault|Receiver|Provider|Collection)\b`, Type: chroma.NameClass, Mutator: nil},

					// Storage and capability keywords
					{Pattern: `\b(Storage|Capabilities|storage|capabilities|borrow|withdraw|deposit|balance|save|load|copy|check)\b`, Type: chroma.NameBuiltin, Mutator: nil},

					// Literals - based on grammar "literals" section
					{Pattern: `\b(true|false|nil)\b`, Type: chroma.KeywordConstant, Mutator: nil},

					// Numbers - matching grammar numeric patterns with underscores support
					{Pattern: `\b[0-9]([_0-9]*[0-9])?\.[0-9]([_0-9]*[0-9])?\b`, Type: chroma.LiteralNumberFloat, Mutator: nil},
					{Pattern: `\b0x[0-9A-Fa-f]([_0-9A-Fa-f]*[0-9A-Fa-f])?\b`, Type: chroma.LiteralNumberHex, Mutator: nil},
					{Pattern: `\b0b[01]([_01]*[01])?\b`, Type: chroma.LiteralNumberBin, Mutator: nil},
					{Pattern: `\b0o[0-7]([_0-7]*[0-7])?\b`, Type: chroma.LiteralNumberOct, Mutator: nil},
					{Pattern: `\b[0-9]([_0-9]*[0-9])?\b`, Type: chroma.LiteralNumberInteger, Mutator: nil},

					// Strings - based on grammar string patterns
					{Pattern: `"(?:[^"\\]|\\.)*"`, Type: chroma.LiteralString, Mutator: nil},

					// Comments - matching grammar comment patterns
					{Pattern: `///.*?$`, Type: chroma.CommentSpecial, Mutator: nil}, // Documentation comment
					{Pattern: `//:.*?$`, Type: chroma.CommentSpecial, Mutator: nil}, // Documentation comment
					{Pattern: `//.*?$`, Type: chroma.CommentSingle, Mutator: nil},
					{Pattern: `/\*`, Type: chroma.CommentMultiline, Mutator: chroma.Push("comment")},

					// Operators - based on grammar operators section
					{Pattern: `<-!`, Type: chroma.Operator, Mutator: nil}, // Force-move
					{Pattern: `<->`, Type: chroma.Operator, Mutator: nil}, // Swap
					{Pattern: `<-`, Type: chroma.Operator, Mutator: nil},  // Move
					{Pattern: `[+\-*/%]`, Type: chroma.Operator, Mutator: nil},
					{Pattern: `[=!<>]=?`, Type: chroma.Operator, Mutator: nil},
					{Pattern: `&&|\|\|`, Type: chroma.Operator, Mutator: nil},
					{Pattern: `\?\?`, Type: chroma.Operator, Mutator: nil}, // Nil coalescing
					{Pattern: `\?\.`, Type: chroma.Operator, Mutator: nil}, // Optional chaining
					{Pattern: `[?!]`, Type: chroma.Operator, Mutator: nil},
					{Pattern: `&|@`, Type: chroma.Operator, Mutator: nil},

					// Path literals - /storage/path, /public/path, /private/path
					{Pattern: `/(storage|public|private)(/[a-zA-Z_][a-zA-Z0-9_]*)?`, Type: chroma.LiteralString, Mutator: nil},

					// Punctuation
					{Pattern: `[(){}\[\],.:;]`, Type: chroma.Punctuation, Mutator: nil},

					// Function calls and names
					{Pattern: `\b([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`, Type: chroma.NameFunction, Mutator: nil},

					// Type annotations (capitalized identifiers)
					{Pattern: `\b[A-Z][a-zA-Z0-9_]*\b`, Type: chroma.NameClass, Mutator: nil},

					// Regular identifiers
					{Pattern: `\b[a-zA-Z_][a-zA-Z0-9_]*\b`, Type: chroma.Name, Mutator: nil},

					// Whitespace
					{Pattern: `\s+`, Type: chroma.Text, Mutator: nil},
				},
				"comment": {
					// Nested comment support
					{Pattern: `[^*/]+`, Type: chroma.CommentMultiline, Mutator: nil},
					{Pattern: `/\*`, Type: chroma.CommentMultiline, Mutator: chroma.Push("comment")},
					{Pattern: `\*/`, Type: chroma.CommentMultiline, Mutator: chroma.Pop(1)},
					{Pattern: `[*/]`, Type: chroma.CommentMultiline, Mutator: nil},
				},
			}
		},
	)
}

// HighlightCadence takes a Cadence code fragment and returns it with ANSI syntax highlighting.
// The highlighted string can be stored and displayed in terminal UIs.
// Returns the original code if highlighting fails.
func HighlightCadence(code string) string {
	return HighlightCadenceWithStyle(code, "solarized-dark")
}

// HighlightCadenceWithStyle highlights Cadence code with a specific style.
// Available styles: "monokai", "solarized-dark", "solarized-light", "github", "vim", etc.
// Returns the original code if highlighting fails.
func HighlightCadenceWithStyle(code, styleName string) string {
	return HighlightCadenceWithStyleAndWidth(code, styleName, 0)
}

// HighlightCadenceWithStyleAndWidth highlights Cadence code with optional ANSI-aware width wrapping AFTER highlighting.
// Available styles: "monokai", "solarized-dark", "solarized-light", "github", "vim", etc.
// maxWidth: maximum line width in visible characters (0 = no wrapping). Wrapping happens AFTER highlighting to preserve ANSI codes.
// Returns the original code if highlighting fails.
func HighlightCadenceWithStyleAndWidth(code, styleName string, maxWidth int) string {
	// Get the lexer
	lexer := NewCadenceLexer()

	// Get the style
	style := styles.Get(styleName)
	if style == nil {
		style = styles.Fallback
	}

	// Use terminal256 formatter for ANSI output
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// Tokenize the code
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		// Return original code if tokenization fails
		return code
	}

	// Format to a buffer
	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		// Return original code if formatting fails
		return code
	}

	highlighted := buf.String()

	// Wrap AFTER highlighting using ANSI-aware wordwrap if maxWidth is specified
	if maxWidth > 0 {
		highlighted = wrap.String(highlighted, maxWidth)
	}

	return highlighted
}

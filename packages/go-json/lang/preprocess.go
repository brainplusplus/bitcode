package lang

// StripComments removes JSONC extensions from input, producing strict JSON:
//   - // line comments (to end of line)
//   - /* */ block comments
//   - trailing commas before ] and }
//
// Comments inside JSON strings are preserved.
func StripComments(input []byte) []byte {
	if len(input) == 0 {
		return input
	}

	out := make([]byte, 0, len(input))
	i := 0
	n := len(input)

	for i < n {
		// Inside a JSON string — copy verbatim until closing quote.
		if input[i] == '"' {
			out = append(out, input[i])
			i++
			for i < n {
				if input[i] == '\\' && i+1 < n {
					// Escaped character — copy both bytes and skip.
					out = append(out, input[i], input[i+1])
					i += 2
					continue
				}
				if input[i] == '"' {
					out = append(out, input[i])
					i++
					break
				}
				out = append(out, input[i])
				i++
			}
			continue
		}

		// Line comment: // ... \n
		if i+1 < n && input[i] == '/' && input[i+1] == '/' {
			i += 2
			for i < n && input[i] != '\n' {
				i++
			}
			continue
		}

		// Block comment: /* ... */
		if i+1 < n && input[i] == '/' && input[i+1] == '*' {
			i += 2
			found := false
			for i+1 < n {
				if input[i] == '*' && input[i+1] == '/' {
					i += 2
					found = true
					break
				}
				i++
			}
			if !found {
				// Unterminated block comment — consume everything remaining.
				break
			}
			continue
		}

		out = append(out, input[i])
		i++
	}

	// Second pass: strip trailing commas before ] and }.
	out = stripTrailingCommas(out)

	return out
}

// stripTrailingCommas removes commas that appear (with optional whitespace)
// immediately before ] or }.
func stripTrailingCommas(input []byte) []byte {
	out := make([]byte, 0, len(input))
	i := 0
	n := len(input)

	for i < n {
		// Inside a string — copy verbatim.
		if input[i] == '"' {
			out = append(out, input[i])
			i++
			for i < n {
				if input[i] == '\\' && i+1 < n {
					out = append(out, input[i], input[i+1])
					i += 2
					continue
				}
				if input[i] == '"' {
					out = append(out, input[i])
					i++
					break
				}
				out = append(out, input[i])
				i++
			}
			continue
		}

		if input[i] == ',' {
			// Look ahead past whitespace for ] or }.
			j := i + 1
			for j < n && isWhitespace(input[j]) {
				j++
			}
			// If next non-whitespace is another comma, skip this comma
			// (handles multiple trailing commas like [1,,,]).
			if j < n && input[j] == ',' {
				// Skip this comma, let the next iteration handle the next one.
				i++
				continue
			}
			if j < n && (input[j] == ']' || input[j] == '}') {
				// Trailing comma — skip it, keep the whitespace and bracket.
				i++
				continue
			}
		}

		out = append(out, input[i])
		i++
	}

	return out
}

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

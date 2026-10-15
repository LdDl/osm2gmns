package csvexport

import (
	"fmt"
	"unicode/utf8"
)

const DefaultDelimiter rune = ';'

type options struct {
	delimiter rune
}

type Option func(*options)

// WithDelimiter selects the separator for every CSV table in an export.
func WithDelimiter(delimiter rune) Option {
	return func(settings *options) {
		settings.delimiter = delimiter
	}
}

// ParseDelimiter accepts one valid CSV separator character.
func ParseDelimiter(value string) (rune, error) {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) != 1 {
		return 0, fmt.Errorf("CSV delimiter must be one UTF-8 character")
	}
	delimiter, _ := utf8.DecodeRuneInString(value)
	err := validateDelimiter(delimiter)
	if err != nil {
		return 0, err
	}
	return delimiter, nil
}

func resolveOptions(configure []Option) (*options, error) {
	settings := &options{
		delimiter: DefaultDelimiter,
	}
	for _, option := range configure {
		if option == nil {
			return nil, fmt.Errorf("nil CSV export option")
		}
		option(settings)
	}
	err := validateDelimiter(settings.delimiter)
	if err != nil {
		return nil, err
	}
	return settings, nil
}

func validateDelimiter(delimiter rune) error {
	if !utf8.ValidRune(delimiter) || delimiter == 0 || delimiter == '"' || delimiter == '\r' || delimiter == '\n' || delimiter == utf8.RuneError {
		return fmt.Errorf("invalid CSV delimiter %q", delimiter)
	}
	return nil
}

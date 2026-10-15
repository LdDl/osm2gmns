package csvexport

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

type delimiterCase struct {
	name      string
	delimiter rune
	useOption bool
}

type invalidDelimiterCase struct {
	name  string
	value string
}

func TestDelimiterRoundTripAndDescriptor(t *testing.T) {
	cases := []delimiterCase{
		{
			name:      "default semicolon",
			delimiter: ';',
		},
		{
			name:      "comma",
			delimiter: ',',
			useOption: true,
		},
		{
			name:      "tab",
			delimiter: '\t',
			useOption: true,
		},
		{
			name:      "unicode",
			delimiter: '§',
			useOption: true,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			dataset := smallDataset()
			dataset.Tables[1].Header = append(dataset.Tables[1].Header, "name")
			dataset.Tables[1].Rows[0] = append(dataset.Tables[1].Rows[0], "Road; comma, quote\" tab\t section§\nnext line")
			options := make([]Option, 0)
			if testCase.useOption {
				options = append(options, WithDelimiter(testCase.delimiter))
			}
			directory := t.TempDir()
			err := WriteDatasets(directory, []Dataset{dataset}, options...)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			data, err := os.ReadFile(filepath.Join(directory, "datapackage_macro.json"))
			assert.NoError(t, err)
			descriptor := make(map[string]any)
			assert.NoError(t, json.Unmarshal(data, &descriptor))
			resources, ok := descriptor["resources"].([]any)
			assert.True(t, ok)
			assert.Len(t, resources, len(dataset.Tables))
			for index, value := range resources {
				resource := value.(map[string]any)
				dialect := resource["dialect"].(map[string]any)
				separator := dialect["delimiter"].(string)
				assert.Equal(t, string(testCase.delimiter), separator)
				decoded, parseErr := ParseDelimiter(separator)
				assert.NoError(t, parseErr)
				data, readErr := os.ReadFile(filepath.Join(directory, resource["path"].(string)))
				assert.NoError(t, readErr)
				reader := csv.NewReader(bytes.NewReader(data))
				reader.Comma = decoded
				rows, readErr := reader.ReadAll()
				assert.NoError(t, readErr)
				expected := [][]string{dataset.Tables[index].Header}
				expected = append(expected, dataset.Tables[index].Rows...)
				assert.Equal(t, expected, rows)
			}
		})
	}
}

func TestInvalidDelimiterDoesNotTouchOutput(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "node_macro.csv")
	original := []byte("existing table")
	assert.NoError(t, os.WriteFile(filename, original, 0600))
	invalid := []rune{0, '"', '\r', '\n', utf8.RuneError, -1, 0xD800, utf8.MaxRune + 1}
	for _, delimiter := range invalid {
		option := WithDelimiter(delimiter)
		dataset := smallDataset()
		assert.Error(t, WriteTable(filename, dataset.Tables[1], option))
		assert.Error(t, WriteDatasets(directory, []Dataset{dataset}, option))
		assert.Error(t, WriteDatasets(filepath.Join(directory, "new"), []Dataset{dataset}, option))
	}
	actual, err := os.ReadFile(filename)
	assert.NoError(t, err)
	assert.Equal(t, original, actual)
	entries, err := os.ReadDir(directory)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestParseDelimiterRejectsInvalidInput(t *testing.T) {
	cases := []invalidDelimiterCase{
		{
			name:  "empty",
			value: "",
		},
		{
			name:  "multiple characters",
			value: ";;",
		},
		{
			name:  "invalid UTF-8",
			value: string([]byte{0xff}),
		},
		{
			name:  "quote",
			value: "\"",
		},
		{
			name:  "newline",
			value: "\n",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := ParseDelimiter(testCase.value)
			assert.Error(t, err)
		})
	}
}

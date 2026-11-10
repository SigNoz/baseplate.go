package kyaml

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"

	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	"github.com/SigNoz/baseplate.go/errors"
)

// ToJSON returns one JSON document for each document in y, in order.
// Documents without content, such as a leading "---" or a document of only
// comments, are skipped. Comments are dropped.
//
// It returns a [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput]
// error with [CodeYAMLInvalid] when y isn't valid YAML, and one with
// [CodeYAMLEmpty] when y has no document with content.
func ToJSON(y YAML) ([]json.RawMessage, error) {
	reader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(y)))

	var docs []json.RawMessage

	for {
		doc, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, errors.Wrap(err, errors.TypeInvalidInput, CodeYAMLInvalid, "yaml is invalid")
		}

		converted, err := utilyaml.ToJSON(doc)
		if err != nil {
			return nil, errors.Wrap(err, errors.TypeInvalidInput, CodeYAMLInvalid, "yaml is invalid")
		}

		// A document without content converts to null.
		if string(converted) == "null" {
			continue
		}

		docs = append(docs, converted)
	}

	if len(docs) == 0 {
		return nil, errors.New(errors.TypeInvalidInput, CodeYAMLEmpty, "yaml has no documents")
	}

	return docs, nil
}

// FromJSON converts each JSON document in docs to YAML, formats it with
// [Format] and joins the results with "---". Strings that YAML 1.1 would
// read as another type, such as "on" or "0755", are quoted, and long lines
// are not wrapped. It returns an empty stream when docs is empty.
//
// It returns a [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput]
// error with [CodeJSONInvalid] for the first document that isn't valid
// JSON.
func FromJSON(docs ...json.RawMessage) (YAML, error) {
	formatted := make([][]byte, 0, len(docs))

	for i, doc := range docs {
		converted, err := yaml.JSONToYAML(doc)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, CodeJSONInvalid, "json document %d is invalid", i)
		}

		out, err := Format(converted)
		if err != nil {
			return nil, err
		}

		formatted = append(formatted, out)
	}

	return bytes.Join(formatted, []byte("---\n")), nil
}

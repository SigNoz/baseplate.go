package kyaml

import "github.com/SigNoz/baseplate.go/errors"

// The codes of the errors [Format], [ToJSON] and [FromJSON] return.
var (
	CodeYAMLInvalid = errors.MustNewCode("yaml_invalid")
	CodeYAMLEmpty   = errors.MustNewCode("yaml_empty")
	CodeJSONInvalid = errors.MustNewCode("json_invalid")
)

// YAML is a YAML stream of one or more documents separated by "---".
type YAML []byte

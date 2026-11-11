package kyaml

import "github.com/SigNoz/baseplate.go/errors"

// The codes of the errors [Format], [ToJSON], [FromJSON], [Set] and
// [EditBlockScalar] return.
var (
	CodeYAMLInvalid       = errors.MustNewCode("yaml_invalid")
	CodeYAMLEmpty         = errors.MustNewCode("yaml_empty")
	CodeJSONInvalid       = errors.MustNewCode("json_invalid")
	CodeMultipleDocuments = errors.MustNewCode("multiple_documents")
	CodePathNotFound      = errors.MustNewCode("path_not_found")
	CodeNotScalar         = errors.MustNewCode("not_scalar")
	CodeUnsafeValue       = errors.MustNewCode("unsafe_value")
	CodeNotBlockScalar    = errors.MustNewCode("not_block_scalar")
)

// YAML is a YAML stream of one or more documents separated by "---".
type YAML []byte

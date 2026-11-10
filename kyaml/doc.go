// Package kyaml formats Kubernetes YAML and converts it to and from JSON,
// so that the manifests a program writes look the same as manifests
// formatted by hand.
//
// # Formatting
//
// [Format] rewrites each document in a YAML stream with kustomize's
// formatter. A document with an apiVersion and a kind gets its fields in
// canonical Kubernetes order: apiVersion, kind, metadata, spec and data
// come first, name leads the fields of a list item, and the keys of maps
// such as labels and data are sorted. A Kustomization keeps apiVersion and
// kind above namespace. A document without an apiVersion or a kind, such
// as a Helm values file, keeps its field order.
//
// Every document is indented by two spaces, with sequences at the same
// indent as their key. Comments and quoting are kept as written, and
// documents with no content are dropped. A List stays one document, with
// its items formatted inside it. Formatting formatted YAML changes nothing.
//
// # Editing through JSON
//
// To change a manifest, a program converts it with [ToJSON], edits the
// JSON documents, for example with a JSON Patch, and converts them back
// with [FromJSON]:
//
//	docs, err := kyaml.ToJSON(kyaml.YAML(file))
//	if err != nil {
//		return err
//	}
//
//	// Change docs[0].
//
//	out, err := kyaml.FromJSON(docs...)
//
// [FromJSON] formats every document with [Format], so its output is what
// [Format] would write. The round trip drops comments and sorts the fields
// that have no canonical position. It quotes strings that YAML 1.1
// parsers, Kubernetes among them, would read as another type, such as "on"
// and "0755", and it doesn't wrap long lines.
//
// # Types
//
// [YAML] holds a YAML stream and [encoding/json.RawMessage] holds a JSON
// document, so a signature says which of the two it expects. Both are byte
// slices, and a []byte, such as the result of [os.ReadFile], converts to
// either.
//
// # Errors
//
// Every error is a [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput]
// error, with [CodeYAMLInvalid] for YAML that doesn't parse,
// [CodeYAMLEmpty] for YAML with no document to convert, and
// [CodeJSONInvalid] for JSON that doesn't parse.
package kyaml

package kyaml_test

import (
	"encoding/json"
	"fmt"

	"github.com/SigNoz/baseplate.go/kyaml"
)

func ExampleFormat() {
	input := kyaml.YAML(`metadata:
  name: myapp # The service name.
kind: Service
apiVersion: v1
spec:
  ports:
    - port: 80
      name: http
`)

	formatted, err := kyaml.Format(input)
	if err != nil {
		_, _ = fmt.Println(err)

		return
	}

	_, _ = fmt.Print(string(formatted))
	// Output:
	// apiVersion: v1
	// kind: Service
	// metadata:
	//   name: myapp # The service name.
	// spec:
	//   ports:
	//   - name: http
	//     port: 80
}

// The leading empty document is skipped and the comment is dropped.
func ExampleToJSON() {
	input := kyaml.YAML(`---
apiVersion: v1
kind: ConfigMap
metadata:
  name: myapp # The app's settings.
data:
  mode: "0755"
`)

	docs, err := kyaml.ToJSON(input)
	if err != nil {
		_, _ = fmt.Println(err)

		return
	}

	for _, doc := range docs {
		_, _ = fmt.Println(string(doc))
	}
	// Output:
	// {"apiVersion":"v1","data":{"mode":"0755"},"kind":"ConfigMap","metadata":{"name":"myapp"}}
}

// The string "on" is quoted, so YAML 1.1 parsers don't read it as a bool.
func ExampleFromJSON() {
	out, err := kyaml.FromJSON(
		json.RawMessage(`{"metadata":{"name":"myapp"},"kind":"ConfigMap","apiVersion":"v1","data":{"enabled":"on"}}`),
		json.RawMessage(`{"kind":"ServiceAccount","apiVersion":"v1","metadata":{"name":"myapp"}}`),
	)
	if err != nil {
		_, _ = fmt.Println(err)

		return
	}

	_, _ = fmt.Print(string(out))
	// Output:
	// apiVersion: v1
	// kind: ConfigMap
	// metadata:
	//   name: myapp
	// data:
	//   enabled: "on"
	// ---
	// apiVersion: v1
	// kind: ServiceAccount
	// metadata:
	//   name: myapp
}

// A program edits a manifest by changing the JSON between the two calls.
// Here the JSON is left as it is, and the YAML comes back formatted.
func Example() {
	docs, err := kyaml.ToJSON(kyaml.YAML("kind: Namespace\napiVersion: v1\nmetadata:\n  name: myapp\n"))
	if err != nil {
		_, _ = fmt.Println(err)

		return
	}

	out, err := kyaml.FromJSON(docs...)
	if err != nil {
		_, _ = fmt.Println(err)

		return
	}

	_, _ = fmt.Print(string(out))
	// Output:
	// apiVersion: v1
	// kind: Namespace
	// metadata:
	//   name: myapp
}

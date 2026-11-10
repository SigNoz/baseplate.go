package expr

// Env is how an expression's names are typed. The zero Env is [Untyped].
type Env struct {
	typed bool
}

// Untyped returns an Env in which any name is allowed and nothing is typed
// until the program runs. Member access isn't nil-safe: a.b fails when a is
// missing. A [Compile]d program returns false for a nil result.
func Untyped() Env {
	return Env{}
}

// Typed returns an Env that types a program against each set of variables
// it runs with, such as the nested maps [Nest] returns. Any name is allowed,
// and member access is nil-safe, as [patcher.NilSafe] makes it. A program is
// typed against the top-level types of each set of variables, so
// "value > 80" is a type error when value is a string, and a nil or
// non-bool result of a [Compile]d program is an error.
func Typed() Env {
	return Env{typed: true}
}

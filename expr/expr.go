package expr

import (
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	upstream "github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/checker"
	"github.com/expr-lang/expr/compiler"
	"github.com/expr-lang/expr/conf"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/optimizer"
	"github.com/expr-lang/expr/parser"
	"github.com/expr-lang/expr/vm"

	"github.com/SigNoz/baseplate.go/errors"
	"github.com/SigNoz/baseplate.go/expr/patcher"
)

// maxShapes caps the programs a [TypedDynamic] program keeps, one per set of
// top-level variable types. Past it, programs are compiled on every run.
const maxShapes = 64

// Program is a compiled expression. It is safe for concurrent use.
type Program struct {
	source  *source
	env     Env
	boolean bool

	program *vm.Program // Untyped and TypedStatic.

	references []string
	values     struct {
		once     sync.Once
		programs map[string]*Program
	}

	names    []string // TypedDynamic: the top-level names the expression references, sorted.
	allNames bool     // TypedDynamic: $env or a caller's patch may read any name, so every name counts.
	shapes   sync.Map
	count    atomic.Int64
}

// Compile compiles an expression whose result is a bool.
//
// It returns a [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput]
// error with [CodeSyntaxInvalid] when src doesn't parse,
// [CodeNodesExceeded] when it has too many nodes, [CodeTypeInvalid] for
// a type error and [CodeResultNotBool] when its result is known not to be a
// bool. An error at a place in src wraps a [*PositionError].
func Compile(src string, env Env) (*Program, error) {
	return compile(src, env, true)
}

// CompileValue compiles an expression whose result can be any value. It
// returns the errors [Compile] does, except [CodeResultNotBool].
func CompileValue(src string, env Env) (*Program, error) {
	return compile(src, env, false)
}

func compile(text string, env Env, boolean bool) (*Program, error) {
	if env.varsErr != nil {
		return nil, env.varsErr
	}

	s, err := newSource(text)
	if err != nil {
		return nil, err
	}

	p := &Program{
		source:  s,
		env:     env,
		boolean: boolean,
	}

	config := conf.CreateNew()
	if env.nodes > 0 {
		upstream.MaxNodes(env.nodes)(config)
	}

	tree, err := parser.ParseWithConfig(s.upstream, config)
	if err != nil {
		return nil, p.parseError(err)
	}

	if err := p.lint(tree); err != nil {
		return nil, err
	}

	p.references = references(tree)

	if env.static {
		if err := p.checkNames(tree); err != nil {
			return nil, err
		}
	}

	if !env.dynamic {
		p.program, err = p.build(nil)
		if err != nil {
			return nil, err
		}

		return p, nil
	}

	p.allNames = len(env.patches) > 0

	ast.Walk(&tree.Node, &nameCollector{program: p})
	slices.Sort(p.names)
	p.names = slices.Compact(p.names)

	if _, err := p.programFor(map[string]any{}); err != nil {
		return nil, err
	}

	return p, nil
}

// Source returns the expression as it was written.
func (p *Program) Source() string {
	return p.source.text
}

// Bool runs the program with vars and returns its bool result. A [TypedDynamic]
// program takes a map[string]any, such as one from [Nest].
//
// It returns a [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput]
// error with [CodeRunFailed] when the run fails and [CodeResultNotBool]
// when the result isn't a bool. A [TypedDynamic] program can also return the
// errors of [Compile], since it is typed against vars.
func (p *Program) Bool(vars any) (bool, error) {
	out, err := p.Run(vars)
	if err != nil {
		return false, err
	}

	result, ok := out.(bool)
	if !ok {
		return false, errors.Newf(errors.TypeInvalidInput, CodeResultNotBool, "result is %T (%v), expected bool", out, out)
	}

	return result, nil
}

// Run runs the program with vars and returns its result. It returns the
// errors [Program.Bool] does, except [CodeResultNotBool].
func (p *Program) Run(vars any) (any, error) {
	program := p.program

	if p.env.static && reflect.TypeOf(vars) != reflect.TypeOf(p.env.vars) {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeRunFailed, "vars are %T, expected %T", vars, p.env.vars)
	}

	if p.env.dynamic {
		byName, ok := vars.(map[string]any)
		if !ok {
			return nil, errors.Newf(errors.TypeInvalidInput, CodeRunFailed, "vars are %T, expected map[string]any", vars)
		}

		var err error

		program, err = p.programFor(byName)
		if err != nil {
			return nil, err
		}
	}

	machine := vm.VM{MemoryBudget: p.env.memory}

	out, err := machine.Run(program, vars)
	if err != nil {
		code := CodeRunFailed
		if strings.Contains(err.Error(), "memory budget exceeded") {
			code = CodeMemoryExceeded
		}

		return nil, p.source.positionError(err, code)
	}

	return out, nil
}

// programFor returns the program typed against the top-level types of
// vars, compiling it on first use.
func (p *Program) programFor(vars map[string]any) (*vm.Program, error) {
	key := p.shape(vars)
	if cached, ok := p.shapes.Load(key); ok {
		program, _ := cached.(*vm.Program)

		return program, nil
	}

	program, err := p.build(vars)
	if err != nil {
		return nil, err
	}

	if p.count.Load() < maxShapes {
		if _, loaded := p.shapes.LoadOrStore(key, program); !loaded {
			p.count.Add(1)
		}
	}

	return program, nil
}

// shape is the top-level types, in vars, of the names the expression
// references: absent, nil or the value's type.
func (p *Program) shape(vars map[string]any) string {
	names := p.names
	if p.allNames {
		names = slices.Sorted(func(yield func(string) bool) {
			for name := range vars {
				if !yield(name) {
					return
				}
			}
		})
	}

	var key []byte
	for _, name := range names {
		key = append(key, name...)

		value, ok := vars[name]

		switch {
		case !ok:
			key = append(key, "\x00-"...)
		case value == nil:
			key = append(key, "\x00nil"...)
		default:
			t := reflect.TypeOf(value)
			key = append(key, "\x00"+t.PkgPath()+"\x00"+t.String()...)
		}

		key = append(key, '\x01')
	}

	return string(key)
}

// build compiles the upstream text as expr.Compile does. Only a
// [TypedDynamic] program passes vars.
func (p *Program) build(vars map[string]any) (*vm.Program, error) {
	var options []upstream.Option

	switch {
	case p.env.static:
		options = append(options, upstream.Env(p.env.vars))
	case p.env.dynamic:
		options = append(options, upstream.Env(vars), upstream.AllowUndefinedVariables())
	default:
		options = append(options, upstream.AllowUndefinedVariables())
	}

	for _, patch := range p.env.patches {
		options = append(options, upstream.Patch(patch))
	}

	switch {
	case p.env.static:
		nilSafe := patcher.NilSafeTyped{}
		name, function := nilSafe.Function()
		options = append(options, upstream.Function(name, function), upstream.Patch(nilSafe))
	case p.env.dynamic:
		nilSafe := patcher.NilSafe{}
		name, function := nilSafe.Function()
		options = append(options, upstream.Function(name, function), upstream.Patch(nilSafe))
	}

	if p.boolean {
		options = append(options, upstream.AsBool())
	}

	if p.env.nodes > 0 {
		options = append(options, upstream.MaxNodes(p.env.nodes))
	}

	config := conf.CreateNew()
	for _, option := range options {
		option(config)
	}

	for name := range config.Disabled {
		delete(config.Builtins, name)
	}

	config.Check()

	tree, err := parser.ParseWithConfig(p.source.upstream, config)
	if err != nil {
		return nil, p.parseError(err)
	}

	if _, err := new(checker.Checker).PatchAndCheck(tree, config); err != nil {
		if _, ok := errors.AsType[*file.Error](err); !ok {
			// The checker reports a result of the wrong kind without a position.
			return nil, errors.New(errors.TypeInvalidInput, CodeResultNotBool, err.Error())
		}

		return nil, p.source.positionError(err, CodeTypeInvalid)
	}

	if config.Optimize {
		if err := optimizer.Optimize(&tree.Node, config); err != nil {
			if fileError, ok := errors.AsType[*file.Error](err); ok {
				err = fileError.Bind(tree.Source)
			}

			return nil, p.source.positionError(err, CodeTypeInvalid)
		}
	}

	// A typed program is checked for a bool result but not cast to one, so a
	// nil result stays an error.
	if p.env.dynamic || p.env.static {
		config.Expect = reflect.Invalid
	}

	program, err := compiler.Compile(tree, config)
	if err != nil {
		return nil, p.source.positionError(err, CodeTypeInvalid)
	}

	return program, nil
}

// parseError gives an error from the parser its code.
func (p *Program) parseError(err error) error {
	if strings.Contains(err.Error(), "exceeds maximum allowed nodes") {
		return p.source.positionError(err, CodeNodesExceeded)
	}

	return p.source.positionError(err, CodeSyntaxInvalid)
}

// nameCollector records the top-level names an expression references.
type nameCollector struct {
	program *Program
}

func (c *nameCollector) Visit(node *ast.Node) {
	switch n := (*node).(type) {
	case *ast.IdentifierNode:
		if n.Value == "$env" {
			c.program.allNames = true
		}

		c.program.names = append(c.program.names, n.Value)
	case *ast.BuiltinNode:
		c.program.names = append(c.program.names, n.Name)
	}
}

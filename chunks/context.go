package chunks

import (
	"fmt"

	"github.com/osteele/liquid/expressions"
)

// Context is the evaluation context for chunk AST rendering.
type Context struct {
	vars map[string]interface{}
}

func NewContext(scope map[string]interface{}) Context {
	// The assign tag modifies the scope, so make a copy first.
	// TODO this isn't really the right place for this.
	vars := map[string]interface{}{}
	for k, v := range scope {
		vars[k] = v
	}
	return Context{vars}
}

// EvaluateExpr evaluates an expression within the template context.
func (c *Context) EvaluateExpr(source string) (out interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case expressions.InterpreterError:
				err = e
			default:
				// fmt.Println(string(debug.Stack()))
				panic(e)
			}
		}
	}()
	return expressions.EvaluateExpr(source, expressions.NewContext(c.vars))
}

func (c *Context) evaluateStatement(tag, source string) (interface{}, error) {
	return c.EvaluateExpr(fmt.Sprintf("%%%s %s", tag, source))
}

func makeExpressionValueFn(source string) (func(Context) (interface{}, error), error) {
	expr, err := expressions.Parse(source)
	if err != nil {
		return nil, err
	}
	return func(ctx Context) (interface{}, error) {
		return expr.Evaluate(expressions.NewContext(ctx.vars))
	}, nil
}

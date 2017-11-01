package ast

import (
	"fmt"

	"github.com/faiface/lambda/machine"
)

func CompileAll(globals map[string]Node) (map[string]machine.Expr, error) {
	var globalNames []string
	for name := range globals {
		globalNames = append(globalNames, name)
	}

	compiled := make([]machine.Expr, len(globalNames))

	compiledPtrs := make(map[string]*machine.Expr)
	for i, name := range globalNames {
		compiledPtrs[name] = &compiled[i]
	}

	for i, name := range globalNames {
		compiledFree, err := globals[name].Compile(compiledPtrs, nil)
		if err != nil {
			return nil, err
		}
		compiled[i] = compiledFree.Fill(nil)
	}

	compiledMap := make(map[string]machine.Expr)
	for i, name := range globalNames {
		compiledMap[name] = compiled[i]
	}

	return compiledMap, nil
}

type CompileError struct {
	Node Node
	Msg  string
}

func (err *CompileError) Error() string {
	return err.Msg
}

type Node interface {
	MetaInfo() interface{}
	HasFree(name string) bool
	Compile(globals map[string]*machine.Expr, free []string) (machine.FreeExpr, error)
}

type Var struct {
	Name string
	Meta interface{}
}

func (v *Var) MetaInfo() interface{}    { return v.Meta }
func (v *Var) HasFree(name string) bool { return name == v.Name }

func (v *Var) Compile(globals map[string]*machine.Expr, free []string) (machine.FreeExpr, error) {
	if len(free) != 1 || free[0] != v.Name {
		return nil, &CompileError{
			Node: v,
			Msg:  fmt.Sprintf("'%s' not defined", v.Name),
		}
	}
	return &machine.FreeVar{
		Meta: v.Meta,
	}, nil
}

type Abst struct {
	Bound string
	Body  Node
	Meta  interface{}
}

func (a *Abst) MetaInfo() interface{} { return a.Meta }
func (a *Abst) HasFree(name string) bool {
	return name != a.Bound && a.Body.HasFree(name)
}

func (a *Abst) Compile(globals map[string]*machine.Expr, free []string) (machine.FreeExpr, error) {
	if !a.Body.HasFree(a.Bound) {
		body, err := a.Body.Compile(globals, free)
		if err != nil {
			return nil, err
		}
		return &machine.FreeAbst{
			Used: false,
			Body: body,
			Meta: a.Meta,
		}, nil
	}

	freeWithBound := append([]string{a.Bound}, free...)
	body, err := a.Body.Compile(globals, freeWithBound)
	if err != nil {
		return nil, err
	}
	return &machine.FreeAbst{
		Used: true,
		Body: body,
		Meta: a.Meta,
	}, nil
}

type Appl struct {
	Left, Right Node
	Meta        interface{}
}

func (ap *Appl) MetaInfo() interface{} { return ap.Meta }
func (ap *Appl) HasFree(name string) bool {
	return ap.Left.HasFree(name) || ap.Right.HasFree(name)
}

func (ap *Appl) Compile(globals map[string]*machine.Expr, free []string) (machine.FreeExpr, error) {
	dirs := make([]machine.Dir, len(free))
	lfree, rfree := []string(nil), []string(nil)
	for i, name := range free {
		if ap.Left.HasFree(name) {
			dirs[i] |= machine.DirLeft
			lfree = append(lfree, name)
		}
		if ap.Right.HasFree(name) {
			dirs[i] |= machine.DirRight
			rfree = append(rfree, name)
		}
	}

	left, err := ap.Left.Compile(globals, lfree)
	if err != nil {
		return nil, err
	}
	right, err := ap.Right.Compile(globals, rfree)
	if err != nil {
		return nil, err
	}

	return &machine.FreeAppl{
		Dirs:  dirs,
		Left:  left,
		Right: right,
		Meta:  ap.Meta,
	}, nil
}

type Global struct {
	Name string
	Meta interface{}
}

func (g *Global) MetaInfo() interface{}    { return g.Meta }
func (g *Global) HasFree(name string) bool { return false }

func (g *Global) Compile(globals map[string]*machine.Expr, free []string) (machine.FreeExpr, error) {
	global, ok := globals[g.Name]
	if !ok {
		return nil, &CompileError{
			Node: g,
			Msg:  fmt.Sprintf("'%s' not defined", g.Name),
		}
	}
	return &machine.FreeRef{
		Ref:  global,
		Meta: g.Meta,
	}, nil
}

package machine

import (
	"math/big"
)

type Int struct {
	Value *big.Int
}

func (i Int) MetaInfo() interface{} { return nil }
func (i Int) IsNormal() bool        { return true }
func (i Int) Reduce() Expr          { return i }
func (i Int) Fill(ctx *Ctx) Expr {
	if ctx != nil {
		panic("int: context not empty")
	}
	return i
}

type IntBinOpType uint8

const (
	IntAdd IntBinOpType = iota
	IntSub
	IntMul
	IntDiv
	IntMod
)

type IntBinOp struct {
	Type  IntBinOpType
	First Int
}

func (ib *IntBinOp) MetaInfo() interface{} { return nil }
func (ib *IntBinOp) IsNormal() bool        { return true }
func (ib *IntBinOp) Reduce() Expr          { return ib }
func (ib *IntBinOp) Fill(ctx *Ctx) Expr {
	if ctx != nil {
		panic("int op: context not empty")
	}
	return ib
}
func (ib *IntBinOp) Apply(expr Expr) Expr {
	for !expr.IsNormal() {
		expr = expr.Reduce()
	}
	i, ok := expr.(Int)
	if !ok {
		panic("int bin op: operand not int")
	}
	if ib.First.Value == nil {
		return &IntBinOp{
			Type:  ib.Type,
			First: i,
		}
	}
	var result big.Int
	switch ib.Type {
	case IntAdd:
		result.Add(ib.First.Value, i.Value)
	case IntSub:
		result.Sub(ib.First.Value, i.Value)
	case IntMul:
		result.Mul(ib.First.Value, i.Value)
	case IntDiv:
		result.Div(ib.First.Value, i.Value)
	case IntMod:
		result.Mod(ib.First.Value, i.Value)
	}
	return Int{Value: &result}
}

type IntCmpOpType uint8

const (
	IntEq IntCmpOpType = iota
	IntNeq
	IntLess
	IntMore
	IntLessEq
	IntMoreEq
)

type IntCmpOp struct {
	Type  IntCmpOpType
	First Int
}

func (ic *IntCmpOp) MetaInfo() interface{} { return nil }
func (ic *IntCmpOp) IsNormal() bool        { return true }
func (ic *IntCmpOp) Reduce() Expr          { return ic }
func (ic *IntCmpOp) Fill(ctx *Ctx) Expr {
	return ic
}
func (ic *IntCmpOp) Apply(expr Expr) Expr {
	for !expr.IsNormal() {
		expr = expr.Reduce()
	}
	i, ok := expr.(Int)
	if !ok {
		panic("int cmp op: operand not int")
	}
	if ic.First.Value == nil {
		return &IntCmpOp{
			Type:  ic.Type,
			First: i,
		}
	}
	cmp := ic.First.Value.Cmp(i.Value)
	var result bool
	switch ic.Type {
	case IntEq:
		result = cmp == 0
	case IntNeq:
		result = cmp != 0
	case IntLess:
		result = cmp == -1
	case IntMore:
		result = cmp == +1
	case IntLessEq:
		result = cmp <= 0
	case IntMoreEq:
		result = cmp >= 0
	}
	if result {
		return True
	}
	return False
}

package machine

var ApplicationCallback func(left, right Expr)

type Ctx struct {
	Expr Expr
	Next *Ctx
}

func (ctx *Ctx) Cons(expr Expr) *Ctx {
	return &Ctx{
		Expr: expr,
		Next: ctx,
	}
}

type Dir uint8

const (
	DirLeft Dir = 1 + iota
	DirRight
	DirBoth
)

type Expr interface {
	MetaInfo() interface{}
	IsNormal() bool
	Reduce() Expr
}

type FreeExpr interface {
	MetaInfo() interface{}
	Fill(ctx *Ctx) Expr
}

type FreeVar struct {
	Meta interface{}
}

func (fv *FreeVar) MetaInfo() interface{} { return fv.Meta }

func (fv *FreeVar) Fill(ctx *Ctx) Expr {
	if ctx == nil {
		panic("free var: no context values")
	}
	if ctx.Next != nil {
		panic("free var: context has more than one value")
	}
	return ctx.Expr
}

type FreeAbst struct {
	Used bool
	Body FreeExpr
	Meta interface{}
}

func (fa *FreeAbst) MetaInfo() interface{} { return fa.Meta }

func (fa *FreeAbst) Fill(ctx *Ctx) Expr {
	return &Abst{
		Ctx:  ctx,
		Used: fa.Used,
		Body: fa.Body,
		Meta: fa.Meta,
	}
}

type FreeAppl struct {
	Dirs        []Dir
	Left, Right FreeExpr
	Meta        interface{}
}

func (fap *FreeAppl) MetaInfo() interface{} { return fap.Meta }

func (fap *FreeAppl) Fill(ctx *Ctx) Expr {
	lctx, rctx := distribute(fap.Dirs, ctx)
	left, right := fap.Left.Fill(lctx), fap.Right.Fill(rctx)
	return &Appl{
		Left:  left,
		Right: right,
		Meta:  fap.Meta,
	}
}

func distribute(dirs []Dir, ctx *Ctx) (left, right *Ctx) {
	for _, dir := range dirs {
		if ctx == nil {
			panic("distribute: context too short")
		}
		if dir&DirLeft != 0 {
			left = left.Cons(ctx.Expr)
		}
		if dir&DirRight != 0 {
			right = right.Cons(ctx.Expr)
		}
		ctx = ctx.Next
	}
	if ctx != nil {
		panic("distribute: context too long")
	}
	return left, right
}

type FreeRef struct {
	Ref  *Expr
	Meta interface{}
}

func (fr *FreeRef) MetaInfo() interface{} { return fr.Meta }
func (fr *FreeRef) Fill(ctx *Ctx) Expr {
	if ctx != nil {
		panic("free ref: context not empty")
	}
	return &Ref{
		Ref:  fr.Ref,
		Meta: fr.Meta,
	}
}

type Abst struct {
	Ctx  *Ctx
	Used bool
	Body FreeExpr
	Meta interface{}
}

func (a *Abst) MetaInfo() interface{} { return a.Meta }
func (a *Abst) IsNormal() bool        { return true }
func (a *Abst) Reduce() Expr          { return a }

type Appl struct {
	Left, Right Expr
	Memo        Expr
	Meta        interface{}
}

func (ap *Appl) MetaInfo() interface{} { return ap.Meta }
func (ap *Appl) IsNormal() bool        { return false }

func (ap *Appl) Reduce() (reduced Expr) {
	if ap.Memo != nil {
		ap.Memo = ap.Memo.Reduce()
		return ap.Memo
	}
	defer func() {
		ap.Memo = reduced
		ap.Left = nil
		ap.Right = nil
	}()
	if !ap.Left.IsNormal() {
		return &Appl{
			Left:  ap.Left.Reduce(),
			Right: ap.Right,
		}
	}
	abst, ok := ap.Left.(*Abst)
	if !ok {
		panic("reduce appl: left side not abst")
	}
	ctx := abst.Ctx
	if abst.Used {
		ctx = ctx.Cons(ap.Right)
	}
	if ApplicationCallback != nil {
		ApplicationCallback(ap.Left, ap.Right)
	}
	return abst.Body.Fill(ctx)
}

type Ref struct {
	Ref  *Expr
	Meta interface{}
}

func (r *Ref) MetaInfo() interface{} { return r.Meta }
func (r *Ref) IsNormal() bool        { return false }
func (r *Ref) Reduce() Expr          { return *r.Ref }

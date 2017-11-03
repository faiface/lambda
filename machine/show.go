package machine

func ShowExpr(repr func(interface{}) string, expr Expr) string {
	switch expr := expr.(type) {
	case *Ref:
		return repr(expr.MetaInfo())
	case *Abst:
		return "(λ" + repr(expr.MetaInfo()) + " " + ShowFreeExpr(repr, expr.Body) + ")"
	case *Appl:
		return "(" + ShowExpr(repr, expr.Left) + " " + ShowExpr(repr, expr.Right) + ")"
	default:
		return "(???)"
	}
}

func ShowFreeExpr(repr func(interface{}) string, free FreeExpr) string {
	switch free := free.(type) {
	case *FreeVar:
		return repr(free.MetaInfo())
	case *FreeRef:
		return repr(free.MetaInfo())
	case *FreeAbst:
		return "(λ" + repr(free.MetaInfo()) + " " + ShowFreeExpr(repr, free.Body) + ")"
	case *FreeAppl:
		return "(" + ShowFreeExpr(repr, free.Left) + " " + ShowFreeExpr(repr, free.Right) + ")"
	default:
		return "(???)"
	}
}

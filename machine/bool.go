package machine

var True = &Abst{
	Used: true,
	Body: &FreeAbst{
		Used: false,
		Body: &FreeVar{},
	},
}

var False = &Abst{
	Used: false,
	Body: &FreeAbst{
		Used: true,
		Body: &FreeVar{},
	},
}

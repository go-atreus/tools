{{$svrType := .ServiceType}}
{{$svrName := .ServiceName}}

type {{.ServiceType}}Impl struct{
	cc {{.ServiceType}}Client
}

func New{{.ServiceType}}Impl (client {{.ServiceType}}Client) *{{.ServiceType}}Impl {
	return &{{.ServiceType}}Impl{client}
}

{{range .Methods}}
func (c *{{$svrType}}Impl) {{.Name}}(ctx context.Context, in *{{.Request}}) (*{{.Reply}}, error) {
    return c.cc.{{.Name}}(ctx, in)
}
{{end}}
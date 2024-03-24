{{$svrType := .ServiceType}}
{{$svrName := .ServiceName}}

type {{.ServiceType}}Impl struct{
	cc {{.ServiceType}}Client
}

func New{{.ServiceType}}Impl (conn *grpc.ClientConn) *{{.ServiceType}}Impl {
	return &{{.ServiceType}}Impl{cc: New{{.ServiceType}}Client(conn)}
}

{{range .Methods}}
func (c *{{$svrType}}Impl) {{.Name}}(ctx context.Context, in *{{.Request}}) (*{{.Reply}}, error) {
    return c.cc.{{.Name}}(ctx, in)
}
{{end}}
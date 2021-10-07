package day5

import (
	"fmt"
	"html/template"
	"net/http"
)

/*提供一个查看每个services被调用次数的http服务
	path = /PRC/ServicesCallNums
	Method = Get
*/

const HtmlText = `<html>
	<body>
	<title>GeeRPC Services</title>
	{{range .}}
	<hr>
	Service {{.Name}}
	<hr>
		<table>
		<th align=center>Method</th><th align=center>the Number of calls</th>
		{{range $name, $mtype := .Methods	}}
			<tr>
			<td align=left font=fixed>{{$name}}({{$mtype.Argv}}, {{$mtype.Reply}}) error</td>
			<td align=center>{{$mtype.NumCalls}}</td>
			</tr>
		{{end}}
		</table>
	{{end}}
	</body>
	</html>`
type HttpCallNums struct {
	 *Server
}
type objects struct {
	Name string
	Methods	map[string]*Method
}

func (server HttpCallNums) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	//先获取模板
	module := template.Must(template.New("callnum").Parse(HtmlText))
	//获取所有的服务
	var services []objects
	server.Services.Range(func(key, value interface{}) bool {
		v:=value.(*Object)
		services=append(services,objects{
			Name:    key.(string),
			Methods: v.methods,
		})
		return true
	})
	err := module.Execute(w, services)
	if err!=nil{
		fmt.Fprintln(w, "rpc: error executing template:", err.Error())
	}
}


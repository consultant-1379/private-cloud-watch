package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/erixzone/crux/pkg/alembicpb"
	"github.com/erixzone/crux/pkg/alembicpb/new"
)

var output string
var noisy bool
var testFlag = true
var templateServices *template.Template

func init() {
	flag.StringVar(&output, "o", ".", "where resulting files go")
	flag.BoolVar(&noisy, "d", false, "be noisy about the input")
}

func main() {
	var err error
	templateServices, err = template.New("gen").Parse(codeServices)
	if err != nil {
		panic(err)
	}
	flag.Parse()
	for _, a := range flag.Args() {
		b, err := ioutil.ReadFile(a)
		if err != nil {
			fmt.Printf("can't read %s: %s\n", a, err.Error())
			os.Exit(1)
		}
		p, err := alembicpbnew.New(a, string(b), noisy)
		//fmt.Printf("got '%s' -> %s\n", b, p.String())
		dump(p, output, a)
	}
}

type GenX struct {
	Tool    string
	Srcfile string
	Package string
	Proto   *alembicpb.ProtoPkg
}

var codeServices = `// Code generated from {{.Srcfile}} by {{.Tool}}; DO NOT EDIT.

package {{.Package}}

import (
	"google.golang.org/grpc"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsvc"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/rucklib"
)

{{range $e := .Proto.Services}}// Code block for service {{$e.Name}}

type {{$e.Name}}ServerStarter struct {
	xxx pb.{{$e.Name}}Server
}

// RegisterServer : rhubarb
func (d {{$e.Name}}ServerStarter) RegisterServer(s *grpc.Server) {
	pb.Register{{$e.Name}}Server(s, d.xxx)
}

// Name : rhubarb
func (d {{$e.Name}}ServerStarter) Name() string {
	return "{{$e.Name}}"
}

// Start{{$e.Name}}Server starts and registers the {{$e.Name}} (grpc whitelist) server; exit on error for now
func Start{{$e.Name}}Server(fid *idutils.NodeIDT, serviceRev, address string, port int, xxx pb.{{$e.Name}}Server, quit <-chan bool, reeveapi rucklib.ReeveAPI) idutils.NetIDT {
	return grpcsvc.StartGrpcServer(fid , serviceRev, address, port, {{$e.Name}}ServerStarter{xxx}, quit, reeveapi)
}

// Connect{{$e.Name}} is how you connect to a {{$e.Name}}
func Connect{{$e.Name}}(dest idutils.NetIDT, signer interface{}, clilog clog.Logger) (pb.{{$e.Name}}Client, *crux.Err) {
	conn, err := grpcsvc.NewGrpcClient(dest, signer, clilog, "{{$e.Name}}")
	if err != nil {
		return nil, err
	}
	return pb.New{{$e.Name}}Client(conn), nil
}
{{end}}`

func dump(p *alembicpb.ProtoPkg, output, a string) {
	aOrig := a
	proto := ".proto"
	if a[len(a)-len(proto):] == proto {
		a = a[:len(a)-len(proto)]
	}
	name := filepath.Join(output, a+".grpc.go")
	o, err := os.Create(name)
	if err != nil {
		fmt.Printf("%s: %s\n", name, err.Error())
		return
	}
	params := GenX{
		Tool:    "tools/server/main.go",
		Srcfile: aOrig,
		Package: "ruckgen", // erk! fixme
		Proto:   p,
	}
	err = templateServices.Execute(o, params)
	if err != nil {
		panic(err)
	}
	o.Close()
}

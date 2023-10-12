package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"strings"
	"syscall/js"

	promise "github.com/nlepage/go-js-promise"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
)

func (n *XheWireguard) ListenTCP(this js.Value, args []js.Value) (p any) {
	p, resolve, reject := promise.New()
	var port int = 80
	if len(args) >= 1 {
		port = args[0].Int()
	}
	go func() (err error) {
		defer err0.Then(&err, nil, func() {
			reject(err.Error())
		})
		addr := netip.AddrPortFrom(netip.Addr{}, uint16(port))
		fa, pn := convertToFullAddr(n.nic, addr)
		l := try.To1(gonet.ListenTCP(n.stack, fa, pn))
		s := NewTCPServer(l)
		resolve(s.ToJS())
		return
	}()
	return
}

type TCPServer struct {
	listener *gonet.TCPListener
	mux      http.Handler
}

func NewTCPServer(l *gonet.TCPListener) *TCPServer {
	return &TCPServer{
		listener: l,
	}
}

func (l *TCPServer) ToJS() (root js.Value) {
	root = js.Global().Get("Object").New()
	root.Set("Serve", js.FuncOf(l.Serve))
	root.Set("ServeHTTP", js.FuncOf(l.ServeHTTP))
	root.Set("Close", js.FuncOf(l.Close))
	root.Set("ServeReady", js.FuncOf(l.ServeReady))
	root.Set("ReverseProxy", js.FuncOf(l.ReverseProxy))
	root.Set("HandleEval", js.FuncOf(l.HandleEval))
	return
}

func (l *TCPServer) Serve(this js.Value, args []js.Value) (p any) {
	p, resolve, reject := promise.New()
	go func() (err error) {
		defer err0.Then(&err, nil, func() {
			reject(err.Error())
		})
		l.mux = http.NewServeMux()
		try.To(http.Serve(l.listener, l.mux))
		resolve("exited")
		return
	}()
	return
}

func (l *TCPServer) ServeHTTP(this js.Value, args []js.Value) (p any) {
	p, resolve, reject := promise.New()
	go func() (err error) {
		defer err0.Then(&err, nil, func() {
			reject(err.Error())
		})
		if len(args) == 0 {
			reject("rqeuire http server implement {fetch(Request):Response}")
			return
		}
		l.mux = NewHono(args[0])
		try.To(http.Serve(l.listener, l.mux))
		resolve("exited")
		return
	}()
	return
}

func (l *TCPServer) ServeReady(this js.Value, args []js.Value) any {
	return l.mux != nil
}

func (l *TCPServer) Close(this js.Value, args []js.Value) (p any) {
	p, resolve, reject := promise.New()
	go func() {
		if err := l.listener.Close(); err != nil {
			reject(err.Error())
			return
		}
		resolve("closed")
	}()
	return
}

func (l *TCPServer) ReverseProxy(this js.Value, args []js.Value) (p any) {
	p, resolve, reject := promise.New()
	go func() (err error) {
		defer err0.Then(&err, nil, func() {
			reject(err.Error())
		})
		if len(args) < 2 {
			reject("path and host is required")
			return
		}
		path := args[0].String()
		remote := try.To1(url.Parse(args[1].String()))

		director := httputil.NewSingleHostReverseProxy(remote).Director
		var proxy = &httputil.ReverseProxy{
			Rewrite: func(pr *httputil.ProxyRequest) {
				r := pr.Out
				r.Header.Del("User-Agent")
				injectJsFetchOptions(r)
				director(r)
			},
		}

		var handler http.Handler = proxy
		if path != "/" {
			handler = http.StripPrefix(path, handler)
		}
		mux, ok := l.mux.(*http.ServeMux)
		if !ok {
			reject("not *http.ServeMux")
			return
		}
		mux.Handle(path, handler)
		resolve(path)
		return
	}()
	return
}

const jsFetchOptInPrefix = "Js.fetch."
const jsFetchOptPrefix = "js.fetch:"

func injectJsFetchOptions(r *http.Request) {
	for k, vv := range r.Header {
		if strings.HasPrefix(k, jsFetchOptInPrefix) {
			r.Header.Del(k)
			k = jsFetchOptPrefix + k[len(jsFetchOptInPrefix):]
			r.Header[k] = vv
		}
	}
}

func (l *TCPServer) HandleEval(this js.Value, args []js.Value) (p any) {
	p, resolve, reject := promise.New()
	go func() (err error) {
		defer err0.Then(&err, nil, func() {
			reject(err.Error())
		})
		path := args[0].String()
		mux, ok := l.mux.(*http.ServeMux)
		if !ok {
			reject("not *http.ServeMux")
			return
		}
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			var err error
			defer err0.Then(&err, nil, func() {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, err.Error())
			})
			content := try.To1(io.ReadAll(r.Body))
			j := try.To1(Eval(string(content)))
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, j)
		})
		resolve(path)
		return
	}()
	return
}

func Eval(content string) (s string, err error) {
	f := js.Global().Get("Function").New("resolve", "reject", fmt.Sprintf(`"use strict";%s;resolve();`, content))
	p := js.Global().Get("Promise").New(f)
	v, err := promise.Await(p)
	if err != nil {
		return
	}
	s = js.Global().Get("JSON").Call("stringify", v).String()
	return
}

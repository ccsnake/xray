package main

import (
	"context"
	"fmt"
	"github.com/ccsnake/xray"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"net"
	"net/http"
	"time"
)

func main() {
	co:= xray.NewUDPCollector(":2000")

	tracer := xray.NewTracer(co, "app1")
	opentracing.SetGlobalTracer(tracer)

	go httptest()
	for {
		act1()
	}
}

func act1() {
	sp := opentracing.StartSpan("act1")
	defer sp.Finish()
	act2(opentracing.ContextWithSpan(context.TODO(), sp))
	sp.SetTag("act1.ok", true)
	sp.LogKV("key", "value")
	go call(opentracing.ContextWithSpan(context.TODO(), sp), ":12233")
}

func act2(ctx context.Context) {
	sp, cc := opentracing.StartSpanFromContext(ctx, "act2")
	defer sp.Finish()
	sp.SetTag("act2.ok", true)
	sp.SetTag("http.code", 200)
	go call(cc, "127.0.0.1:12233")
	time.Sleep(time.Millisecond * 30)
}

func httptest() {
	ln, err := net.Listen("tcp", ":12233")
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/test", func(rw http.ResponseWriter, r *http.Request) {
		var sp opentracing.Span
		opName := r.URL.Path
		// Attempt to join a trace by getting trace context from the headers.
		wireContext, err := opentracing.GlobalTracer().Extract(
			opentracing.TextMap,
			opentracing.HTTPHeadersCarrier(r.Header))
		if err != nil {
			// If for whatever reason we can't join, go ahead an start a new root span.
			sp = opentracing.StartSpan(opName)
		} else {
			sp = opentracing.StartSpan(opName, opentracing.ChildOf(wireContext))
		}
		defer sp.Finish()
		sp.SetTag("tag", "dodo")
		sp.LogKV("meta", "bla")

		fmt.Fprint(rw, "ok!")
	})
	go http.Serve(ln, nil)
}

func call(ctx context.Context, addr string) error {
	url := fmt.Sprintf("http://%s/test", addr)

	sp, _ := opentracing.StartSpanFromContext(ctx, "GET "+url)
	defer sp.Finish()

	cli := http.Client{Timeout: time.Second}
	req, _ := http.NewRequest("GET", url, nil)

	if err := sp.Tracer().Inject(sp.Context(),
		opentracing.TextMap,
		opentracing.HTTPHeadersCarrier(req.Header)); err != nil {
		return err
	}
	sp.SetTag("tag", "dodo")
	sp.LogKV("meta", "bla")

	if resp, err := cli.Do(req); err != nil {
		sp.SetTag("error", err.Error())
		ext.Error.Set(sp, true)
		fmt.Println("err", err)
		return err
	} else {
		sp.SetTag("http.code", resp.StatusCode)
	}

	return nil
}

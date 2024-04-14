package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/aaronland/go-http-server"
	"github.com/aaronland/go-http-server/handler"	
	"github.com/jtacoma/uritemplates"
	"github.com/sfomuseum/go-flags/flagset"
	"github.com/whosonfirst/go-whosonfirst-findingaid/v2/resolver"
	"github.com/whosonfirst/go-whosonfirst-uri"
)

func main() {

	var server_uri string
	var resolver_uri string
	var data_uri_template string

	fs := flagset.NewFlagSet("server")

	fs.StringVar(&server_uri, "server-uri", "http://localhost:8080", "...")
	fs.StringVar(&resolver_uri, "resolver-uri", "https://data.whosonfirst.org/findingaid", "...")
	fs.StringVar(&data_uri_template, "data-uri-template", "https://raw.githubusercontent.com/whosonfirst-data/{repo}/master/data", "...")

	flagset.Parse(fs)

	err := flagset.SetFlagsFromEnvVars(fs, "WHOSONFIRST")

	if err != nil {
		slog.Error("Failed to assign flags from environment variables", "error", err)
		os.Exit(1)
	}
	
	ctx := context.Background()

	//

	r, err := resolver.NewResolver(ctx, resolver_uri)

	if err != nil {
		slog.Error("Failed to create resolver", "error", err)
		os.Exit(1)
	}

	t, err := uritemplates.Parse(data_uri_template)

	if err != nil {
		slog.Error("Failed to create URI template", "error", err)
		os.Exit(1)
	}

	data_opts := &RedirectHandlerOptions{
		Resolver: r,
		Template: t,
	}

	data_handler, err := NewRedirectHandler(data_opts)

	if err != nil {
		slog.Error("Failed to create data handler", "error", err)
		os.Exit(1)
	}

	//

	null_handler := handler.NullHandler()
	
	//
	
	mux := http.NewServeMux()
	mux.Handle("/favicon.ico", null_handler)	
	mux.Handle("/", data_handler)

	s, err := server.NewServer(ctx, server_uri)

	if err != nil {
		slog.Error("Failed to create new server", "error", err)
		os.Exit(1)
	}

	slog.Info("Listening for requests", "address", s.Address())

	err = s.ListenAndServe(ctx, mux)

	if err != nil {
		slog.Error("Failed to serve requests", "error", err)
		os.Exit(1)
	}

}

type RedirectHandlerOptions struct {
	Resolver resolver.Resolver
	Template *uritemplates.UriTemplate
}

func NewRedirectHandler(opts *RedirectHandlerOptions) (http.Handler, error) {

	fn := func(rsp http.ResponseWriter, req *http.Request) {

		ctx := req.Context()

		logger := slog.Default()
		logger = logger.With("request", req.URL)

		// TBD: cache this?

		id, uri_args, err := uri.ParseURI(req.URL.Path)

		if err != nil {
			logger.Error("Failed to parse path", "error", err)
			http.Error(rsp, "Bad request", http.StatusBadRequest)
			return
		}

		logger = logger.With("id", id)

		repo, err := opts.Resolver.GetRepo(ctx, id)

		if err != nil {
			logger.Error("Failed to derive respository", "error", err)
			http.Error(rsp, "Not found", http.StatusNotFound)
			return
		}

		rel_path, err := uri.Id2RelPath(id, uri_args)

		if err != nil {
			logger.Error("Failed to derive relative path", "error", err)
			http.Error(rsp, "Internal server error", http.StatusInternalServerError)
			return
		}

		values := map[string]interface{}{
			"repo": repo,
		}

		root_uri, err := opts.Template.Expand(values)

		if err != nil {
			logger.Error("Failed to expand template", "error", err)
			http.Error(rsp, "Internal server error", http.StatusInternalServerError)
			return
		}

		data_uri, err := url.JoinPath(root_uri, rel_path)

		if err != nil {
			logger.Error("Failed to derive final data URI", "error", err)
			http.Error(rsp, "Internal server error", http.StatusInternalServerError)
			return
		}

		logger.Info("redirect", "url", data_uri)

		http.Redirect(rsp, req, data_uri, http.StatusSeeOther)
		return

	}

	return http.HandlerFunc(fn), nil
}

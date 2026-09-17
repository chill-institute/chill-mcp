// Command chill-mcp serves chill.institute tools over MCP.
//
//	chill-mcp stdio [--profile NAME] [--api-url URL]   local agent using the chilly profile
//	chill-mcp http                                       hosted stateless Streamable HTTP
//	chill-mcp health                                     probe a running http server
//	chill-mcp version
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/chill-institute/chill-cli/v2/pkg/config"
	"github.com/chill-institute/chill-cli/v2/pkg/rpc"
	"github.com/chill-institute/chill-mcp/internal/buildinfo"
	"github.com/chill-institute/chill-mcp/internal/server"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	envAPIBaseURL = "CHILL_ENGINE_BASE_URL"
	envListenHost = "CHILL_LISTEN_HOST"
	envListenPort = "CHILL_LISTEN_PORT"
	defaultHost   = "127.0.0.1"
	defaultPort   = "7100"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: chill-mcp <stdio|http|health|version>")
		return 2
	}
	logger := slog.New(slog.NewTextHandler(stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		// After the first signal starts a graceful stop, a second one must not be swallowed.
		<-ctx.Done()
		stop()
	}()

	var err error
	switch args[0] {
	case "stdio":
		err = runStdio(ctx, args[1:], logger)
	case "http":
		err = runHTTP(ctx, logger)
	case "health":
		err = runHealth(ctx)
	case "version":
		info := buildinfo.Current()
		_, _ = fmt.Fprintf(stdout, "chill-mcp %s (%s)\n", info.Version, info.Commit)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
	if errors.Is(err, errUsage) {
		return 2
	}
	if err != nil {
		logger.Error("chill-mcp failed", "error", err.Error())
		return 1
	}
	return 0
}

var errUsage = errors.New("usage")

func apiClient(baseURL string) *rpc.Client {
	return rpc.NewClient(baseURL, nil, rpc.WithClientName(server.ClientName), rpc.WithClientVersion(buildinfo.Current().Version))
}

func runStdio(ctx context.Context, args []string, logger *slog.Logger) error {
	flags := flag.NewFlagSet("stdio", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	profile := flags.String("profile", "", "chilly profile name")
	apiURL := flags.String("api-url", "", "override API base URL")
	configPath := flags.String("config", "", "chilly config file path")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			flags.SetOutput(os.Stderr)
			flags.Usage()
			return errUsage
		}
		return fmt.Errorf("stdio flags: %w", err)
	}

	cfg, err := loadProfile(*profile, *configPath)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*apiURL) != "" {
		cfg.APIBaseURL = strings.TrimSpace(*apiURL)
	}
	token := strings.TrimSpace(cfg.AuthToken)
	if token == "" {
		return errors.New("missing auth token: run `chilly auth login` first")
	}

	srv, err := server.New(server.Options{
		Version: buildinfo.Current().Version,
		API:     apiClient(cfg.APIBaseURL),
		Token: func(context.Context, *mcp.CallToolRequest) (string, error) {
			return token, nil
		},
	})
	if err != nil {
		return err
	}
	logger.Info("serving chill.institute MCP over stdio", "api", redactedURL(cfg.APIBaseURL))
	return srv.MCP().Run(ctx, &mcp.StdioTransport{})
}

func loadProfile(profile string, configPath string) (config.Config, error) {
	path := strings.TrimSpace(configPath)
	if path == "" {
		resolved, err := config.ResolveProfile(profile, buildinfo.Current().Version == "dev")
		if err != nil {
			return config.Config{}, err
		}
		path, err = config.DefaultPath(resolved)
		if err != nil {
			return config.Config{}, err
		}
	}
	store, err := config.NewStore(path)
	if err != nil {
		return config.Config{}, err
	}
	cfg, err := store.Load()
	if err != nil {
		return config.Config{}, err
	}
	return cfg.Normalized(), nil
}

func listenAddress() string {
	host := strings.TrimSpace(os.Getenv(envListenHost))
	if host == "" {
		host = defaultHost
	}
	port := strings.TrimSpace(os.Getenv(envListenPort))
	if port == "" {
		port = defaultPort
	}
	return net.JoinHostPort(host, port)
}

func runHTTP(ctx context.Context, logger *slog.Logger) error {
	srv, err := server.New(server.Options{
		Version: buildinfo.Current().Version,
		API:     apiClient(os.Getenv(envAPIBaseURL)),
		Token:   server.HTTPTokenSource,
	})
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Addr:              listenAddress(),
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	errs := make(chan error, 1)
	go func() {
		logger.Info("serving chill.institute MCP over http", "addr", httpServer.Addr, "path", server.MCPPath)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- err
		}
		close(errs)
	}()
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
}

func runHealth(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+listenAddress()+server.HealthPath, nil)
	if err != nil {
		return err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("health returned status %d", response.StatusCode)
	}
	return nil
}

func redactedURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "invalid"
	}
	return parsed.Redacted()
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

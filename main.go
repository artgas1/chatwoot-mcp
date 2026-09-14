package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gobenpark/chatwoot-mcp/chatwoot"
	"github.com/gobenpark/chatwoot-mcp/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	version = "dev"
	commit  = "none"
)

type transportConfig struct {
	mode string
	host string
	port int
}

func loadTransportConfig() (transportConfig, error) {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("MCP_TRANSPORT")))
	if mode == "" {
		mode = "stdio"
	}
	if mode != "stdio" && mode != "http" {
		return transportConfig{}, fmt.Errorf("MCP_TRANSPORT must be stdio or http, got %q", mode)
	}

	host := strings.TrimSpace(os.Getenv("MCP_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	if mode == "http" && !isLoopbackHost(host) {
		return transportConfig{}, fmt.Errorf("MCP_HOST must be a loopback address, got %q", host)
	}

	portValue := strings.TrimSpace(os.Getenv("MCP_PORT"))
	if portValue == "" {
		portValue = "3000"
	}
	port, err := strconv.Atoi(portValue)
	if err != nil || port < 1 || port > 65535 {
		return transportConfig{}, fmt.Errorf("MCP_PORT must be an integer from 1 to 65535, got %q", portValue)
	}
	return transportConfig{mode: mode, host: host, port: port}, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func newMCPServer(client *chatwoot.Client) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "chatwoot-mcp",
		Version: version,
	}, nil)

	tools.RegisterConversationTools(server, client)
	tools.RegisterContactTools(server, client)
	tools.RegisterAccountTools(server, client)
	tools.RegisterReportTools(server, client)
	tools.RegisterAutomationTools(server, client)
	tools.RegisterAuditTools(server, client)
	tools.RegisterAgentBotTools(server, client)
	tools.RegisterIntegrationTools(server, client)
	tools.RegisterHelpCenterTools(server, client)
	return server
}

func newHTTPHandler(server *mcp.Server) http.Handler {
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{SessionTimeout: 30 * time.Minute})
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	return mux
}

func run(ctx context.Context) error {
	config, err := loadTransportConfig()
	if err != nil {
		return err
	}

	client, err := chatwoot.NewClientFromEnv()
	if err != nil {
		return fmt.Errorf("initialize Chatwoot client: %w", err)
	}
	server := newMCPServer(client)

	if config.mode == "stdio" {
		log.Println("Starting chatwoot-mcp server (stdio)...")
		return server.Run(ctx, &mcp.StdioTransport{})
	}

	address := net.JoinHostPort(config.host, strconv.Itoa(config.port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", address, err)
	}
	httpServer := &http.Server{
		Handler:           newHTTPHandler(server),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP shutdown error: %v", err)
		}
	}()

	log.Printf("Starting chatwoot-mcp server (http) on http://%s/mcp...", address)
	if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}

func main() {
	log.SetOutput(os.Stderr)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

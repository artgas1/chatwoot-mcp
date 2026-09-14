package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"github.com/gobenpark/chatwoot-mcp/chatwoot"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func configuredClient(t *testing.T) *chatwoot.Client {
	t.Helper()
	t.Setenv("CHATWOOT_URL", "https://chatwoot.example.test")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	client, err := chatwoot.NewClientFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestLoadTransportConfigDefaultsToStdio(t *testing.T) {
	for _, name := range []string{"MCP_TRANSPORT", "MCP_HOST", "MCP_PORT"} {
		t.Setenv(name, "")
	}
	config, err := loadTransportConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.mode != "stdio" || config.host != "127.0.0.1" || config.port != 3000 {
		t.Fatalf("unexpected defaults: %+v", config)
	}
}

func TestLoadTransportConfigRejectsUnsafeHTTPSettings(t *testing.T) {
	t.Setenv("MCP_TRANSPORT", "http")
	t.Setenv("MCP_HOST", "0.0.0.0")
	if _, err := loadTransportConfig(); err == nil {
		t.Fatal("non-loopback MCP_HOST was accepted")
	}
	t.Setenv("MCP_HOST", "127.0.0.1")
	t.Setenv("MCP_PORT", "70000")
	if _, err := loadTransportConfig(); err == nil {
		t.Fatal("out-of-range MCP_PORT was accepted")
	}
}

func TestHTTPTransportSupportsParallelClients(t *testing.T) {
	server := newMCPServer(configuredClient(t))
	httpServer := httptest.NewServer(newHTTPHandler(server))
	defer httpServer.Close()

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, nil)
			session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
				Endpoint: httpServer.URL + "/mcp",
			}, nil)
			if err != nil {
				errs <- err
				return
			}
			defer session.Close()
			result, err := session.ListTools(context.Background(), nil)
			if err != nil {
				errs <- err
				return
			}
			if len(result.Tools) != 67 {
				errs <- fmt.Errorf("tool count = %d, want 67", len(result.Tools))
				return
			}
			names := make([]string, 0, len(result.Tools))
			for _, tool := range result.Tools {
				names = append(names, tool.Name)
			}
			for _, required := range []string{"list_conversations", "list_contacts", "list_agents", "list_audit_logs"} {
				if !slices.Contains(names, required) {
					errs <- fmt.Errorf("missing tool: %s", required)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func TestHTTPTransportRejectsForeignHost(t *testing.T) {
	server := newMCPServer(configuredClient(t))
	httpServer := httptest.NewServer(newHTTPHandler(server))
	defer httpServer.Close()

	req, err := http.NewRequest(http.MethodPost, httpServer.URL+"/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "evil.example"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign Host status = %d, want 403", resp.StatusCode)
	}
}

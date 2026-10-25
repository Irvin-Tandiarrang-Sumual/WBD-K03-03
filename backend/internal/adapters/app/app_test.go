package app

import (
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
)

func TestApplicationCloseCallsTargetCleanup(t *testing.T) {
	called := false
	appInstance := &application{closeTarget: func() { called = true }}
	appInstance.close()
	if !called {
		t.Fatal("closeTarget was not called")
	}

	(&application{}).close()
}

func TestBuildTargetDatabaseInfrastructure(t *testing.T) {
	configuration := &config.Config{
		Security: config.SecurityConfig{DatabaseCredentialKey: "0123456789abcdef0123456789abcdef"},
	}

	infrastructure, err := buildTargetDatabaseInfrastructure(configuration)
	if err != nil {
		t.Fatalf("buildTargetDatabaseInfrastructure() error = %v", err)
	}
	if infrastructure.credentialCipher == nil || infrastructure.connectionTester == nil || infrastructure.poolManager == nil || infrastructure.databaseExecutor == nil || infrastructure.close == nil {
		t.Fatalf("incomplete infrastructure: %+v", infrastructure)
	}
	infrastructure.close()
}

func TestBuildTargetDatabaseInfrastructureRejectsInvalidKey(t *testing.T) {
	configuration := &config.Config{
		Security: config.SecurityConfig{DatabaseCredentialKey: "too-short"},
	}
	if _, err := buildTargetDatabaseInfrastructure(configuration); err == nil {
		t.Fatal("expected invalid AES key to be rejected")
	}
}

func TestBuildHandlersConstructsAllHTTPHandlers(t *testing.T) {
	configuration := &config.Config{Session: config.SessionConfig{CookieSecure: true}}
	services := &applicationServices{}

	handlers := buildHandlers(configuration, services)
	if handlers == nil || handlers.auth == nil || handlers.user == nil || handlers.group == nil || handlers.database == nil || handlers.databaseExecutor == nil || handlers.migration == nil {
		t.Fatalf("incomplete handlers: %+v", handlers)
	}
}

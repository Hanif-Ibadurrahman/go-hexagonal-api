package test

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/pkg/errors"
	"github.com/sergicanet9/go-mongo-restapi/api"
	"github.com/sergicanet9/go-mongo-restapi/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	contentType        = "application/json"
	mongoInternalPort  = "27017/tcp"
	mongoDBName        = "test-db"
	mongoConnectionEnv = "mongoConnection"
	jwtSecret          = "eaeBbXUxks"
	nonExpiryToken     = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdXRob3JpemVkIjp0cnVlfQ.sNqMUoCjbo995YsmwCXzxZ3EVF4SoHRZp8w6lhjx2GM"
)

// TestMain does the setup before running the tests and the teardown afterwards
func TestMain(m *testing.M) {
	// Uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("could not connect to docker: %s", err)
	}

	// Pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mongo",
		Tag:        "3.0",
		Env: []string{
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Fatalf("could not start resource: %s", err)
	}
	connectionString := fmt.Sprintf("mongodb://localhost:%s", resource.GetPort(mongoInternalPort))
	os.Setenv(mongoConnectionEnv, connectionString)

	// Exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		var err error
		client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(connectionString))
		if err != nil {
			return err
		}

		return client.Ping(context.Background(), nil)
	}); err != nil {
		log.Fatalf("could not connect to docker: %s", err)
	}

	// Runs the tests
	code := m.Run()

	// When it´s done, kill and remove the container
	if err = pool.Purge(resource); err != nil {
		log.Fatalf("could not purge resource: %s", err)
	}

	os.Unsetenv(mongoConnectionEnv)
	os.Exit(code)
}

// New starts a testing instance of the API and returns its config
func New(t *testing.T) config.Config {
	t.Helper()

	c, err := testConfig()
	if err != nil {
		t.Fatal(err)
	}

	a := api.API{}
	a.Initialize(c)
	go func() {
		a.Run()
	}()

	return c
}

func testConfig() (c config.Config, err error) {
	c.Env = "Integration tests"

	port, err := freePort()
	if err != nil {
		return c, err
	}
	c.Port = port
	c.Address = "http://localhost"
	c.DBConnectionString = os.Getenv(mongoConnectionEnv)
	c.DBName = mongoDBName
	c.JWTSecret = jwtSecret

	return c, nil
}

func freePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, errors.WithStack(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

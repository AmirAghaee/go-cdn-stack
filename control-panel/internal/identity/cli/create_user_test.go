package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/identity"
)

type registrarFake struct {
	email    string
	password string
	err      error
}

func (f *registrarFake) Register(_ context.Context, email, password string) error {
	f.email = email
	f.password = password
	return f.err
}

func newTestCommand(registrar Registrar) (*CreateUserCommand, *bytes.Buffer) {
	output := &bytes.Buffer{}
	return &CreateUserCommand{
		registrar: registrar,
		stdin:     strings.NewReader("secret\n"),
		stdout:    output,
		stderr:    &bytes.Buffer{},
		timeout:   time.Second,
	}, output
}

func TestCreateUserFromStdin(t *testing.T) {
	registrar := &registrarFake{}
	command, output := newTestCommand(registrar)

	err := command.Run(context.Background(), []string{
		"--email", "admin@example.com",
		"--password-stdin",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if registrar.email != "admin@example.com" || registrar.password != "secret" {
		t.Fatalf("Register() received email %q and password %q", registrar.email, registrar.password)
	}
	if !strings.Contains(output.String(), "admin@example.com") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestCreateUserRequiresValidEmail(t *testing.T) {
	command, _ := newTestCommand(&registrarFake{})

	err := command.Run(context.Background(), []string{"--email", "not-an-email", "--password-stdin"})
	if err == nil || !strings.Contains(err.Error(), "valid --email") {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestCreateUserConfirmsInteractivePassword(t *testing.T) {
	command, _ := newTestCommand(&registrarFake{})
	responses := []string{"secret", "different"}
	command.readSecret = func(string) (string, error) {
		response := responses[0]
		responses = responses[1:]
		return response, nil
	}

	err := command.Run(context.Background(), []string{"--email", "admin@example.com"})
	if err == nil || !strings.Contains(err.Error(), "passwords do not match") {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestCreateUserReportsExistingUser(t *testing.T) {
	command, _ := newTestCommand(&registrarFake{err: identity.ErrUserExists})

	err := command.Run(context.Background(), []string{
		"--email", "admin@example.com",
		"--password-stdin",
	})
	if !errors.Is(err, identity.ErrUserExists) {
		t.Fatalf("Run() error = %v, want %v", err, identity.ErrUserExists)
	}
}

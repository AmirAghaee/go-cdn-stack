package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/identity"
	"golang.org/x/term"
)

type Registrar interface {
	Register(ctx context.Context, email, password string) error
}

type CreateUserCommand struct {
	registrar  Registrar
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
	timeout    time.Duration
	readSecret func(string) (string, error)
}

func NewCreateUserCommand(registrar Registrar) *CreateUserCommand {
	command := &CreateUserCommand{
		registrar: registrar,
		stdin:     os.Stdin,
		stdout:    os.Stdout,
		stderr:    os.Stderr,
		timeout:   10 * time.Second,
	}
	command.readSecret = func(prompt string) (string, error) {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return "", errors.New("stdin is not a terminal; use --password-stdin")
		}
		if _, err := fmt.Fprint(command.stderr, prompt); err != nil {
			return "", fmt.Errorf("write password prompt: %w", err)
		}
		value, err := term.ReadPassword(int(os.Stdin.Fd()))
		if _, writeErr := fmt.Fprintln(command.stderr); err == nil && writeErr != nil {
			err = writeErr
		}
		if err != nil {
			return "", fmt.Errorf("read password: %w", err)
		}
		return string(value), nil
	}
	return command
}

func (c *CreateUserCommand) Run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("create-user", flag.ContinueOnError)
	flags.SetOutput(c.stderr)
	email := flags.String("email", "", "email address for the new user")
	passwordStdin := flags.Bool("password-stdin", false, "read the password from standard input")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	if err := validateEmail(*email); err != nil {
		return err
	}

	password, err := c.password(*passwordStdin)
	if err != nil {
		return err
	}
	if password == "" {
		return errors.New("password must not be empty")
	}

	operationCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	if err := c.registrar.Register(operationCtx, *email, password); err != nil {
		if errors.Is(err, identity.ErrUserExists) {
			return fmt.Errorf("create user: %w", identity.ErrUserExists)
		}
		return fmt.Errorf("create user: %w", err)
	}
	if _, err := fmt.Fprintf(c.stdout, "user %s created successfully\n", *email); err != nil {
		return fmt.Errorf("write success message: %w", err)
	}
	return nil
}

func (c *CreateUserCommand) password(fromStdin bool) (string, error) {
	if fromStdin {
		password, err := bufio.NewReader(c.stdin).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("read password from stdin: %w", err)
		}
		return strings.TrimRight(password, "\r\n"), nil
	}

	password, err := c.readSecret("Password: ")
	if err != nil {
		return "", err
	}
	confirmation, err := c.readSecret("Confirm password: ")
	if err != nil {
		return "", err
	}
	if password != confirmation {
		return "", errors.New("passwords do not match")
	}
	return password, nil
}

func validateEmail(value string) error {
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value {
		return errors.New("a valid --email is required")
	}
	return nil
}

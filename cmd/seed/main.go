// Command seed creates a user directly in the database, for deployments where
// public registration is disabled.
//
//	SEED_PASSWORD='...' go run ./cmd/seed -email alice@example.com -name Alice
//
// Without SEED_PASSWORD the password is read from the first line of stdin.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"shortener/internal/config"
	"shortener/internal/platform"
	"shortener/internal/user"
)

func main() {
	email := flag.String("email", "", "email address (the login ID)")
	name := flag.String("name", "", "display name")
	flag.Parse()

	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		fmt.Fprint(os.Stderr, "Password: ")
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			fail("could not read password from stdin: %v", err)
		}
		password = strings.TrimRight(line, "\r\n")
	}

	cfg, err := config.LoadPostgres()
	if err != nil {
		fail("%v", err)
	}
	db, err := platform.OpenPostgresCurrent(cfg)
	if err != nil {
		fail("%v", err)
	}

	users := user.NewService(user.NewGormRepository(db))
	profile, err := users.Register(context.Background(), user.RegisterInput{
		Email:    *email,
		Name:     *name,
		Password: password,
	})
	var fields user.ValidationErrors
	if errors.As(err, &fields) {
		for field, msg := range fields {
			fmt.Fprintf(os.Stderr, "  %s %s\n", field, msg)
		}
		fail("invalid input")
	}
	if err != nil {
		fail("%v", err)
	}
	fmt.Printf("Created user %d (%s)\n", profile.ID, profile.Email)
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "seed: "+format+"\n", args...)
	os.Exit(1)
}

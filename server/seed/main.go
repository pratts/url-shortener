// Command seed creates a user directly in the database, for deployments where
// public registration is disabled.
//
//	SEED_PASSWORD='...' go run ./server/seed -email alice@example.com -name Alice
//
// Without SEED_PASSWORD the password is read from the first line of stdin.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"shortener/configs"
	"shortener/db"
	"shortener/models"
	"shortener/users"
)

func main() {
	email := flag.String("email", "", "email address")
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

	configs.InitConfig()
	db.InitDb()

	user, err := users.CreateUser(models.UserCreateDto{
		Email:    *email,
		Name:     *name,
		Password: password,
	})
	var validationErrs users.ValidationErrors
	if errors.As(err, &validationErrs) {
		for field, msg := range validationErrs {
			fmt.Fprintf(os.Stderr, "  %s %s\n", field, msg)
		}
		fail("invalid input")
	}
	if err != nil {
		fail("%v", err)
	}
	fmt.Printf("Created user %d (%s)\n", user.Id, user.Email)
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "seed: "+format+"\n", args...)
	os.Exit(1)
}

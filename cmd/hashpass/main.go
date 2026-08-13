// Command hashpass prints a bcrypt hash of a password for use as
// ADMIN_PASSWORD_HASH. The app never stores plaintext passwords, so setup
// needs a one-time hash instead.
//
// Usage:
//
//	go run ./cmd/hashpass 'your-password'          # print a hash and exit
//	echo 'your-password' | go run ./cmd/hashpass   # read from stdin, keeps the
//	                                                # password out of shell history
//	go run ./cmd/hashpass -cost 12 'your-password' # pick a cost (default 10)
//
// Paste the printed $2a$... string into .env as ADMIN_PASSWORD_HASH, next to
// ADMIN_EMAIL. Wrap it in single quotes (ADMIN_PASSWORD_HASH='$2a$...'):
// unquoted, the $ characters are read as variable references and stripped,
// which silently breaks every login. Any bcrypt tool works, but this one
// matches the app's Go bcrypt exactly.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	cost := flag.Int("cost", 10, "bcrypt cost factor (default 10)")
	flag.Parse()

	password := strings.Join(flag.Args(), " ")
	if password == "" {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			fmt.Fprintln(os.Stderr, "hashpass: no password given")
			os.Exit(2)
		}
		password = line
	}
	password = strings.TrimRight(password, "\r\n")

	hash, err := bcrypt.GenerateFromPassword([]byte(password), *cost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hashpass:", err)
		os.Exit(1)
	}
	fmt.Println(string(hash))
}

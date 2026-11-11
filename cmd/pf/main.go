// Command pf validates fight scripts, simulates rounds and runs the balance harness.
package main

import "os"

// version is stamped at link time (-ldflags "-X main.version=v1.2.3");
// plain go build leaves it as dev.
var version = "dev"

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

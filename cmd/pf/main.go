// Command pf validates fight scripts, simulates rounds and runs the balance harness.
package main

import "os"

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

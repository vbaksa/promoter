package main

import (
	"fmt"
	"github.com/vbaksa/promoter/cmd"
	"os"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Println("No command provided. User promoter --help to get list of available commands")
	os.Exit(1)
}

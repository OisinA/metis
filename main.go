package main

import "metis/cmd"

func main() {
	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}

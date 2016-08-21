package main

import (
	"flag"
	"fmt"
	"os"
	"github.com/romana/core/pkg/util/iptsave"
)

func main() {
	flag.Parse()

	ipt := iptsave.IPtables{}
	ipt.Parse(os.Stdin)

	fmt.Println(ipt.Render())
}

package main

import (
	"context"
	"encoding/xml"
	"flag"
	"io/ioutil"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/romana/core/ipset"
)

func main() {
	flagFile := flag.String("file", "", "")
	flagCmd := flag.String("cmd", "add", "")
	flagSet1 := flag.String("set1", "", "")
	flagSet2 := flag.String("set2", "", "")
	flagType := flag.String("type", "", "")
	flagMember := flag.String("member", "", "")
	flag.Parse()

	if *flagFile != "" {
		data, err := ioutil.ReadFile(*flagFile)
		if err != nil {
			panic(err)
		}

		var sets ipset.Ipset
		err = xml.Unmarshal(data, &sets)
		if err != nil {
			panic(err)
		}

		spew.Dump(sets)

		os.Exit(0)
	}

	sets, err := ipset.Load()
	if err != nil {
		panic(err)
	}

	spew.Dump(sets)

	handle, err := ipset.New()
	if err != nil {
		panic(err)
	}

	err = handle.Start()
	if err != nil {
		panic(err)
	}

	switch *flagCmd {
	case "add":
		err = handle.Add(&ipset.Set{Name: *flagSet1, Members: []ipset.Member{ipset.Member{Elem: *flagMember}}})
		if err != nil {
			panic(err)
		}

	case "del":
		err = handle.Delete(&ipset.Set{Name: *flagSet1, Members: []ipset.Member{ipset.Member{Elem: *flagMember}}})
		if err != nil {
			panic(err)
		}
	case "create":
		err = handle.Create(&ipset.Set{Name: *flagSet1, Type: *flagType})
		if err != nil {
			panic(err)
		}
	case "destroy":
		err = handle.Destroy(&ipset.Set{Name: *flagSet1})
		if err != nil {
			panic(err)
		}
	case "flush":
		err = handle.Flush(&ipset.Set{Name: *flagSet1})
		if err != nil {
			panic(err)
		}
	case "swap":
		set1 := sets.SetByName(*flagSet1)
		set2 := sets.SetByName(*flagSet2)

		err = handle.Swap(set1, set2)
		if err != nil {
			panic(err)
		}
	default:
		panic("Unknown command")
	}

	err = handle.Quit()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = handle.Wait(ctx)
	if err != nil {
		panic(err)
	}

	sets, err = ipset.Load()
	if err != nil {
		panic(err)
	}

	spew.Dump(sets)

	return
}

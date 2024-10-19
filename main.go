package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
)

type Log struct {
	URL      url.URL       `json:"url"`
	Request  http.Request  `json:"request,omitempty"`
	Response http.Response `json:"response,omitempty"`
}

const defaultProxyListenAddr = "0.0.0.0:21001"

var (
	// Server configuration
	fromToAddresses arrayFlags = getFromEnvStringSlice("PROXY_ADDR", []string{})

	// Other
	impostersDir = getFromEnvString("IMPOSTERS_DIR", "")
)

func init() {
	flag.Var(&fromToAddresses, "proxy", "Multiple values, proxy listen address and target address (if empty, env:PROXY_ADDR will be used).")
	flag.StringVar(&impostersDir, "imposters-dir", impostersDir, "Directory to store imposters.")
	flag.Parse()

	if impostersDir == "" {
		fmt.Println("imposters-dir is required")
		os.Exit(1)
	}
}

func main() {
	addresses := prepareFromToAddresses(fromToAddresses)
	if len(addresses) == 0 {
		fmt.Println("No proxy addresses provided")
		os.Exit(1)
	}

	logChan := make(chan Log, 10)

	wg := sync.WaitGroup{}
	wg.Add(1 + len(fromToAddresses))

	go func() {
		defer wg.Done()
		createImposters(impostersDir, logChan)
	}()

	for _, fromToAddr := range addresses {
		from, to := fromToAddr[0], fromToAddr[1]

		go func() {
			defer wg.Done()

			fmt.Printf("Proxy server listening on http://%s, forwarding to %s\n", from, to)
			err := http.ListenAndServe(from, NewProxyHandler(parseUrl(to), logChan))
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}()
	}

	wg.Wait()
}

func prepareFromToAddresses(fromToAddresses []string) [][]string {
	fromTos := make([][]string, 0)
	usedAddrs := make([]string, 0)

	for _, fromToAddr := range fromToAddresses {
		split := strings.Split(fromToAddr, "::")
		if len(split) > 2 || len(split) == 0 {
			fmt.Println("Invalid proxy address:", fromToAddr)
			os.Exit(1)
		}

		if len(split) == 1 {
			split = []string{defaultProxyListenAddr, split[0]}
		}

		if slices.Contains(usedAddrs, split[0]) {
			fmt.Println("Duplicate proxy listen address:", split[0])
			os.Exit(1)
		}

		if split[1] == "" {
			fmt.Println("Target address is empty")
			os.Exit(1)
		}

		fromTos = append(fromTos, split)
		usedAddrs = append(usedAddrs, split[0])
	}

	return fromTos
}

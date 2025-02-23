package main

import (
	"context"
	"flag"
	"log"
	"regexp"

	azlog "github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
	"github.com/abicky/terraform-provider-azurekv/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary.
	version string = "dev"

	// goreleaser can pass other information to the main package, such as the specific commit
	// https://goreleaser.com/cookbooks/using-main.version/

	beginningOfLineRegexp = regexp.MustCompile(`(?m)^`)
)

func main() {
	var debug bool

	// Omit timestamps because logs that does not start with a log level are ignored
	// cf. https://github.com/hashicorp/go-plugin/blob/v1.6.3/client.go#L1202-L1217
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	azlog.SetListener(func(event azlog.Event, msg string) {
		log.Print(beginningOfLineRegexp.ReplaceAllLiteralString(string(event)+" "+msg, "[DEBUG] "))
	})

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/abicky/azurekv",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}

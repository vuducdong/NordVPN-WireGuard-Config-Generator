package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"nordgen/internal/client"
	"nordgen/internal/constants"
	"nordgen/internal/generator"
	"nordgen/internal/models"
	"nordgen/internal/ui"
)

var tokenPattern = regexp.MustCompile(`^(?i)[0-9a-f]{64}$`)

type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, " ")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func normalizeGroupArgs(args []string) []string {
	var normalized []string
	inGroup := false
	for _, arg := range args {
		if arg == "-g" || arg == "--group" {
			inGroup = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			inGroup = false
			normalized = append(normalized, arg)
			continue
		}
		if inGroup {
			normalized = append(normalized, "-g", arg)
		} else {
			normalized = append(normalized, arg)
		}
	}
	return normalized
}

func printHelp() {
	fmt.Fprint(os.Stdout, `USAGE:
  nordgen [options]
  nordgen get-key [options]

COMMANDS:
  (default)   Generate WireGuard configurations
  get-key     Extract NordLynx private key from token
  help        Show this help message

GENERATE OPTIONS:
  -t, --token              NordVPN Access Token (Prompts interactively if omitted)
  -d, --dns                DNS Server IP (Default: 103.86.96.100)
  -i, --ip                 Use IP addresses instead of hostnames for endpoints
  -k, --keepalive          PersistentKeepalive in seconds (Default: 25)
  -e, --exclude-dedicated  Exclude servers in the dedicated IP group
  -g, --group              Server groups to include (Supports space-separated lists)
                           Valid groups: standard, p2p, dedicated, onion, double

GET-KEY OPTIONS:
  -t, --token              NordVPN Access Token

EXAMPLES:
  nordgen -t <your-token>
  nordgen -d 1.1.1.1 -k 15 -g standard p2p
  nordgen get-key -t <your-token>

`)
	os.Exit(0)
}

func resolvePrivateKey(consoleManager *ui.ConsoleManager, nordClient *client.NordClient, token string) string {
	if token == "" {
		token = consoleManager.PromptSecret("NordVPN access token")
	}
	if !tokenPattern.MatchString(token) {
		consoleManager.Fail("Invalid token format")
		return ""
	}
	consoleManager.StartStatus("Validating token...")
	key, err := nordClient.GetKey(token)
	consoleManager.StopStatus()

	if err != nil || key == "" {
		consoleManager.Fail("Token invalid")
		return ""
	}
	consoleManager.Success("Token validated")
	return key
}

func runGetKey(consoleManager *ui.ConsoleManager, nordClient *client.NordClient, token string) {
	consoleManager.Header()
	key := resolvePrivateKey(consoleManager, nordClient, token)
	if key != "" {
		consoleManager.ShowKey(key)
	}
	if token == "" {
		consoleManager.Wait()
	}
}

func runGenerate(consoleManager *ui.ConsoleManager, nordClient *client.NordClient, token string, prefs models.UserPreferences) {
	isInteractive := token == ""

	consoleManager.Header()
	key := resolvePrivateKey(consoleManager, nordClient, token)
	if key == "" {
		if isInteractive {
			consoleManager.Wait()
		}
		return
	}

	if isInteractive {
		consoleManager.ClearScreen()
		prefs = consoleManager.PromptPreferences(prefs)
		consoleManager.ClearScreen()
	}

	if prefs.Keepalive < 0 {
		consoleManager.Fail("Keepalive value must be greater than or equal to 0")
		if isInteractive {
			consoleManager.Wait()
		}
		return
	}

	if prefs.ExcludeDedicated {
		for _, g := range prefs.Groups {
			if g == constants.AliasToGroupID["dedicated"] {
				consoleManager.Fail("Conflict: Cannot require 'dedicated' group while using exclude-dedicated option")
				if isInteractive {
					consoleManager.Wait()
				}
				return
			}
		}
	}

	gen := generator.NewGenerator(nordClient, consoleManager)

	startedAt := time.Now()
	outPath, err := gen.Process(key, prefs)
	if err != nil {
		if isInteractive {
			consoleManager.Wait()
		}
		return
	}

	consoleManager.ClearScreen()
	consoleManager.Summary(outPath, gen.Stats, time.Since(startedAt).Seconds())
	if isInteractive {
		consoleManager.Wait()
	}
}

func main() {
	consoleManager := ui.NewConsoleManager()
	nordClient := client.NewNordClient()

	var cmd string
	var parseArgs []string

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "get-key":
			cmd = "get-key"
			parseArgs = os.Args[2:]
		case "generate":
			cmd = "generate"
			parseArgs = os.Args[2:]
		default:
			cmd = "generate"
			parseArgs = os.Args[1:]
		}
	} else {
		cmd = "generate"
		parseArgs = []string{}
	}

	if cmd == "help" {
		printHelp()
	}
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			printHelp()
		}
	}

	genCmd := flag.NewFlagSet("generate", flag.ExitOnError)
	genCmd.Usage = func() { printHelp() }

	var genToken string
	genCmd.StringVar(&genToken, "t", "", "NordVPN Access Token")
	genCmd.StringVar(&genToken, "token", "", "NordVPN Access Token")

	var genDNS string
	genCmd.StringVar(&genDNS, "d", "103.86.96.100", "DNS Server")
	genCmd.StringVar(&genDNS, "dns", "103.86.96.100", "DNS Server")

	var genIP bool
	genCmd.BoolVar(&genIP, "i", false, "Use IP Endpoint")
	genCmd.BoolVar(&genIP, "ip", false, "Use IP Endpoint")

	var genKeepalive int
	genCmd.IntVar(&genKeepalive, "k", 25, "Keepalive seconds")
	genCmd.IntVar(&genKeepalive, "keepalive", 25, "Keepalive seconds")

	var genExclude bool
	genCmd.BoolVar(&genExclude, "e", false, "Exclude servers in the dedicated IP group")
	genCmd.BoolVar(&genExclude, "exclude-dedicated", false, "Exclude servers in the dedicated IP group")

	var genGroups stringSlice
	genCmd.Var(&genGroups, "g", "Server groups to include")
	genCmd.Var(&genGroups, "group", "Server groups to include")

	keyCmd := flag.NewFlagSet("get-key", flag.ExitOnError)
	keyCmd.Usage = func() { printHelp() }

	var keyToken string
	keyCmd.StringVar(&keyToken, "t", "", "NordVPN Access Token")
	keyCmd.StringVar(&keyToken, "token", "", "NordVPN Access Token")

	switch cmd {
	case "get-key":
		keyCmd.Parse(parseArgs)
		runGetKey(consoleManager, nordClient, keyToken)
	case "generate":
		normalizedArgs := normalizeGroupArgs(parseArgs)
		genCmd.Parse(normalizedArgs)

		var internalGroups []string
		for _, g := range genGroups {
			if alias, exists := constants.AliasToGroupID[g]; exists {
				internalGroups = append(internalGroups, alias)
			}
		}

		prefs := models.UserPreferences{
			DNS:              genDNS,
			UseIP:            genIP,
			Keepalive:        genKeepalive,
			Groups:           internalGroups,
			ExcludeDedicated: genExclude,
		}
		runGenerate(consoleManager, nordClient, genToken, prefs)
	default:
		consoleManager.Fail("Unknown command: " + cmd)
		printHelp()
	}
}

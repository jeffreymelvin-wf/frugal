package crossrunner

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"
)

const (
	// Default timeout in seconds for client/server configurations without a defined timeout
	DefaultTimeout     = 7
	TestFailure        = 101
	CrossrunnerFailure = 102
)

// client/server level options defined in tests.json
// this is useful if there is option supported by a client but not a server within a language.
type options struct {
	Command    []string
	Transports []string
	Protocols  []string
	Timeout    time.Duration
}

// language level options defined in tests.json.
type languages struct { // Example
	Name       string   // Language name
	Client     options  // client specific commands, protocols, transports, and timesouts
	Server     options  // server specific commands, protocols, transports, and timesouts
	Transports []string // transports that apply to both clients and servers within a language
	Protocols  []string // protocols that apply to both clients and servers within a language
	Command    []string // command that applies to both clients and servers within a language
	Workdir    string   // working directory relative to /test/integration
}

//  Complete information required to shell out a client or server command.
type config struct {
	Name      string
	Timeout   time.Duration
	Transport string
	Protocol  string
	Command   []string
	Workdir   string
	Logs      *os.File
}

// Matched client and server commands.
type Pair struct {
	Client     config
	Server     config
	ReturnCode int
	Err        error
}

func newPair(client, server config) *Pair {
	return &Pair{
		Client: client,
		Server: server,
	}
}

// Load takes a json file of client/server definitions and returns a list of
// valid client/server pairs.
func Load(jsonFile string) (pairs []*Pair, err error) {
	bytes, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		return nil, err
	}

	var tests []languages

	// Unmarshal json into defined structs
	if err := json.Unmarshal(bytes, &tests); err != nil {
		return nil, err
	}

	// Create empty lists of client and server configurations
	var clients []config
	var servers []config

	// Iterate over each language to get all client/server configurations in that language
	for _, test := range tests {

		// Append language level transports and protocols to client/server level
		test.Client.Transports = append(test.Client.Transports, test.Transports...)
		test.Server.Transports = append(test.Server.Transports, test.Transports...)
		test.Client.Protocols = append(test.Client.Protocols, test.Protocols...)
		test.Server.Protocols = append(test.Server.Protocols, test.Protocols...)

		// Get expanded list of clients/servers, using both language and config level options
		clients = append(clients, getExpandedConfigs(test.Client, test)...)
		servers = append(servers, getExpandedConfigs(test.Server, test)...)
	}

	// Find all valid client/server pairs
	// TODO: Accept some sort of flag(s) that would limit this list of pairs by
	// desired language(s) or other restrictions
	for _, client := range clients {
		for _, server := range servers {
			if server.Transport == client.Transport && server.Protocol == client.Protocol {
				pairs = append(pairs, newPair(client, server))
			}
		}
	}

	return pairs, nil
}

// getExpandedConfigs takes a client/server at the language level and the options
// associated with that client/server and returns a list of unique configs.
func getExpandedConfigs(options options, test languages) (apps []config) {
	app := new(config)

	// Loop through each transport and protocol to construct expanded list
	for _, transport := range options.Transports {
		for _, protocol := range options.Protocols {
			app.Name = test.Name
			app.Protocol = protocol
			app.Transport = transport
			app.Command = append(test.Command, options.Command...)
			app.Workdir = test.Workdir
			app.Timeout = DefaultTimeout * time.Second
			if options.Timeout != 0 {
				app.Timeout = options.Timeout
			}
			apps = append(apps, *app)
		}
	}
	return apps
}
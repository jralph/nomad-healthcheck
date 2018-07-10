package main

import (
	"encoding/json"
	consul "github.com/hashicorp/consul/api"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/pborman/getopt/v2"
	"log"
	"net/http"
	"os"
	"time"
)

// Configuration settings.
type Configuration struct {
	ListenAddr   string
	PollInterval int
	NomadHost string
	ConsulHost   string
}

// Global variable to determine if the load-balancer is healthy or not.
var healthy bool

func main() {
	configFile := getopt.String('c', "./nomad-healthcheck.json", "The path to the nomad-healthcheck config file.", "string")

	opts := getopt.CommandLine
	opts.Parse(os.Args)

	log.Print("Starting Nomad Healthcheck...")
	log.Printf("Using config file \"%s\"", *configFile)

	config := newConfig(*configFile)

	go pollHealth(config)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !healthy {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	})

	log.Printf("HTTP server listening on: %s", config.ListenAddr)
	log.Fatal(http.ListenAndServe(config.ListenAddr, nil))

	log.Println("Fnished.")
}

// Create a new configuration setup.
func newConfig(path string) Configuration {
	config := Configuration{
		ListenAddr:   "0.0.0.0:10700",
		PollInterval: 10,
		NomadHost:    "localhost:4646",
		ConsulHost:   "localhost:8500",
	}

	if _, err := os.Stat(path); err == nil {
		file, _ := os.Open(path)
		defer file.Close()

		decoder := json.NewDecoder(file)
		err := decoder.Decode(&config)

		if err != nil {
			log.Fatal("Unable to read config file. Check json is correct.", err)
		}
	}

	return config
}

// Check that consul is healthy.
func consulIsHealthy(consulAddress string) bool {
	config := consul.Config{
		Address: consulAddress,
	}

	client, err := consul.NewClient(&config)

	if err != nil {
		log.Print("Error connecting to consul client.", err)
		return false
	}

	status := client.Status()
	leader, err := status.Leader()

	if err != nil {
		log.Print("Error querying consul leader.", err)
		return false
	}

	if leader != "" {
		return true
	}

	return false
}

// Check nomad is healthy.
func nomadIsHealthy(nomadAddress string) bool {
	config := nomad.Config{
		Address: nomadAddress,
	}

	client, err := nomad.NewClient(&config)

	if err != nil {
		log.Print("Error connecting to nomad client.", err)
		return false
	}

	status := client.Status()
	leader, err := status.Leader()

	if err != nil {
		log.Print("Error querying nomad leader.", err)
		return false
	}

	if leader == "" {
		return false
	}

	members, err := client.Agent().Members()

	if err != nil {
		log.Print("Error getting nomad agent members.", err)
		return false
	}

	if len(members.Members) < 2 {
		log.Print("Found 1 or less nomad agent members.")
		return false
	}

	nodes, _, err := client.Nodes().List(&nomad.QueryOptions{})

	if err != nil {
		log.Print("Error getting nomad nodes.", err)
		return false
	}

	if len(nodes) < 2 {
		log.Print("Found 1 or less nomad nodes.")
		return false
	}

	jobs, _, err := client.Jobs().List(&nomad.QueryOptions{})

	if err != nil {
		log.Print("Error getting nomad jobs.", err)
	}

	if len(jobs) < 1 {
		log.Print("Found no jobs.")
		return false
	}

	return true
}

// Check the overall nomad agent is healthy.
func isHealthy(config Configuration) bool {
	return consulIsHealthy(config.ConsulHost) && nomadIsHealthy(config.NomadHost)
}

// Poll for health changes and save to the global healthy variable.
func pollHealth(config Configuration) {
	healthy = isHealthy(config)
	time.Sleep(time.Second * time.Duration(config.PollInterval))
	pollHealth(config)
}

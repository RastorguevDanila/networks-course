package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
)

const MAX_METRIC = 16
const CONFIG_FILE_NAME = "config.json"

type RouteEntry struct {
	DestinationIP string
	NextHopIP     string
	Metric        int
}

type Router struct {
	IP           string
	RoutingTable map[string]RouteEntry
	NeighborIPs  []string
}

type Config struct {
	Routers []string `json:"routers"`
	Links   []struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"links"`
}


func printTable(router *Router, title string) {
	fmt.Printf("\n%s\n", title)
	fmt.Println("[Source IP]      [Destination IP]    [Next Hop]       [Metric]")

	var destinations []string
	for dest := range router.RoutingTable {
		destinations = append(destinations, dest)
	}
	sort.Strings(destinations) 

	for _, destIP := range destinations {
		entry := router.RoutingTable[destIP]
		nextHopDisplay := entry.NextHopIP
		if entry.Metric == 0 {
			nextHopDisplay = router.IP
		}
		fmt.Printf("%-15s    %-15s     %-15s     %d\n",
			router.IP,
			entry.DestinationIP,
			nextHopDisplay,
			entry.Metric)
	}
}

func initializeRouterTable(r *Router, allRouters map[string]*Router) {
	r.RoutingTable = make(map[string]RouteEntry)
	r.RoutingTable[r.IP] = RouteEntry{DestinationIP: r.IP, NextHopIP: r.IP, Metric: 0}
	for _, neighborIP := range r.NeighborIPs {
		if _, exists := allRouters[neighborIP]; exists {
			r.RoutingTable[neighborIP] = RouteEntry{DestinationIP: neighborIP, NextHopIP: neighborIP, Metric: 1}
		}
	}
}

func uniqueStrings(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func runSimulationSync(allRouters map[string]*Router) {
	for _, r := range allRouters {
		initializeRouterTable(r, allRouters)
	}

	fmt.Println("Initial state of tables (Step 0):")
	for _, r := range allRouters {
		printTable(r, fmt.Sprintf("Router %s table:", r.IP))
	}

	maxSteps := len(allRouters) * 2
	if maxSteps == 0 {
		maxSteps = 1
	} else if maxSteps < 5 && len(allRouters) > 1 {
		maxSteps = 5
	}

	for step := 0; step < maxSteps; step++ {
		changedInThisIteration := false
		advertisementsToSend := make(map[string][]RouteEntry)

		for _, sender := range allRouters {
			currentAd := make([]RouteEntry, 0, len(sender.RoutingTable))
			for _, entry := range sender.RoutingTable {
				currentAd = append(currentAd, entry)
			}
			advertisementsToSend[sender.IP] = currentAd
		}

		for _, receiver := range allRouters {
			for _, neighborIP := range receiver.NeighborIPs {
				if routesFromNeighbor, ok := advertisementsToSend[neighborIP]; ok {
					for _, advEntry := range routesFromNeighbor {

						if advEntry.DestinationIP == receiver.IP { // маршрут к самому себе
							continue
						}

						costViaNeighbor := advEntry.Metric + 1
						if costViaNeighbor > MAX_METRIC {
							costViaNeighbor = MAX_METRIC
						}

						currentEntry, exists := receiver.RoutingTable[advEntry.DestinationIP]

						if !exists && costViaNeighbor < MAX_METRIC { // если маршрута нет и новая стоимость нормальная - создаём
							receiver.RoutingTable[advEntry.DestinationIP] = RouteEntry{
								DestinationIP: advEntry.DestinationIP,
								NextHopIP:     neighborIP,
								Metric:        costViaNeighbor,
							}
							changedInThisIteration = true
						} else if exists { 
							if neighborIP == currentEntry.NextHopIP { // маршрут есть но он проходит через того же соседа
								if costViaNeighbor != currentEntry.Metric {
									receiver.RoutingTable[advEntry.DestinationIP] = RouteEntry{
										DestinationIP: advEntry.DestinationIP,
										NextHopIP:     neighborIP,
										Metric:        costViaNeighbor,
									}
									changedInThisIteration = true
								}
							} else { // через другого соседа
								if costViaNeighbor < currentEntry.Metric {
									receiver.RoutingTable[advEntry.DestinationIP] = RouteEntry{
										DestinationIP: advEntry.DestinationIP,
										NextHopIP:     neighborIP,
										Metric:        costViaNeighbor,
									}
									changedInThisIteration = true
								}
							}
						}
					}
				}
			}
		}

		fmt.Printf("\n--- Simulation step %d ---\n", step+1)
		for _, r := range allRouters {
			printTable(r, fmt.Sprintf("Router %s table:", r.IP))
		}

		if !changedInThisIteration && step > 0 {
			fmt.Printf("\n--- Convergence reached after %d iteration(s) ---\n", step+1)
			break
		}
		if step == maxSteps-1 {
			fmt.Printf("\n--- Max iterations %d reached ---\n", maxSteps)
		}
	}

	fmt.Println("\n\n--- Final Routing Tables ---")
	for _, r := range allRouters {
		printTable(r, fmt.Sprintf("Final state of router %s table:", r.IP))
	}
}

func loadConfigFromFile(filePath string) (map[string]*Router, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", filePath, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON from '%s': %w", filePath, err)
	}

	allRouters := make(map[string]*Router)
	for _, ipStr := range config.Routers {
		allRouters[ipStr] = &Router{IP: ipStr, NeighborIPs: []string{}}
	}

	for _, link := range config.Links {
		r1, ok1 := allRouters[link.From]
		r2, ok2 := allRouters[link.To]
		if ok1 && ok2 {
			r1.NeighborIPs = append(r1.NeighborIPs, link.To)
			r2.NeighborIPs = append(r2.NeighborIPs, link.From)
		} else {
			log.Printf("Warning: Invalid link %s-%s", link.From, link.To)
		}
	}
	for _, r := range allRouters {
		r.NeighborIPs = uniqueStrings(r.NeighborIPs)
	}
	return allRouters, nil
}

func main() {
	allRouters, err := loadConfigFromFile(CONFIG_FILE_NAME)
	if err != nil {
		log.Fatalf("Fatal error: Could not load or parse '%s': %v", CONFIG_FILE_NAME, err)
		os.Exit(1) 
	}
	runSimulationSync(allRouters)
}
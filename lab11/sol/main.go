package main

import (
	"fmt"
	"math"
	"sort"
)

const INFINITY = math.MaxInt32

type RouteInfo struct {
	Cost    int
	NextHop int
}

type Node struct {
	ID             int
	NetworkSize    int
	DistanceVector map[int]RouteInfo
	Neighbors      map[int]int
	AdvertisedDV   map[int]RouteInfo
}

func NewNode(id int, networkSize int) *Node {
	dv := make(map[int]RouteInfo)
	for i := 0; i < networkSize; i++ {
		dv[i] = RouteInfo{Cost: INFINITY, NextHop: -1}
	}
	dv[id] = RouteInfo{Cost: 0, NextHop: id}

	return &Node{
		ID:             id,
		NetworkSize:    networkSize,
		DistanceVector: dv,
		Neighbors:      make(map[int]int),
	}
}

func (n *Node) AddNeighbor(neighborID int, cost int) {
	n.Neighbors[neighborID] = cost
	n.DistanceVector[neighborID] = RouteInfo{Cost: cost, NextHop: neighborID}
}

func (n *Node) UpdateDistanceVector(neighborID int, receivedDV map[int]RouteInfo) bool {
	changed := false
	costToNeighbor, ok := n.Neighbors[neighborID]
	if !ok {
		return false
	}

	for destNodeID := 0; destNodeID < n.NetworkSize; destNodeID++ {
		routeInfoFromNeighbor, neighborKnowsRoute := receivedDV[destNodeID]
		if !neighborKnowsRoute {
			routeInfoFromNeighbor = RouteInfo{Cost: INFINITY, NextHop: -1}
		}

		var newCostViaNeighbor int
		if routeInfoFromNeighbor.Cost == INFINITY {
			newCostViaNeighbor = INFINITY
		} else {
			if costToNeighbor > INFINITY-routeInfoFromNeighbor.Cost {
				newCostViaNeighbor = INFINITY
			} else {
				newCostViaNeighbor = costToNeighbor + routeInfoFromNeighbor.Cost
			}
		}

		if routeInfoFromNeighbor.NextHop == n.ID && destNodeID != n.ID {
			continue
		}

		currentRouteToDest := n.DistanceVector[destNodeID]

		if newCostViaNeighbor < currentRouteToDest.Cost {
			n.DistanceVector[destNodeID] = RouteInfo{Cost: newCostViaNeighbor, NextHop: neighborID}
			changed = true
		} else if currentRouteToDest.NextHop == neighborID && newCostViaNeighbor > currentRouteToDest.Cost {
			n.DistanceVector[destNodeID] = RouteInfo{Cost: newCostViaNeighbor, NextHop: neighborID}
			changed = true
		}
	}
	return changed
}

func (n *Node) PrepareAdvertisedDV() {
	n.AdvertisedDV = make(map[int]RouteInfo)
	for dest, info := range n.DistanceVector {
		n.AdvertisedDV[dest] = info
	}
}

func (n *Node) PrintRoutingTable() {
	header := fmt.Sprintf("Узел %d", n.ID)
	fmt.Printf("\n%s:\n", header)
	fmt.Println("-----------------------------------------")
	fmt.Println("| Пункт назн. | Стоимость | Следующий узел |")
	fmt.Println("-----------------------------------------")

	var destIDs []int
	for id := range n.DistanceVector {
		destIDs = append(destIDs, id)
	}
	sort.Ints(destIDs)

	for _, destID := range destIDs {
		routeInfo := n.DistanceVector[destID]
		costStr := "INF"
		if routeInfo.Cost != INFINITY {
			costStr = fmt.Sprintf("%d", routeInfo.Cost)
		}
		nextHopStr := "-"
		if routeInfo.NextHop != -1 && routeInfo.Cost != INFINITY {
			nextHopStr = fmt.Sprintf("%d", routeInfo.NextHop)
		}
		if routeInfo.Cost == 0 && routeInfo.NextHop == n.ID {
			nextHopStr = fmt.Sprintf("%d", n.ID)
		}
		fmt.Printf("|      %d         | %-9s |       %-10s |\n", destID, costStr, nextHopStr)
	}
	fmt.Println("-----------------------------------------")
}

type Link struct {
	U, V, Cost int
}

var nodes []*Node

func SetupNetwork(numNodes int, linksConfig []Link) {
	nodes = make([]*Node, numNodes)
	for i := 0; i < numNodes; i++ {
		nodes[i] = NewNode(i, numNodes)
	}

	for _, link := range linksConfig {
		nodes[link.U].AddNeighbor(link.V, link.Cost)
		nodes[link.V].AddNeighbor(link.U, link.Cost)
	}
}

func RunDistanceVectorSimulation(maxIterations int) {
	for i := 0; i < maxIterations; i++ {
		for _, node := range nodes {
			node.PrepareAdvertisedDV()
		}
		anyTableChangedInIteration := false
		for _, node := range nodes {
			nodeChangedItsTable := false
			for neighborID := range node.Neighbors {
				receivedDV := nodes[neighborID].AdvertisedDV
				if node.UpdateDistanceVector(neighborID, receivedDV) {
					nodeChangedItsTable = true
				}
			}
			if nodeChangedItsTable {
				anyTableChangedInIteration = true
			}
		}
		if !anyTableChangedInIteration {
			break
		}
		if i == maxIterations-1 {
			break;
		}
	}

	fmt.Println("\n--- Финальное состояние таблиц маршрутизации ---")
	for _, node := range nodes {
		node.PrintRoutingTable()
	}
}

func main() {
	const NUM_NODES = 4

	initialLinks := []Link{
		{0, 1, 1},
		{0, 2, 3},
		{0, 3, 7},
		{1, 2, 1},
		{2, 3, 2},
	}
	for i, link := range initialLinks { //
		if (link.U == 0 && link.V == 3) || (link.U == 3 && link.V == 0) {
			initialLinks[i].Cost = 1
			break
		}
	} //Эти 5 строк на задание B.
	SetupNetwork(NUM_NODES, initialLinks)
	RunDistanceVectorSimulation(10)
}
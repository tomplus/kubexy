package main

import (
	"fmt"
	"math/rand"

	"github.com/technosophos/moniker"
)

// RandomNameGenerator is a struct with mapping real names to the randomized ones
type RandomNameGenerator struct {
	clusterName string
	nodes       map[string]string
	namespaces  map[string]string
	pods        map[string]string
	randNames   *RandomMoniker
}

// RandomMoniker has permuations of names from Moniker
type RandomMoniker struct {
	permDesc, permNoun []int
	currDesc, currNoun int
}

// NewRandomMoniker inits RandomMoniker struct
func NewRandomMoniker() *RandomMoniker {
	return &RandomMoniker{
		permDesc: rand.Perm(len(moniker.Descriptors)),
		permNoun: rand.Perm(len(moniker.Animals)),
	}
}

// GetName returns random name (one word)
func (rm *RandomMoniker) GetName() string {
	name := moniker.Animals[rm.permNoun[rm.currNoun%len(moniker.Animals)]]
	rm.currNoun++
	return name
}

// GetLongName returns 2-word long random name
func (rm *RandomMoniker) GetLongName() string {
	name := rm.GetName()
	desc := moniker.Descriptors[rm.permDesc[rm.currDesc%len(moniker.Descriptors)]]
	rm.currDesc++
	return fmt.Sprintf("%v_%v", desc, name)
}

// NewRandomNameGenerator inits RandomNameGenerator struct
func NewRandomNameGenerator() *RandomNameGenerator {
	rng := RandomNameGenerator{}
	rng.clusterName = fmt.Sprintf("cluster-%v", 10+rand.Intn(90))
	rng.nodes = make(map[string]string)
	rng.namespaces = make(map[string]string)
	rng.pods = make(map[string]string)
	rng.randNames = NewRandomMoniker()
	return &rng
}

// GetNode returns randomized name for passed node
func (rng *RandomNameGenerator) GetNode(node string) string {
	if val, ok := rng.nodes[node]; ok {
		return val
	}
	number := len(rng.nodes)
	zone := number % 3
	name := fmt.Sprintf("%v-az%v-%x", rng.clusterName, zone, 256+10*number)
	rng.nodes[node] = name
	return name
}

// GetNamespace returns randomized name for passed namespace
func (rng *RandomNameGenerator) GetNamespace(namespace string) string {
	if val, ok := rng.namespaces[namespace]; ok {
		return val
	}
	name := rng.randNames.GetLongName()
	rng.namespaces[namespace] = name
	return name
}

// GetPodName returns randomized name for passed pods generateName
func (rng *RandomNameGenerator) GetPodName(generateName string) string {
	if val, ok := rng.pods[generateName]; ok {
		return val
	}
	name := rng.randNames.GetName()
	rng.pods[generateName] = rng.randNames.GetLongName()
	return name
}

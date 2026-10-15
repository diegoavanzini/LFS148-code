package main

import (
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

type UserInterface interface {
	GetUser(id int) (*User, int)
}

// FakerClient simulates a database.
type FakerClient struct {
	faker *gofakeit.Faker
}

func NewFakerClient() *FakerClient {
	return &FakerClient{faker: gofakeit.New(0)}
}

func (c *FakerClient) GetUser(id int) (*User, int) {
	return &User{
		ID:      rand.IntN(1000) + 1,
		Name:    c.faker.Name(),
		Address: c.faker.Address().Address,
	}, http.StatusOK
}

// ChaosClient wraps another client for application-level fault injection.
type ChaosClient struct {
	client                  UserInterface
	baseDelay               time.Duration
	requestType             []string
	requestProbability      []float64
	requestLatency          map[string]time.Duration
	responseStatus          []string
	responseCodeProbability []float64
}

func NewChaosClient(client UserInterface) *ChaosClient {
	return &ChaosClient{
		client:             client,
		baseDelay:          50 * time.Millisecond,
		requestType:        []string{"fast", "medium", "slow"},
		requestProbability: []float64{.80, .15, .05},
		requestLatency: map[string]time.Duration{
			"fast":   100 * time.Millisecond,
			"medium": 300 * time.Millisecond,
			"slow":   2000 * time.Millisecond,
		},
		responseStatus:          []string{"success", "fail"},
		responseCodeProbability: []float64{.90, .10},
	}
}

func (c *ChaosClient) GetUser(id int) (*User, int) {
	faultsEnabled := strings.ToLower(os.Getenv("CHAOS")) == "true"
	if faultsEnabled {
		additionalLatency := c.requestLatency[weightedChoice(c.requestType, c.requestProbability)]
		time.Sleep(c.baseDelay)
		time.Sleep(additionalLatency)

		switch weightedChoice(c.responseStatus, c.responseCodeProbability) {
		case "success":
			user, _ := c.client.GetUser(id)
			return user, http.StatusOK
		case "fail":
			return nil, http.StatusNotFound
		}
	}

	user, _ := c.client.GetUser(id)
	return user, http.StatusOK
}

func weightedChoice(items []string, weights []float64) string {
	var total float64
	for _, w := range weights {
		total += w
	}
	r := rand.Float64() * total
	for i, w := range weights {
		r -= w
		if r < 0 {
			return items[i]
		}
	}
	return items[len(items)-1]
}

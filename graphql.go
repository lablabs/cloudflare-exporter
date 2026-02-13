package main

import (
	"net/http"
	"sync"

	"github.com/machinebox/graphql"
)

const (
	cfGraphQLEndpoint = "https://api.cloudflare.com/client/v4/graphql/"
	gqlQueryLimit     = 9999
	cfgraphqlreqlimit = 10 // 10 is the maximum amount of zones you can request at once
)

type GraphQL struct {
	Client *graphql.Client
	Mu     *sync.RWMutex
}

func NewGraphQLClient(httpClient *http.Client) *GraphQL {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	gqlClient := graphql.NewClient(cfGraphQLEndpoint, graphql.WithHTTPClient(httpClient))
	gqlClient.Log = func(s string) { log.Debug(s) }
	return &GraphQL{
		Client: gqlClient,
		Mu:     &sync.RWMutex{},
	}
}

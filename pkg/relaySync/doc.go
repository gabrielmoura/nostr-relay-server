// Package relaysync provides the transport-agnostic RelaySync domain contracts.
//
// It deliberately does not open network listeners or depend on NRServer
// internals. Transport adapters and durable storage implementations belong to
// their consumers.
package relaysync

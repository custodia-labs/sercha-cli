// Package connectors provides implementations of the Connector interface
// for various document sources. Each connector knows how to fetch documents
// from a specific source type (filesystem, Notion, etc.).
//
// Connectors are registered with the ConnectorFactory at startup.
package connectors

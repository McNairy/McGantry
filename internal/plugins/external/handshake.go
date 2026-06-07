package external

import "github.com/hashicorp/go-plugin"

// Handshake is the shared configuration used by both Gantry (client) and
// external plugin binaries (server). A mismatching magic cookie causes
// go-plugin to return an error immediately so accidentally-invoked binaries
// fail fast rather than hanging.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  2,
	MagicCookieKey:   "GANTRY_PLUGIN",
	MagicCookieValue: "gantry-plugin-v1",
}

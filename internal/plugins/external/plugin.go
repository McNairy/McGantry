// Package external implements the Gantry external plugin system.
// External plugins are separate binaries discovered at runtime from a configured
// directory. They communicate with Gantry via net/rpc using hashicorp/go-plugin
// for subprocess lifecycle management.
package external

import (
	"encoding/gob"
	"encoding/json"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

func init() {
	// map[string]any is not registered by default; without this, ConfigSchema
	// values are silently dropped during gob encoding over net/rpc.
	gob.Register(map[string]any{})
}

// Manifest is a plugin's self-description, returned by GetManifest.
type Manifest struct {
	Name            string        `json:"name"`
	Title           string        `json:"title"`
	Description     string        `json:"description"`
	Version         string        `json:"version"`
	Author          string        `json:"author"`
	Category        string        `json:"category"`
	IconURL         string        `json:"iconUrl,omitempty"`
	Homepage        string        `json:"homepage,omitempty"`
	SupportsSync    bool          `json:"supportsSync"`
	SupportsPanels  bool          `json:"supportsPanels"`
	SupportsActions bool          `json:"supportsActions"`
	EntityPanels    []string      `json:"entityPanels,omitempty"`
	ActionTypes     []string      `json:"actionTypes,omitempty"`
	Requirements    []Requirement `json:"requirements,omitempty"`
	// ConfigSchemaJSON carries the JSON-encoded configSchema over gob.
	// map[string]any cannot cross the gob boundary safely; a JSON string can.
	ConfigSchemaJSON string `json:"configSchemaJson,omitempty"`
	SupportsHTTP     bool   `json:"supportsHttp,omitempty"`
	// HTTPRoutesJSON carries the JSON-encoded []Route over gob.
	HTTPRoutesJSON string `json:"httpRoutesJson,omitempty"`
}

// Requirement describes an infrastructure or configuration prerequisite.
type Requirement struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Optional    bool   `json:"optional,omitempty"`
}

// SyncResult is returned by Sync.
type SyncResult struct {
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Errors  []string `json:"errors,omitempty"`
}

// PanelArgs are the arguments passed to GetPanelData.
type PanelArgs struct {
	Kind      string
	Namespace string
	Name      string
}

// ActionArgs are the arguments passed to ExecuteAction.
type ActionArgs struct {
	ActionName string
	Inputs     map[string]string
}

// ActionResult is returned by ExecuteAction.
type ActionResult struct {
	Outputs map[string]string
}

// Route mirrors sdk.Route: a path prefix the plugin wants Gantry to proxy.
type Route struct {
	Path    string   `json:"path"`
	Methods []string `json:"methods,omitempty"`
}

// GantryPluginServer is the interface that plugin binaries implement.
type GantryPluginServer interface {
	GetManifest() (Manifest, error)
	Configure(config map[string]string) error
	Sync() (SyncResult, error)
	GetPanelData(args PanelArgs) (json.RawMessage, error)
	ExecuteAction(args ActionArgs) (ActionResult, error)
}

// GantryPluginRPC is the client-side interface used inside Gantry.
type GantryPluginRPC interface {
	GetManifest() (Manifest, error)
	GetListenAddr() (string, error)
	Configure(config map[string]string) error
	Sync() (SyncResult, error)
	GetPanelData(args PanelArgs) (json.RawMessage, error)
	ExecuteAction(args ActionArgs) (ActionResult, error)
}

type rpcClient struct {
	client *rpc.Client
}

func (c *rpcClient) GetManifest() (Manifest, error) {
	var reply Manifest
	return reply, c.client.Call("Plugin.GetManifest", new(struct{}), &reply)
}

func (c *rpcClient) GetListenAddr() (string, error) {
	var addr string
	return addr, c.client.Call("Plugin.GetListenAddr", struct{}{}, &addr)
}

func (c *rpcClient) Configure(config map[string]string) error {
	return c.client.Call("Plugin.Configure", &config, new(struct{}))
}

func (c *rpcClient) Sync() (SyncResult, error) {
	var reply SyncResult
	return reply, c.client.Call("Plugin.Sync", new(struct{}), &reply)
}

func (c *rpcClient) GetPanelData(args PanelArgs) (json.RawMessage, error) {
	var reply json.RawMessage
	return reply, c.client.Call("Plugin.GetPanelData", &args, &reply)
}

func (c *rpcClient) ExecuteAction(args ActionArgs) (ActionResult, error) {
	var reply ActionResult
	return reply, c.client.Call("Plugin.ExecuteAction", &args, &reply)
}

type rpcServer struct {
	impl GantryPluginServer
}

func (s *rpcServer) GetManifest(_ *struct{}, reply *Manifest) error {
	m, err := s.impl.GetManifest()
	if err != nil {
		return err
	}
	*reply = m
	return nil
}

func (s *rpcServer) GetListenAddr(_ *struct{}, reply *string) error {
	*reply = ""
	return nil
}

func (s *rpcServer) Configure(config *map[string]string, _ *struct{}) error {
	if config == nil {
		return s.impl.Configure(nil)
	}
	return s.impl.Configure(*config)
}

func (s *rpcServer) Sync(_ *struct{}, reply *SyncResult) error {
	r, err := s.impl.Sync()
	if err != nil {
		return err
	}
	*reply = r
	return nil
}

func (s *rpcServer) GetPanelData(args *PanelArgs, reply *json.RawMessage) error {
	if args == nil {
		args = &PanelArgs{}
	}
	data, err := s.impl.GetPanelData(*args)
	if err != nil {
		return err
	}
	*reply = data
	return nil
}

func (s *rpcServer) ExecuteAction(args *ActionArgs, reply *ActionResult) error {
	if args == nil {
		args = &ActionArgs{}
	}
	r, err := s.impl.ExecuteAction(*args)
	if err != nil {
		return err
	}
	*reply = r
	return nil
}

// GantryPlugin is the go-plugin Plugin interface implementation.
type GantryPlugin struct {
	Impl GantryPluginServer
}

func (p *GantryPlugin) Server(_ *plugin.MuxBroker) (interface{}, error) {
	return &rpcServer{impl: p.Impl}, nil
}

func (p *GantryPlugin) Client(_ *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &rpcClient{client: c}, nil
}

// PluginMap must use "Plugin" as the key — net/rpc registers methods under
// the struct type name (Plugin.GetManifest, Plugin.Configure, etc.).
var PluginMap = map[string]plugin.Plugin{
	"Plugin": &GantryPlugin{},
}

// Package external implements the Gantry external plugin system.
// External plugins are separate binaries discovered at runtime from a configured
// directory. They communicate with Gantry via gRPC using hashicorp/go-plugin
// for subprocess lifecycle management, allowing plugins to be written in any
// language that supports gRPC.
package external

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-plugin"
	pluginpb "github.com/go2engle/gantry/internal/plugins/external/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
	// ConfigSchemaJSON carries the JSON-encoded configSchema.
	ConfigSchemaJSON string `json:"configSchemaJson,omitempty"`
	SupportsHTTP     bool   `json:"supportsHttp,omitempty"`
	// HTTPRoutesJSON carries the JSON-encoded []Route.
	HTTPRoutesJSON string `json:"httpRoutesJson,omitempty"`
	// AuthBeginPath is the Gantry-relative URL path that starts the auth flow
	// for auth-provider plugins (e.g. "/api/v1/auth/authentik").
	// Currently unused: Gantry generates the login URL as /api/v1/auth/plugin/{name}
	// and handles the OIDC flow itself using the plugin's config.
	// TODO(option-b): when a plugin sets AuthBeginPath + SupportsHTTP + HTTPRoutesJSON,
	// Gantry should proxy the auth flow to the plugin's embedded HTTP server and expose
	// a POST /api/v1/internal/auth/session endpoint so the plugin can exchange user info
	// for a Gantry JWT. This allows plugins to implement custom auth flows (SAML, LDAP, etc.)
	// without Gantry needing to understand the protocol.
	AuthBeginPath string `json:"authBeginPath,omitempty"`
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
// Plugin authors implement this Go interface; the gRPC adapter layer translates
// between it and the generated protobuf types.
type GantryPluginServer interface {
	GetManifest() (Manifest, error)
	GetListenAddr() (string, error)
	Configure(config map[string]string) error
	Sync() (SyncResult, error)
	GetPanelData(args PanelArgs) (json.RawMessage, error)
	ExecuteAction(args ActionArgs) (ActionResult, error)
}

// gantryPluginClient is the client-side interface used inside Gantry.
// It mirrors GantryPluginServer and is implemented by grpcClientWrapper.
type gantryPluginClient interface {
	GetManifest() (Manifest, error)
	GetListenAddr() (string, error)
	Configure(config map[string]string) error
	Sync() (SyncResult, error)
	GetPanelData(args PanelArgs) (json.RawMessage, error)
	ExecuteAction(args ActionArgs) (ActionResult, error)
}

// ── gRPC server adapter ───────────────────────────────────────────────────────

// grpcPluginServer wraps GantryPluginServer and implements the generated
// pluginpb.GantryPluginServer proto interface for the server (plugin binary) side.
type grpcPluginServer struct {
	pluginpb.UnimplementedGantryPluginServer
	impl GantryPluginServer
}

func (s *grpcPluginServer) GetManifest(_ context.Context, _ *pluginpb.GetManifestRequest) (*pluginpb.ManifestResponse, error) {
	m, err := s.impl.GetManifest()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	reqs := make([]*pluginpb.Requirement, len(m.Requirements))
	for i, r := range m.Requirements {
		reqs[i] = &pluginpb.Requirement{Name: r.Name, Description: r.Description, Optional: r.Optional}
	}
	return &pluginpb.ManifestResponse{
		Name:             m.Name,
		Title:            m.Title,
		Description:      m.Description,
		Version:          m.Version,
		Author:           m.Author,
		Category:         m.Category,
		IconUrl:          m.IconURL,
		Homepage:         m.Homepage,
		SupportsSync:     m.SupportsSync,
		SupportsPanels:   m.SupportsPanels,
		SupportsActions:  m.SupportsActions,
		SupportsHttp:     m.SupportsHTTP,
		EntityPanels:     m.EntityPanels,
		ActionTypes:      m.ActionTypes,
		Requirements:     reqs,
		ConfigSchemaJson: m.ConfigSchemaJSON,
		HttpRoutesJson:   m.HTTPRoutesJSON,
		AuthBeginPath:    m.AuthBeginPath,
	}, nil
}

func (s *grpcPluginServer) GetListenAddr(_ context.Context, _ *pluginpb.GetListenAddrRequest) (*pluginpb.GetListenAddrResponse, error) {
	addr, err := s.impl.GetListenAddr()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &pluginpb.GetListenAddrResponse{Addr: addr}, nil
}

func (s *grpcPluginServer) Configure(_ context.Context, req *pluginpb.ConfigureRequest) (*pluginpb.ConfigureResponse, error) {
	if err := s.impl.Configure(req.Config); err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &pluginpb.ConfigureResponse{}, nil
}

func (s *grpcPluginServer) Sync(_ context.Context, _ *pluginpb.SyncRequest) (*pluginpb.SyncResponse, error) {
	r, err := s.impl.Sync()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &pluginpb.SyncResponse{
		Created: int32(r.Created),
		Updated: int32(r.Updated),
		Errors:  r.Errors,
	}, nil
}

func (s *grpcPluginServer) GetPanelData(_ context.Context, req *pluginpb.GetPanelDataRequest) (*pluginpb.GetPanelDataResponse, error) {
	data, err := s.impl.GetPanelData(PanelArgs{Kind: req.Kind, Namespace: req.Namespace, Name: req.Name})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &pluginpb.GetPanelDataResponse{PanelJson: []byte(data)}, nil
}

func (s *grpcPluginServer) ExecuteAction(_ context.Context, req *pluginpb.ExecuteActionRequest) (*pluginpb.ExecuteActionResponse, error) {
	r, err := s.impl.ExecuteAction(ActionArgs{ActionName: req.ActionName, Inputs: req.Inputs})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &pluginpb.ExecuteActionResponse{Outputs: r.Outputs}, nil
}

// ── gRPC client adapter ───────────────────────────────────────────────────────

// grpcClientWrapper wraps the generated pluginpb.GantryPluginClient and implements
// gantryPluginClient, converting proto types to the Go types used by ExternalPlugin.
type grpcClientWrapper struct {
	c pluginpb.GantryPluginClient
}

func (w *grpcClientWrapper) GetManifest() (Manifest, error) {
	resp, err := w.c.GetManifest(context.Background(), &pluginpb.GetManifestRequest{})
	if err != nil {
		return Manifest{}, fmt.Errorf("%s", status.Convert(err).Message())
	}
	reqs := make([]Requirement, len(resp.Requirements))
	for i, r := range resp.Requirements {
		reqs[i] = Requirement{Name: r.Name, Description: r.Description, Optional: r.Optional}
	}
	return Manifest{
		Name:             resp.Name,
		Title:            resp.Title,
		Description:      resp.Description,
		Version:          resp.Version,
		Author:           resp.Author,
		Category:         resp.Category,
		IconURL:          resp.IconUrl,
		Homepage:         resp.Homepage,
		SupportsSync:     resp.SupportsSync,
		SupportsPanels:   resp.SupportsPanels,
		SupportsActions:  resp.SupportsActions,
		SupportsHTTP:     resp.SupportsHttp,
		EntityPanels:     resp.EntityPanels,
		ActionTypes:      resp.ActionTypes,
		Requirements:     reqs,
		ConfigSchemaJSON: resp.ConfigSchemaJson,
		HTTPRoutesJSON:   resp.HttpRoutesJson,
		AuthBeginPath:    resp.AuthBeginPath,
	}, nil
}

func (w *grpcClientWrapper) GetListenAddr() (string, error) {
	resp, err := w.c.GetListenAddr(context.Background(), &pluginpb.GetListenAddrRequest{})
	if err != nil {
		return "", fmt.Errorf("%s", status.Convert(err).Message())
	}
	return resp.Addr, nil
}

func (w *grpcClientWrapper) Configure(config map[string]string) error {
	_, err := w.c.Configure(context.Background(), &pluginpb.ConfigureRequest{Config: config})
	if err != nil {
		return fmt.Errorf("%s", status.Convert(err).Message())
	}
	return nil
}

func (w *grpcClientWrapper) Sync() (SyncResult, error) {
	resp, err := w.c.Sync(context.Background(), &pluginpb.SyncRequest{})
	if err != nil {
		return SyncResult{}, fmt.Errorf("%s", status.Convert(err).Message())
	}
	return SyncResult{
		Created: int(resp.Created),
		Updated: int(resp.Updated),
		Errors:  resp.Errors,
	}, nil
}

func (w *grpcClientWrapper) GetPanelData(args PanelArgs) (json.RawMessage, error) {
	resp, err := w.c.GetPanelData(context.Background(), &pluginpb.GetPanelDataRequest{
		Kind:      args.Kind,
		Namespace: args.Namespace,
		Name:      args.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("%s", status.Convert(err).Message())
	}
	return json.RawMessage(resp.PanelJson), nil
}

func (w *grpcClientWrapper) ExecuteAction(args ActionArgs) (ActionResult, error) {
	resp, err := w.c.ExecuteAction(context.Background(), &pluginpb.ExecuteActionRequest{
		ActionName: args.ActionName,
		Inputs:     args.Inputs,
	})
	if err != nil {
		return ActionResult{}, fmt.Errorf("%s", status.Convert(err).Message())
	}
	return ActionResult{Outputs: resp.Outputs}, nil
}

// ── Plugin registration ───────────────────────────────────────────────────────

// GantryPlugin implements plugin.GRPCPlugin for the hashicorp/go-plugin framework.
// Plugin binaries set Impl to their server implementation and pass this to plugin.Serve.
type GantryPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	Impl GantryPluginServer
}

func (p *GantryPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginpb.RegisterGantryPluginServer(s, &grpcPluginServer{impl: p.Impl})
	return nil
}

func (p *GantryPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, conn *grpc.ClientConn) (interface{}, error) {
	return &grpcClientWrapper{c: pluginpb.NewGantryPluginClient(conn)}, nil
}

// PluginMap is passed to plugin.NewClient and plugin.Serve.
// The key "Plugin" is the dispense name and must match on both sides.
var PluginMap = map[string]plugin.Plugin{
	"Plugin": &GantryPlugin{},
}

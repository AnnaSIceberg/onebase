//go:build linux

package ui

// PRIVATE DIAGNOSTIC ONLY. No production hook or SDK change. This file is
// deliberately opt-in and requires independently reviewed private inputs.
// Not built/run during preparation of this diff.
import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"log/slog"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/access"
	"github.com/ivantit66/onebase/internal/api"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/extform"
	"github.com/ivantit66/onebase/internal/i18n"
	"github.com/ivantit66/onebase/internal/jobqueue"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/project"
	obruntime "github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/scheduler"
	"github.com/ivantit66/onebase/internal/storage"
)

const s2Base = "f49a9e0bfd2eb5184ce9e13e583a5a163e967534"
const s2Config = "2a964f3f03ad36125f41a65e3ac2e02f0da8985591d48293196a1369722c4dee"
const s2Overlay = "ed54e6717300e29a60b876eabcb64867861c1710"
const s2SourceRoot = "/home/onebase/cc2b-audio-task-polish-20261114"
const s2LoginSourceSHA = "e1b6dffeb3d601c022bc7b43551d9592859499775f6fe7ac23aa95c71835a8eb"
const s2LoginSourcePath = "C:/Users/a.sevrjukova/AppData/Local/Temp/cc2b-audio-task-polish-20261114/after_b10_browser.py"
const s2CoordinatorSHA = "f64fbb4856ab4fbabfad98d6c657c546445157a97913a6e85862126fc7131dd4"
const s2OwnershipSHA = "d8c0c41b97f4e68ab0affa5df3af163388f8c7e6df637c59d78db81c3682e1fe"
const s2MaxWire = 8 << 20
const s2MaxFlags = 64 << 10
const s2MaxEvents = 16

type s2FilePin struct {
	Path string `json:"path"`
	SHA  string `json:"sha256"`
}
type s2Input struct {
	Schema          string    `json:"schema"`
	Base            string    `json:"base_runtime"`
	ExecutableSHA   string    `json:"executable_sha256"`
	Work            string    `json:"work_root"`
	Mode            string    `json:"explicit_mode"`
	OriginalParity  string    `json:"original_strict_rls_parity"`
	ExperimentScope string    `json:"experiment_scope"`
	Provenance      s2FilePin `json:"mode_provenance"`
	Ownership       s2FilePin `json:"ownership_preflight"`
	SourceManifest  s2FilePin `json:"source_manifest"`
	Copy            struct {
		SourceDB        string            `json:"source_db"`
		SourceDBSHA     string            `json:"source_db_sha256"`
		DB              string            `json:"database"`
		Project         string            `json:"project"`
		Media           string            `json:"media"`
		Fresh           *bool             `json:"fresh_copy"`
		WALInactive     *bool             `json:"source_wal_inactive"`
		SourceImmutable *bool             `json:"source_immutable"`
		OwnerVerified   *bool             `json:"own_synthetic_verified"`
		MediaFiles      map[string]string `json:"media_files"`
	} `json:"copy"`
	BindingsFile s2FilePin `json:"private_bindings"`
	Login        struct {
		Source     s2FilePin `json:"existing_source"`
		Adapter    s2FilePin `json:"reviewed_adapter"`
		CaptureSHA string    `json:"capture_sha256"`
		Target     string    `json:"reviewed_target_origin"`
		ImportOnly *bool     `json:"capture_import_only"`
	} `json:"login"`
	Native struct {
		Known                  *bool             `json:"options_known"`
		Mode                   string            `json:"launch_mode"`
		ConfigSource           string            `json:"config_source"`
		H2C                    *bool             `json:"h2c"`
		StrictLexical          *bool             `json:"strict_lexical_scope"`
		UI                     json.RawMessage   `json:"ui_scalar_mapping"`
		API                    json.RawMessage   `json:"api_scalar_mapping"`
		Queue                  json.RawMessage   `json:"queue_mapping"`
		QueueConfiguredWorkers *int              `json:"queue_configured_workers"`
		QueueEffectiveWorkers  *int              `json:"queue_effective_workers"`
		Components             map[string]bool   `json:"components"`
		RolePolicies           json.RawMessage   `json:"actor_role_policies"`
		Platform               map[string]string `json:"platform_presentation"`
	} `json:"native"`
	Supervisor struct {
		Proof                     s2FilePin `json:"proof"`
		Namespace                 *bool     `json:"private_network"`
		BackendKernelNetDeny      *bool     `json:"backend_kernel_net_deny"`
		BrowserNetworkBoundary    string    `json:"browser_network_boundary"`
		BrowserOSNetworkIsolated  *bool     `json:"browser_os_network_isolated"`
		BrowserRoutingCacheEffect string    `json:"browser_routing_cache_effect"`
		BrowserJobBound           *bool     `json:"browser_job_bound"`
		JobsOff                   *bool     `json:"jobs_stopped"`
		ConnectionsZero           *bool     `json:"connections_zero"`
		Units                     []string  `json:"owned_units"`
		TunnelCount               *int      `json:"owned_tunnel_count"`
		BackendMemoryMax          *int64    `json:"backend_memory_max"`
		BrowserMemoryMax          *int64    `json:"browser_memory_max"`
		RuntimeMax                *int      `json:"runtime_max_seconds"`
		BrowserMax                *int      `json:"browser_max_seconds"`
		Port                      *int      `json:"backend_port"`
		FinallyContract           *bool     `json:"finally_cleanup_bound"`
	} `json:"supervisor"`
}

// Binding VALUES never enter a receipt, error, argv, environment or logger.
type s2Bindings struct {
	Schema   string   `json:"schema"`
	Own      *bool    `json:"own_synthetic"`
	ActorID  string   `json:"actor_id"`
	IsAdmin  *bool    `json:"is_admin"`
	Roles    []string `json:"role_names"`
	Entity   string   `json:"entity"`
	RecordID string   `json:"record_id"`
	Form     string   `json:"form"`
	Player   string   `json:"player"`
	FormOS   string   `json:"form_os_relative"`
}

type s2Manifest struct {
	Files   map[string]string `json:"files_sha256"`
	Archive string            `json:"config_archive_sha256"`
	Overlay string            `json:"overlay_commit"`
}
type s2ModeProof struct {
	Schema          string `json:"schema"`
	Mode            string `json:"explicit_mode"`
	Original        string `json:"original_parity"`
	ExperimentScope string `json:"experiment_scope"`
	Work            string `json:"work_root"`
	Fresh           *bool  `json:"fresh_copy"`
	Approved        *bool  `json:"execution_approved"`
}
type s2SupervisorProof struct {
	Schema                    string   `json:"schema"`
	Work                      string   `json:"work_root"`
	Mode                      string   `json:"explicit_mode"`
	ExperimentScope           string   `json:"experiment_scope"`
	OwnershipSHA              string   `json:"ownership_sha256"`
	LoginSourceSHA            string   `json:"login_source_sha256"`
	CoordinatorSHA            string   `json:"coordinator_sha256"`
	CaptureSHA                string   `json:"capture_sha256"`
	Origin                    string   `json:"origin"`
	NativeOptionsReviewed     *bool    `json:"native_mapping_reviewed"`
	SourceInactive            *bool    `json:"source_units_inactive"`
	Namespace                 *bool    `json:"private_network"`
	BackendKernelNetDeny      *bool    `json:"backend_kernel_net_deny"`
	BrowserNetworkBoundary    string   `json:"browser_network_boundary"`
	BrowserOSNetworkIsolated  *bool    `json:"browser_os_network_isolated"`
	BrowserRoutingCacheEffect string   `json:"browser_routing_cache_effect"`
	BrowserJobBound           *bool    `json:"browser_job_bound"`
	JobsOff                   *bool    `json:"jobs_stopped"`
	Finally                   *bool    `json:"finally_cleanup_bound"`
	SourceImmutable           *bool    `json:"source_immutable"`
	Units                     []string `json:"owned_units"`
	BackendMemoryMax          *int64   `json:"backend_memory_max"`
	BrowserMemoryMax          *int64   `json:"browser_memory_max"`
	RuntimeMax                *int     `json:"runtime_max_seconds"`
	BrowserMax                *int     `json:"browser_max_seconds"`
	TunnelCount               *int     `json:"owned_tunnel_count"`
}

// Written by the reviewed OWNER runner only AFTER coordinator context.close
// AND browser.close finally. This is a file handshake, never a debug route.
type s2BrowserDone struct {
	Schema        string `json:"schema"`
	Mode          string `json:"explicit_mode"`
	Stage         string `json:"stage"`
	ContextClosed *bool  `json:"context_closed"`
	BrowserClosed *bool  `json:"browser_closed"`
	CaptureSHA    string `json:"capture_sha256"`
}
type s2RequestKey struct{}
type s2Reason struct {
	Class string `json:"class"`
	Field string `json:"field,omitempty"`
	Type  string `json:"type,omitempty"`
}
type s2Flags struct {
	Sequence         int         `json:"sequence"`
	Event            string      `json:"event"`
	Element          string      `json:"element"`
	Point            string      `json:"point"`
	Mode             string      `json:"explicit_mode"`
	OriginalParity   string      `json:"original_parity"`
	Form             string      `json:"form,omitempty"`
	PlayerCount      int         `json:"player_count"`
	Header           bool        `json:"named_header_player"`
	Excluded         []string    `json:"excluded_metadata_names,omitempty"`
	Eligibility      []s2Carrier `json:"carrier_eligibility,omitempty"`
	Before           s2Reason    `json:"baseline_proxy"`
	Original         s2Reason    `json:"original_proxy"`
	Projected        s2Reason    `json:"projected_proxy"`
	ObservedDirty    *bool       `json:"observed_reply_dirty,omitempty"`
	Deferred         bool        `json:"deferred"`
	OK               bool        `json:"ok"`
	PolicyLive       bool        `json:"policy_context_live"`
	TxSettled        bool        `json:"transaction_settled"`
	SnapshotStable   bool        `json:"own_db_snapshot_stable"`
	NativeInterfaces bool        `json:"native_writer_interfaces"`
	UICommands       bool        `json:"native_ui_commands_header"`
	FirstWrite       bool        `json:"first_json_write_observed"`
	Gap              string      `json:"observation_gap,omitempty"`
	ReasonRequested  bool        `json:"reason_requested"`
}
type s2Carrier struct {
	Name       string   `json:"name"`
	Type       string   `json:"declared_type"`
	Scalar     bool     `json:"scalar"`
	Save       bool     `json:"save"`
	Main       bool     `json:"main"`
	Entity     bool     `json:"entity"`
	Service    bool     `json:"service"`
	Collisions []string `json:"non_audio_collision_names,omitempty"`
}
type s2Snapshot struct {
	Header map[string]any
	Tables map[string][]map[string]any
}
type s2Record struct {
	mu           sync.Mutex
	flag         s2Flags
	before       *obruntime.Object
	live         *formObjectThis
	entryDB      *s2Snapshot
	finished     bool
	inHook       bool
	wire         bytes.Buffer
	wireOverflow bool
	body         *s2BodyTee
	contentType  string
	metaDone     bool
}
type s2Capture struct {
	mu        sync.Mutex
	records   map[int]*s2Record
	completed []s2Flags
	count     int
	front     *Server
	binding   s2Bindings
	input     s2Input
	entity    *metadata.Entity
	stop      chan struct{}
	once      sync.Once
	sealed    bool
	globalGap string
}

func (c *s2Capture) requestStop() {
	c.mu.Lock()
	c.sealed = true
	c.mu.Unlock()
	c.once.Do(func() { close(c.stop) })
}
func (c *s2Capture) gapStop(code string) {
	c.mu.Lock()
	if c.globalGap == "" {
		c.globalGap = code
	}
	c.mu.Unlock()
	c.requestStop()
}
func (c *s2Capture) mark(r *s2Record, code string) {
	if r.flag.Gap == "" {
		r.flag.Gap = code
	}
}

// Observe native reads without pre-parsing, replacing bytes, swallowing a
// read error or imposing a different native request-body limit.
type s2BodyTee struct {
	io.ReadCloser
	mu       sync.Mutex
	b        bytes.Buffer
	overflow bool
}

func (b *s2BodyTee) Read(p []byte) (int, error) {
	n, e := b.ReadCloser.Read(p)
	b.mu.Lock()
	if b.b.Len()+n <= s2MaxWire {
		_, _ = b.b.Write(p[:n])
	} else {
		b.overflow = true
	}
	b.mu.Unlock()
	return n, e
}
func (c *s2Capture) meta(r *s2Record) {
	if r.metaDone {
		return
	}
	r.metaDone = true
	r.body.mu.Lock()
	defer r.body.mu.Unlock()
	if r.body.overflow {
		c.mark(r, "request_observation_limit")
		return
	}
	event, element, e := s2EventMeta(r.contentType, r.body.b.Bytes())
	if e != nil {
		c.mark(r, "request_metadata_gap")
		return
	}
	r.flag.Event = event
	r.flag.ReasonRequested = event == "Воспроизведение" && element == c.binding.Player
	if element == "" {
		return
	}
	known := false
	for _, f := range c.entity.Forms {
		if f.Name == c.binding.Form {
			walkBrowserFormElements(f, func(v browserFormElementVisit) {
				if strings.EqualFold(v.element.Name, element) {
					r.flag.Element = v.element.Name
					known = true
				}
			})
		}
	}
	if !known {
		r.flag.Element = "unknown"
		c.mark(r, "target_metadata_gap")
	}
}

// The installed observer is per interpreter exec, NOT a shared active/stack.
// It takes one entry copy and retains a live pointer; never evaluates DSL.
type s2Observer struct {
	c        *s2Capture
	captured bool
}

func (*s2Observer) HookCheckBreakpoint(string, int, func(string) (bool, error)) bool { return false }
func (h *s2Observer) HookShouldStep(file string, _ int) bool {
	h.c.mu.Lock()
	sealed := h.c.sealed
	h.c.mu.Unlock()
	if sealed {
		return false
	}
	return !h.captured && filepath.Clean(file) == filepath.Join(h.c.input.Copy.Project, filepath.FromSlash(h.c.binding.FormOS))
}
func (*s2Observer) HookPushFrame(string, int) {}
func (*s2Observer) HookPopFrame()             {}
func (h *s2Observer) HookOnPause(_ string, _ int, vars map[string]any, _ func(string) (any, error), _ string) {
	defer func() {
		if recover() != nil {
			h.c.gapStop("observer_panic")
		}
	}()
	if h.captured {
		return
	}
	for _, value := range vars {
		f, ok := value.(*formObjectThis)
		if !ok || f.obj == nil || f.entity == nil || f.entity.Name != h.c.binding.Entity || f.obj.ID.String() != h.c.binding.RecordID || f.ctx == nil {
			continue
		}
		sequence, ok := f.ctx.Value(s2RequestKey{}).(int)
		if !ok {
			continue
		}
		h.c.mu.Lock()
		r := h.c.records[sequence]
		h.c.mu.Unlock()
		if r == nil {
			continue
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		h.captured = true
		h.c.meta(r)
		if r.finished || r.inHook || r.live != nil {
			h.c.mark(r, "clone_race")
			return
		}
		r.inHook = true
		defer func() { r.inHook = false }()
		actor := auth.UserFromContext(f.ctx)
		if actor == nil || actor.ID != h.c.binding.ActorID || actor.IsAdmin != *h.c.binding.IsAdmin || !s2Same(s2RoleNames(actor.Roles), h.c.binding.Roles) {
			h.c.mark(r, "actor_policy_parity")
			return
		}
		if f.form == nil || f.form.Name != h.c.binding.Form {
			h.c.mark(r, "selected_form_parity")
			return
		}
		r.live = f
		r.flag.Form = f.form.Name
		if r.flag.Event != "Воспроизведение" || r.flag.Element != h.c.binding.Player {
			return
		}
		ctx, code := s2Live(f)
		if code != "" {
			h.c.mark(r, code)
			return
		}
		if !h.c.policyBound(ctx) {
			h.c.mark(r, "actor_policy_parity")
			return
		}
		var snap *s2Snapshot
		err := h.c.front.store.WithReadSnapshot(storage.ReadOnlyContext(ctx), func(sc context.Context) error {
			var e error
			snap, e = s2ReadSnapshot(sc, h.c.front.store, h.c.entity, f.obj.ID)
			return e
		})
		if err != nil {
			h.c.mark(r, "entry_snapshot_unavailable")
			return
		}
		r.before = audioF49Clone(f.obj)
		r.entryDB = snap
		return
	}
}

// No WithoutCancel, Background, request-context substitution, eval or retry.
func s2Live(f *formObjectThis) (context.Context, string) {
	if f == nil || f.ctx == nil || f.ctx.Err() != nil || f.ctxSrc == nil {
		return nil, "policy_context_closed"
	}
	ctx := f.ctxSrc.Ctx()
	if ctx == nil || ctx.Err() != nil {
		return nil, "transaction_context_closed"
	}
	if storage.HasTx(ctx) {
		return nil, "transaction_not_settled"
	}
	if tx, ok := f.ctxSrc.(interface{ HasOpen() bool }); ok && tx.HasOpen() {
		return nil, "transaction_not_settled"
	}
	return ctx, ""
}
func (c *s2Capture) policyBound(ctx context.Context) bool {
	u := auth.UserFromContext(ctx)
	if u == nil || u.ID != c.binding.ActorID || u.IsAdmin != *c.binding.IsAdmin || !s2Same(s2RoleNames(u.Roles), c.binding.Roles) {
		return false
	}
	policies := map[string]auth.Permission{}
	for _, role := range u.Roles {
		if role == nil {
			return false
		}
		policies[role.Name] = role.Permissions
	}
	return s2Witness(c.input.Native.RolePolicies, policies)
}
func s2ReadSnapshot(ctx context.Context, db *storage.DB, e *metadata.Entity, id uuid.UUID) (*s2Snapshot, error) {
	h, err := db.GetByID(ctx, e.Name, id, e)
	if err != nil {
		return nil, err
	}
	s := &s2Snapshot{Header: cloneRecord(h), Tables: map[string][]map[string]any{}}
	for _, tp := range e.TableParts {
		rows, err := db.GetTablePartRows(ctx, e.Name, tp.Name, id, tp)
		if err != nil {
			return nil, err
		}
		s.Tables[tp.Name] = cloneTablePartRowMap(map[string][]map[string]any{tp.Name: rows})[tp.Name]
	}
	return s, nil
}

// Called synchronously before forwarding the FIRST native JSON bytes. The
// native guard and serialization already made their decision. Read-only own
// snapshot queries run while observed policy context is live, not after HTTP.
func (c *s2Capture) firstWrite(r *s2Record) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer func() {
		if recover() != nil {
			c.mark(r, "observer_panic")
		}
	}()
	c.meta(r)
	r.flag.FirstWrite = true
	r.flag.Point = "first_native_json_write_pre_forward_proxy"
	if r.flag.Event != "Воспроизведение" || r.flag.Element != c.binding.Player {
		return
	}
	if r.finished || r.inHook {
		c.mark(r, "clone_race")
		return
	}
	if r.flag.Gap != "" || r.live == nil || r.before == nil || r.entryDB == nil {
		c.mark(r, "entry_observation_missing")
		return
	}
	ctx, code := s2Live(r.live)
	if code != "" {
		c.mark(r, code)
		return
	}
	if !c.policyBound(ctx) {
		c.mark(r, "actor_policy_parity")
		return
	}
	r.flag.PolicyLive = true
	r.flag.TxSettled = true
	if r.live.obj == nil || r.live.obj.ID.String() != c.binding.RecordID || r.live.form == nil || r.live.form.Name != c.binding.Form {
		c.mark(r, "live_scope_changed")
		return
	}
	after := audioF49Clone(r.live.obj)
	form := r.live.form
	projection, excluded := audioPlaybackComparison(form, c.entity, r.flag.Element, r.flag.Event)
	for name := range excluded {
		r.flag.Excluded = append(r.flag.Excluded, name)
	}
	sort.Strings(r.flag.Excluded)
	walkBrowserFormElements(form, func(v browserFormElementVisit) {
		if v.element.Name == c.binding.Player {
			r.flag.PlayerCount++
			r.flag.Header = v.element.Kind == metadata.FormElementAudio && v.element.Placement == metadata.FormPlacementHeaderEnd
		}
	})
	if r.flag.PlayerCount != 1 || !r.flag.Header {
		c.mark(r, "named_header_player_parity")
		return
	}
	for _, a := range form.Attributes {
		if a == nil {
			continue
		}
		wanted := false
		walkBrowserFormElements(form, func(v browserFormElementVisit) {
			if v.element.Name == c.binding.Player && (strings.EqualFold(dpFieldName(v.element.DataPath), a.Name) || strings.EqualFold(dpFieldName(v.element.TitleFrom), a.Name)) {
				wanted = true
			}
		})
		if !wanted {
			continue
		}
		cf := s2Carrier{Name: a.Name, Type: a.TypeRef, Scalar: formAttrIsScalar(a), Save: a.Save, Main: a.MainAttribute, Entity: entityField(c.entity, a.Name) != nil, Service: audioPlaybackServiceField(a.Name)}
		walkBrowserFormElements(form, func(v browserFormElementVisit) {
			if v.element.Kind != metadata.FormElementAudio && strings.EqualFold(dpFieldName(v.element.DataPath), a.Name) {
				cf.Collisions = append(cf.Collisions, v.element.Name)
			}
		})
		r.flag.Eligibility = append(r.flag.Eligibility, cf)
	}
	beforeFields := snapshotFieldValues(r.before.Fields)
	beforeTables := cloneTablePartRowMap(r.before.TablePartRows)
	err := c.front.store.WithReadSnapshot(storage.ReadOnlyContext(ctx), func(sc context.Context) error {
		now, err := s2ReadSnapshot(sc, c.front.store, c.entity, r.live.obj.ID)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(r.entryDB, now) {
			c.mark(r, "db_snapshot_changed")
			return nil
		}
		r.flag.SnapshotStable = true
		r.flag.Before = s2Classify(c.front.managedCloseStateDirtyReason(sc, c.entity, form, r.before, beforeFields, beforeTables, now.Tables), c.entity, form)
		r.flag.Original = s2Classify(c.front.managedCloseStateDirtyReason(sc, c.entity, form, after, beforeFields, beforeTables, now.Tables), c.entity, form)
		r.flag.Projected = s2Classify(c.front.managedCloseStateDirtyReason(sc, c.entity, projection, after, beforeFields, beforeTables, now.Tables), c.entity, form)
		return nil
	})
	if err != nil {
		c.mark(r, "reason_snapshot_unavailable")
	}
	if r.flag.Before.Class == "unknown" || r.flag.Original.Class == "unknown" || r.flag.Projected.Class == "unknown" {
		c.mark(r, "reason_class_unknown")
	}
	if _, code = s2Live(r.live); code != "" {
		c.mark(r, code)
	}
	// Best-effort detection of a changed live object across the observation.
	// This is not an assertion that arbitrary concurrent/ABA writes are safe.
	if !reflect.DeepEqual(after, audioF49Clone(r.live.obj)) {
		c.mark(r, "clone_race")
	}
}
func s2Classify(raw string, e *metadata.Entity, f *metadata.FormModule) s2Reason {
	if raw == "" {
		return s2Reason{Class: "clean"}
	}
	for _, field := range e.Fields {
		if strings.HasPrefix(raw, "entity-field "+field.Name+" ") {
			return s2Reason{Class: "entity_field", Field: field.Name, Type: string(field.Type)}
		}
	}
	for _, tp := range e.TableParts {
		if strings.HasPrefix(raw, "table-part "+tp.Name+" ") {
			return s2Reason{Class: "table_part", Field: tp.Name}
		}
	}
	for _, a := range f.Attributes {
		if a != nil {
			if strings.HasPrefix(raw, "form-attr "+a.Name+" ") {
				return s2Reason{Class: "form_attr", Field: a.Name, Type: a.TypeRef}
			}
			if strings.HasPrefix(raw, "form-table "+a.Name+" ") {
				return s2Reason{Class: "form_table", Field: a.Name, Type: a.TypeRef}
			}
		}
	}
	for prefix, class := range map[string]string{"no-record:": "no_record", "record-read-error:": "record_read_error", "record-missing:": "record_missing", "service-field ": "service_field"} {
		if strings.HasPrefix(raw, prefix) {
			return s2Reason{Class: class}
		}
	}
	return s2Reason{Class: "unknown"}
}

type s2JSONWriter struct {
	http.ResponseWriter
	c     *s2Capture
	r     *s2Record
	first bool
}

func (w *s2JSONWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *s2JSONWriter) Write(p []byte) (int, error) {
	if !w.first && len(p) > 0 {
		w.first = true
		typ, _, e := mime.ParseMediaType(w.Header().Get("Content-Type"))
		if e == nil && typ == "application/json" {
			w.c.firstWrite(w.r)
		} else {
			w.r.mu.Lock()
			w.c.mark(w.r, "not_native_json")
			w.r.mu.Unlock()
		}
	}
	w.r.mu.Lock()
	if w.r.wire.Len()+len(p) <= s2MaxWire {
		_, _ = w.r.wire.Write(p)
	} else {
		w.r.wireOverflow = true
	}
	w.r.mu.Unlock()
	return w.ResponseWriter.Write(p) // identical bytes, headers/status untouched
}

// Actual H1 server interface set is verified before wrapping. No added H2
// interface, no wrapper on SSE/HTML/static/auth/service paths.
type s2H1Writer struct{ *s2JSONWriter }

func (w *s2H1Writer) Flush() { w.ResponseWriter.(http.Flusher).Flush() }
func (w *s2H1Writer) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}
func (w *s2H1Writer) CloseNotify() <-chan bool {
	return w.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

type s2WriteOnly struct{ w io.Writer }

func (w s2WriteOnly) Write(p []byte) (int, error)         { return w.w.Write(p) }
func (w *s2H1Writer) ReadFrom(r io.Reader) (int64, error) { return io.Copy(s2WriteOnly{w}, r) }

func (c *s2Capture) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/ui/document/"+c.binding.Entity+"/form-event" {
			next.ServeHTTP(w, r)
			return
		}
		c.mu.Lock()
		sealed := c.sealed
		c.mu.Unlock()
		if sealed {
			next.ServeHTTP(w, r)
			return
		}
		_, fl := w.(http.Flusher)
		_, hj := w.(http.Hijacker)
		_, rf := w.(io.ReaderFrom)
		_, cn := w.(http.CloseNotifier)
		_, push := w.(http.Pusher)
		if r.ProtoMajor != 1 || !fl || !hj || !rf || !cn || push {
			c.gapStop("native_writer_interface_gap")
			next.ServeHTTP(w, r)
			return
		}
		c.mu.Lock()
		c.count++
		sequence := c.count
		if sequence > s2MaxEvents {
			c.mu.Unlock()
			c.gapStop("event_limit")
			next.ServeHTTP(w, r)
			return
		}
		// Native parser runs first and receives unchanged Read n/errors/bytes.
		body := &s2BodyTee{ReadCloser: r.Body}
		r.Body = body
		rec := &s2Record{body: body, contentType: r.Header.Get("Content-Type"), flag: s2Flags{Sequence: sequence, Mode: c.input.Mode, OriginalParity: "UNKNOWN", NativeInterfaces: true, UICommands: r.Header.Get("X-OneBase-UI-Commands") == "1", Before: s2Reason{Class: "unknown"}, Original: s2Reason{Class: "unknown"}, Projected: s2Reason{Class: "unknown"}}}
		c.records[sequence] = rec
		c.mu.Unlock()
		r = r.WithContext(context.WithValue(r.Context(), s2RequestKey{}, sequence))
		defer func() {
			// AFTER ServeHTTP: parse bounded reply/flush flags ONLY. No queries,
			// reason calculation, context revival or live-object reads here.
			rec.mu.Lock()
			defer rec.mu.Unlock()
			rec.finished = true
			// Metadata parsing of bytes already consumed by native is RAM-only.
			c.meta(rec)
			event, element := rec.flag.Event, rec.flag.Element
			var reply struct {
				OK       bool  `json:"ok"`
				Dirty    *bool `json:"dirty"`
				Deferred bool  `json:"deferred"`
				Cancel   bool  `json:"cancel"`
			}
			if rec.wireOverflow || json.Unmarshal(rec.wire.Bytes(), &reply) != nil {
				c.mark(rec, "reply_capture_gap")
			} else {
				rec.flag.OK = reply.OK
				rec.flag.Deferred = reply.Deferred
				rec.flag.ObservedDirty = reply.Dirty
				if event == "Воспроизведение" && element == c.binding.Player {
					if !rec.flag.FirstWrite || reply.Dirty == nil || !reply.OK || reply.Cancel {
						c.mark(rec, "reply_semantic_gap")
					}
					if rec.flag.Gap == "" && (rec.flag.Projected.Class != "clean") != *reply.Dirty {
						c.mark(rec, "projection_response_gap")
					}
					if !rec.flag.UICommands {
						c.mark(rec, "ui_header_parity")
					}
				}
			}
			for i := range rec.wire.Bytes() {
				rec.wire.Bytes()[i] = 0
			}
			rec.wire.Reset()
			rec.body.mu.Lock()
			for i := range rec.body.b.Bytes() {
				rec.body.b.Bytes()[i] = 0
			}
			rec.body.b.Reset()
			rec.body.mu.Unlock()
			rec.before = nil
			rec.live = nil
			rec.entryDB = nil
			c.mu.Lock()
			c.completed = append(c.completed, rec.flag)
			delete(c.records, sequence)
			c.mu.Unlock()
			if rec.flag.Gap != "" || (event == "Воспроизведение" && element == c.binding.Player) {
				c.requestStop()
			}
		}()
		next.ServeHTTP(&s2H1Writer{&s2JSONWriter{ResponseWriter: w, c: c, r: rec}}, r)
	})
}
func s2EventMeta(contentType string, raw []byte) (string, string, error) {
	typ, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", "", err
	}
	values := map[string][]string{}
	if typ == "application/x-www-form-urlencoded" {
		v, e := url.ParseQuery(string(raw))
		if e != nil {
			return "", "", e
		}
		values = v
	} else if typ == "multipart/form-data" {
		mr := multipart.NewReader(bytes.NewReader(raw), params["boundary"])
		for {
			p, e := mr.NextPart()
			if e == io.EOF {
				break
			}
			if e != nil {
				return "", "", e
			}
			if p.FormName() == "_event" || p.FormName() == "_element" {
				b, e := io.ReadAll(io.LimitReader(p, 257))
				if e != nil || len(b) > 256 {
					return "", "", errors.New("metadata_limit")
				}
				values[p.FormName()] = append(values[p.FormName()], string(b))
			}
			_ = p.Close()
		}
	} else {
		return "", "", errors.New("content_type")
	}
	if len(values["_event"]) != 1 || len(values["_element"]) > 1 {
		return "", "", errors.New("metadata_cardinality")
	}
	event := strings.TrimSpace(values["_event"][0])
	if !metadata.IsKnownFormEventType(metadata.FormEventType(event)) {
		return "", "", errors.New("metadata_event")
	}
	element := ""
	if len(values["_element"]) == 1 {
		element = strings.TrimSpace(values["_element"][0])
	}
	return event, element, nil
}

func s2Same(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}
func s2RoleNames(roles []*auth.Role) []string {
	r := []string{}
	for _, role := range roles {
		if role != nil {
			r = append(r, role.Name)
		}
	}
	sort.Strings(r)
	return r
}
func s2Hash(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	_, e = io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), e
}
func s2SHA(s string) bool {
	if len(s) != 64 || s != strings.ToLower(s) {
		return false
	}
	_, e := hex.DecodeString(s)
	return e == nil
}
func s2Pinned(p s2FilePin) bool {
	if !s2SHA(p.SHA) {
		return false
	}
	h, e := s2Hash(p.Path)
	return e == nil && h == p.SHA
}
func s2Private(path string) bool {
	fi, e := os.Lstat(path)
	if e != nil || !fi.Mode().IsRegular() || fi.Mode().Perm() != 0600 {
		return false
	}
	s, ok := fi.Sys().(*syscall.Stat_t)
	return ok && int(s.Uid) == os.Geteuid() && s.Nlink == 1
}
func s2Under(root, path string) bool {
	a, e := filepath.EvalSymlinks(root)
	if e != nil {
		return false
	}
	b, e := filepath.EvalSymlinks(path)
	if e != nil {
		return false
	}
	rel, e := filepath.Rel(a, b)
	return e == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel) && filepath.Clean(path) == b
}
func s2JSON(path string, dst any) error {
	if !s2Private(path) {
		return errors.New("private_file")
	}
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 2<<20))
	d.DisallowUnknownFields()
	if e = d.Decode(dst); e != nil {
		return e
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return errors.New("trailing_json")
	}
	return nil
}
func s2True(p *bool) bool  { return p != nil && *p }
func s2False(p *bool) bool { return p != nil && !*p }

// Mirrors registryFromProject, no reduced entities/forms/modules.
func s2Registry(p *project.Project) *obruntime.Registry {
	r := obruntime.NewRegistry()
	r.Load(obruntime.LoadOptions{Entities: p.Entities, Programs: p.Programs, ManagerPrograms: p.ManagerPrograms, ServicePrograms: p.ServicePrograms, PagePrograms: p.PagePrograms, Registers: p.Registers, InfoRegs: p.InfoRegisters, Enums: p.Enums, Constants: p.Constants, Reports: p.Reports, PrintForms: p.PrintForms})
	r.LoadDSLPrintForms(p.DSLPrintForms)
	r.LoadLayoutForms(p.LayoutForms)
	r.LoadModules(p.Modules)
	r.LoadProcessors(p.Processors)
	r.LoadHTTPServices(p.HTTPServices)
	r.LoadPages(p.Pages)
	r.LoadExchangePlans(p.ExchangePlans)
	r.LoadIntakes(p.Intakes)
	r.LoadSubsystems(p.Subsystems)
	r.LoadJournals(p.Journals)
	r.LoadAccountRegisters(p.AccountRegisters, p.ChartsOfAccounts)
	r.LoadWidgets(p.Widgets)
	r.LoadHomePage(p.HomePage)
	r.LoadHelp(p.Help)
	return r
}
func s2Queue(a *project.AppConfig) jobqueue.Config {
	c := jobqueue.DefaultConfig()
	if a.Queue == nil {
		return c
	}
	q := a.Queue
	if q.Workers != nil {
		c.Workers = *q.Workers
	}
	if q.PollIntervalSec > 0 {
		c.PollInterval = time.Duration(q.PollIntervalSec) * time.Second
	}
	if q.LeaseSec > 0 {
		c.Lease = time.Duration(q.LeaseSec) * time.Second
	}
	if q.MaxAttempts > 0 {
		c.MaxAttempts = q.MaxAttempts
	}
	if q.RetryBackoffSec > 0 {
		c.RetryBackoff = time.Duration(q.RetryBackoffSec) * time.Second
	}
	if q.RetentionDays > 0 {
		c.Retention = time.Duration(q.RetentionDays) * 24 * time.Hour
	}
	if q.DrainTimeoutSec > 0 {
		c.DrainTimeout = time.Duration(q.DrainTimeoutSec) * time.Second
	}
	return c
}
func s2UI(a *project.AppConfig, in s2Input) (Config, error) {
	for _, key := range []string{"version", "commit", "date", "author", "license"} {
		if in.Native.Platform[key] == "" {
			return Config{}, errors.New("platform_witness_missing")
		}
	}
	c := Config{DatabaseType: "sqlite", DatabaseLocation: in.Copy.DB, ConfigSource: "file", ConfigLocation: in.Copy.Project, PlatVersion: in.Native.Platform["version"], PlatCommit: in.Native.Platform["commit"], PlatDate: in.Native.Platform["date"], PlatAuthor: in.Native.Platform["author"], PlatLicense: in.Native.Platform["license"]}
	c.AppName = a.Name
	c.AppVersion = a.Version
	c.AppAuthor = a.Author
	c.AppCopyright = a.Copyright
	c.AppLicense = a.License
	c.AppSupport = a.Support
	c.Lang = a.Lang
	if a.Logo != "" {
		c.Logo = filepath.Join(in.Copy.Project, a.Logo)
	}
	if a.Attachments != nil {
		if a.Attachments.MaxFileSizeMB > 0 {
			c.MaxFileSizeMB = a.Attachments.MaxFileSizeMB
		}
		c.AllowedTypes = a.Attachments.AllowedTypes
	}
	if l := a.Limits; l != nil {
		c.Limits = RuntimeLimits{RequestTimeoutSec: l.RequestTimeoutSec, ReportTimeoutSec: l.ReportTimeoutSec, ReportMaxRows: l.ReportMaxRows, ReportConcurrency: l.ReportConcurrency, ExportTimeoutSec: l.ExportTimeoutSec, ExportMaxRows: l.ExportMaxRows, ExportConcurrency: l.ExportConcurrency, ProcessorTimeoutSec: l.ProcessorTimeoutSec, ProcessorConcurrency: l.ProcessorConcurrency, HTTPServiceTimeoutSec: l.HTTPServiceTimeoutSec, HTTPServiceConcurrency: l.HTTPServiceConcurrency, SlowOperationMS: l.SlowOperationMS}
	}
	c.POSEnabled = a.POSEnabled()
	c.CompactLists = a.CompactLists()
	c.HideListDetailPanel = !a.ListDetailPanelEnabled()
	c.HideListRowActions = !a.ListRowActionsEnabled()
	c.HideAttachments = !a.AttachmentsEnabled()
	c.UIEnhancements = a.UIEnhancementsConfigured()
	if u := a.UI; u != nil {
		c.UIRoles = u.Roles
		c.UIExceptRoles = u.ExceptRoles
		c.UIFormRoles = u.FormRoles
		c.UIFormExceptRoles = u.FormExceptRoles
		c.DefaultFormDensity = u.FormDensity
		c.UIReportRoles = u.ReportRoles
		c.UIReportExceptRoles = u.ReportExceptRoles
		c.DefaultReportDensity = u.ReportDensity
		c.ShortDateInput = u.ShortDateInput
	}
	var err error
	c.Bundle, err = i18n.Load(i18n.EmbeddedLocales, filepath.Join(in.Copy.Project, "locales"))
	return c, err
}
func s2ScalarMap(v any, skip map[string]bool) map[string]any {
	r := map[string]any{}
	x := reflect.ValueOf(v)
	t := x.Type()
	for i := 0; i < x.NumField(); i++ {
		if !skip[t.Field(i).Name] {
			r[t.Field(i).Name] = x.Field(i).Interface()
		}
	}
	return r
}
func s2Witness(raw json.RawMessage, v any) bool {
	if len(raw) == 0 {
		return false
	}
	var a, b any
	if json.Unmarshal(raw, &a) != nil {
		return false
	}
	p, e := json.Marshal(v)
	if e != nil || json.Unmarshal(p, &b) != nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}
func s2Dependencies(a *project.AppConfig, in s2Input) bool {
	// No silent nil for a native dependency that could affect the form. Such a
	// shape requires a separate concrete proposal, never a test fallback.
	for _, key := range []string{"scheduler", "scheduler_jobs_loaded_dormant", "queue", "bundle", "login_limit"} {
		v, ok := in.Native.Components[key]
		if !ok || !v {
			return false
		}
	}
	for _, key := range []string{"mailer", "webhooks", "debug_routes", "metrics", "workers_running"} {
		v, ok := in.Native.Components[key]
		if !ok || v {
			return false
		}
	}
	v := reflect.ValueOf(a).Elem()
	for _, key := range []string{"Email", "Webhooks", "FileStorage", "Backup", "Demo", "AI", "LLM"} {
		x := v.FieldByName(key)
		if x.IsValid() && !x.IsZero() {
			return false
		}
	}
	return true
}

func s2Preflight(in s2Input) (s2Bindings, s2Manifest, string) {
	var b s2Bindings
	var m s2Manifest
	if in.Schema != "onebase.audio.stage2.private/v1" || in.Base != s2Base || in.OriginalParity != "UNKNOWN" || in.ExperimentScope != "controlled_own_synthetic" || (in.Mode != "strict_false" && in.Mode != "strict_true") {
		return b, m, "mode_provenance"
	}
	work := filepath.Clean(in.Work)
	suffix := strings.TrimPrefix(work, "/home/onebase/platform-audio-stage2-20261114-")
	if suffix == work || len(suffix) != 8 {
		return b, m, "work_root"
	}
	if _, e := hex.DecodeString(suffix); e != nil {
		return b, m, "work_root"
	}
	wi, e := os.Lstat(work)
	if e != nil || !wi.IsDir() || wi.Mode().Perm() != 0700 || wi.Mode()&os.ModeSymlink != 0 {
		return b, m, "work_root"
	}
	ws, ok := wi.Sys().(*syscall.Stat_t)
	real, e := filepath.EvalSymlinks(work)
	if !ok || int(ws.Uid) != os.Geteuid() || e != nil || real != work {
		return b, m, "work_root"
	}
	if !s2WorkBound(work) {
		return b, m, "work_size_or_file_kind"
	}
	for _, pin := range []s2FilePin{in.Provenance, in.Ownership, in.Supervisor.Proof} {
		if !s2Under(work, pin.Path) || !s2Private(pin.Path) {
			return b, m, "private_proof_path"
		}
	}
	if !s2Pinned(in.Provenance) || !s2Pinned(in.Ownership) || in.Ownership.SHA != s2OwnershipSHA {
		return b, m, "ownership_or_mode_proof"
	}
	var mode s2ModeProof
	if s2JSON(in.Provenance.Path, &mode) != nil || mode.Schema != "onebase.audio.stage2.mode/v1" || mode.Mode != in.Mode || mode.Original != "UNKNOWN" || mode.ExperimentScope != in.ExperimentScope || mode.Work != in.Work || !s2True(mode.Fresh) || !s2True(mode.Approved) {
		return b, m, "mode_provenance_missing"
	}
	if in.Copy.SourceDB != s2SourceRoot+"/after-b10.sqlite" || !s2True(in.Copy.Fresh) || !s2True(in.Copy.WALInactive) || !s2True(in.Copy.SourceImmutable) || !s2True(in.Copy.OwnerVerified) {
		return b, m, "source_copy_attestation"
	}
	for _, p := range []string{in.Copy.Project, in.Copy.DB, in.Copy.Media, in.BindingsFile.Path} {
		if !s2Under(work, p) {
			return b, m, "copy_path"
		}
	}
	if in.Copy.DB != filepath.Join(work, "after-b10.sqlite") || in.Copy.Project != filepath.Join(work, "config-after-b10") || in.Copy.Media != filepath.Join(work, ".onebase/files/after-b10") {
		return b, m, "native_path_shape"
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, e := os.Lstat(in.Copy.SourceDB + suffix); !os.IsNotExist(e) {
			return b, m, "source_wal_or_shm"
		}
	}
	src, e := os.Lstat(in.Copy.SourceDB)
	if e != nil || !src.Mode().IsRegular() || src.Size() > 512<<20 {
		return b, m, "source_size"
	}
	if h, e := s2Hash(in.Copy.SourceDB); e != nil || h != in.Copy.SourceDBSHA {
		return b, m, "source_identity"
	}
	if !s2Private(in.Copy.DB) {
		return b, m, "copy_permissions"
	}
	if h, e := s2Hash(in.Copy.DB); e != nil || h != in.Copy.SourceDBSHA {
		return b, m, "fresh_copy_identity"
	}
	if !s2Pinned(in.SourceManifest) || in.SourceManifest.Path != s2SourceRoot+"/after-b10-manifest.json" {
		return b, m, "source_manifest"
	}
	// Frozen native manifest has additional metadata; decode ONLY required
	// members in RAM, never emit it (it contains own IDs/media paths).
	mi, e := os.Stat(in.SourceManifest.Path)
	if e != nil || !mi.Mode().IsRegular() || mi.Size() > 2<<20 {
		return b, m, "frozen_config"
	}
	data, e := os.ReadFile(in.SourceManifest.Path)
	if e != nil || json.Unmarshal(data, &m) != nil || len(m.Files) != 2131 || m.Archive != s2Config || m.Overlay != s2Overlay {
		return b, m, "frozen_config"
	}
	if !s2VerifyTree(in.Copy.Project, m.Files, 256<<20) {
		return b, m, "copy_config_identity"
	}
	if !s2VerifyTree(s2SourceRoot+"/config-after-b10", m.Files, 256<<20) {
		return b, m, "source_config_identity"
	}
	if len(in.Copy.MediaFiles) == 0 || !s2VerifyTree(in.Copy.Media, in.Copy.MediaFiles, 64<<20) {
		return b, m, "own_media_identity"
	}
	if !s2Pinned(in.BindingsFile) || s2JSON(in.BindingsFile.Path, &b) != nil || b.Schema != "onebase.audio.stage2.bindings/v1" || !s2True(b.Own) || b.IsAdmin == nil || b.Roles == nil {
		return b, m, "private_bindings"
	}
	if id, e := uuid.Parse(b.ActorID); e != nil || id == uuid.Nil || id.String() != b.ActorID {
		return b, m, "private_bindings"
	}
	if id, e := uuid.Parse(b.RecordID); e != nil || id == uuid.Nil || id.String() != b.RecordID {
		return b, m, "private_bindings"
	}
	if *b.IsAdmin || !s2Same(b.Roles, []string{"ОТКСотрудник"}) {
		return b, m, "actor_binding_scope"
	}
	if b.Entity != "А_Звонок" || b.Form != "ФормаСАудиоплеером" || b.Player != "ПлеерЗаписи" || b.FormOS != "forms/а_звонок/прослушивание.form.os" {
		return b, m, "binding_scope"
	}
	// Login source and coordinator live in the owner's WINDOWS process, not
	// this Linux backend. Only a pinned owner attestation is consumed here;
	// neither source harness nor credentials are imported/read/executed.
	if in.Login.Source.Path != s2LoginSourcePath || in.Login.Source.SHA != s2LoginSourceSHA || !s2True(in.Login.ImportOnly) {
		return b, m, "login_provenance"
	}
	if filepath.Base(in.Login.Adapter.Path) != "stage2_login_coordinator_v2.py" || in.Login.Adapter.SHA != s2CoordinatorSHA || !s2SHA(in.Login.CaptureSHA) {
		return b, m, "reviewed_login_adapter_missing"
	}
	if !s2True(in.Native.Known) || in.Native.Mode != "run" || in.Native.ConfigSource != "file" || in.Native.H2C == nil || *in.Native.H2C || in.Native.StrictLexical == nil {
		return b, m, "native_options_missing"
	}
	s := in.Supervisor
	if !s2Pinned(s.Proof) || !s2True(s.Namespace) || !s2True(s.BackendKernelNetDeny) || !s2True(s.JobsOff) || !s2True(s.ConnectionsZero) || !s2True(s.FinallyContract) || len(s.Units) != 4 || s.TunnelCount == nil || *s.TunnelCount != 1 {
		return b, m, "supervisor_proof"
	}
	if s.BrowserNetworkBoundary != "owner_context_loopback_allowlist" || !s2False(s.BrowserOSNetworkIsolated) || s.BrowserRoutingCacheEffect != "disabled_by_context_route" || !s2True(s.BrowserJobBound) {
		return b, m, "browser_boundary_provenance"
	}
	var proof s2SupervisorProof
	if s2JSON(s.Proof.Path, &proof) != nil || proof.Schema != "onebase.audio.stage2.supervisor/v1" || proof.Work != work || proof.Mode != in.Mode || proof.ExperimentScope != in.ExperimentScope || proof.OwnershipSHA != s2OwnershipSHA || proof.LoginSourceSHA != s2LoginSourceSHA || proof.CoordinatorSHA != s2CoordinatorSHA || proof.CaptureSHA != in.Login.CaptureSHA || proof.BrowserNetworkBoundary != s.BrowserNetworkBoundary || !s2Same(proof.BrowserOSNetworkIsolated, s.BrowserOSNetworkIsolated) || proof.BrowserRoutingCacheEffect != s.BrowserRoutingCacheEffect || !s2Same(proof.BrowserJobBound, s.BrowserJobBound) || proof.Origin != in.Login.Target || !s2True(proof.NativeOptionsReviewed) || !s2True(proof.SourceInactive) || !s2True(proof.Namespace) || !s2True(proof.BackendKernelNetDeny) || !s2True(proof.JobsOff) || !s2True(proof.Finally) || !s2True(proof.SourceImmutable) || !s2Same(proof.Units, s.Units) || !s2Same(proof.BackendMemoryMax, s.BackendMemoryMax) || !s2Same(proof.BrowserMemoryMax, s.BrowserMemoryMax) || !s2Same(proof.RuntimeMax, s.RuntimeMax) || !s2Same(proof.BrowserMax, s.BrowserMax) || !s2Same(proof.TunnelCount, s.TunnelCount) {
		return b, m, "supervisor_mapping_gap"
	}
	seenUnits := map[string]bool{}
	for _, n := range s.Units {
		if !strings.HasPrefix(n, filepath.Base(work)+"-") || seenUnits[n] {
			return b, m, "unit_scope"
		}
		seenUnits[n] = true
		for _, ch := range n {
			if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '.' || ch == '_') {
				return b, m, "unit_scope"
			}
		}
	}
	if s.BackendMemoryMax == nil || *s.BackendMemoryMax > 1<<30 || *s.BackendMemoryMax <= 0 || s.BrowserMemoryMax == nil || *s.BrowserMemoryMax > 1<<30 || *s.BrowserMemoryMax <= 0 || s.RuntimeMax == nil || *s.RuntimeMax > 180 || *s.RuntimeMax <= 0 || s.BrowserMax == nil || *s.BrowserMax > 60 || *s.BrowserMax <= 0 || s.Port == nil || *s.Port != 18548 {
		return b, m, "resource_bound"
	}
	u, e := url.Parse(in.Login.Target)
	if e != nil || u.Scheme != "http" || (u.Host != "127.0.0.1:19548" && u.Host != "127.0.0.1:18548") || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return b, m, "login_target_binding"
	}
	if self, e := os.Executable(); e != nil {
		return b, m, "executable_identity"
	} else if h, e := s2Hash(self); e != nil || h != in.ExecutableSHA {
		return b, m, "executable_identity"
	}
	if ns, e := os.Readlink("/proc/self/ns/net"); e != nil {
		return b, m, "network_namespace"
	} else if host, e := os.Readlink("/proc/1/ns/net"); e != nil || ns == host {
		return b, m, "network_namespace"
	}
	for _, key := range []string{"ONEBASE_DEBUG_TOKEN", "ONEBASE_CONTROL_TOKEN", "DATABASE_URL", "TEST_DATABASE_URL", "ONEBASE_MASTER_KEY", "ONEBASE_MASTER_KEY_FILE", "ONEBASE_STRICT_RLS", "ONEBASE_H2C"} {
		if os.Getenv(key) != "" {
			return b, m, "inherited_environment"
		}
	}
	return b, m, ""
}

// Includes every pre-created WORK file, not just named config/media inputs.
// No content reads, executable traversal, device/FIFO/symlink/hardlink allowed.
func s2WorkBound(root string) bool {
	var total int64
	count := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("work_kind")
		}
		if d.IsDir() {
			return nil
		}
		fi, e := d.Info()
		if e != nil {
			return e
		}
		st, ok := fi.Sys().(*syscall.Stat_t)
		if !fi.Mode().IsRegular() || !ok || st.Nlink != 1 || int(st.Uid) != os.Geteuid() {
			return errors.New("work_kind")
		}
		total += fi.Size()
		count++
		if total > 1<<30 || count > 2400 {
			return errors.New("work_cap")
		}
		return nil
	})
	return err == nil
}
func s2VerifyTree(root string, want map[string]string, limit int64) bool {
	var total int64
	count := 0
	ok := true
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type()&os.ModeSymlink != 0 {
			ok = false
			return errors.New("symlink")
		}
		if d.IsDir() {
			return nil
		}
		rel, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		rel = filepath.ToSlash(rel)
		pin, exists := want[rel]
		if !exists {
			ok = false
			return errors.New("extra_file")
		}
		fi, e := d.Info()
		if e != nil {
			return e
		}
		st, valid := fi.Sys().(*syscall.Stat_t)
		if !fi.Mode().IsRegular() || !valid || st.Nlink != 1 {
			ok = false
			return errors.New("file_kind")
		}
		total += fi.Size()
		if total > limit {
			ok = false
			return errors.New("size")
		}
		h, e := s2Hash(path)
		if e != nil || h != pin {
			ok = false
			return errors.New("identity")
		}
		count++
		return nil
	})
	return err == nil && ok && count == len(want)
}
func s2WritePrivate(path string, value any) error {
	p, e := json.MarshalIndent(value, "", "  ")
	if e != nil {
		return e
	}
	if len(p) > s2MaxFlags {
		return errors.New("flags_limit")
	}
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	_, e = f.Write(p)
	ce := f.Close()
	if e != nil {
		return e
	}
	return ce
}

func TestAudioDirtyStage2Private(t *testing.T) {
	if os.Getenv("ONEBASE_AUDIO_STAGE2_PRIVATE") != "1" {
		t.Skip("private stage2 off; no input opened")
	}
	// Suppress native uri/user_login/error logging BEFORE any project/API
	// construction. Never save raw logs; test output is closed enums only.
	oldSlog := slog.Default()
	oldLog := log.Writer()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	log.SetOutput(io.Discard)
	defer slog.SetDefault(oldSlog)
	defer log.SetOutput(oldLog)
	defer func() {
		if recover() != nil {
			t.Error("stage2_stop:test_panic")
		}
	}()
	var in s2Input
	if p := os.Getenv("ONEBASE_AUDIO_STAGE2_INPUT"); p == "" || s2JSON(p, &in) != nil {
		t.Fatal("stage2_stop:input_schema")
	}
	binding, m, code := s2Preflight(in)
	if code != "" {
		t.Fatal("stage2_stop:" + code)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*in.Supervisor.RuntimeMax)*time.Second)
	defer cancel()
	defer func() {
		_ = s2WritePrivate(filepath.Join(in.Work, "stop.request"), map[string]bool{"stop_owned_supervisor": true})
	}()
	p, e := project.Load(in.Copy.Project)
	if e != nil {
		t.Fatal("stage2_stop:project_load")
	}
	defer p.Close()
	a, e := project.LoadConfig(in.Copy.Project)
	if e != nil || a == nil {
		t.Fatal("stage2_stop:app_config")
	}
	if !s2Dependencies(a, in) {
		t.Fatal("stage2_stop:unsupported_native_dependency")
	}
	strict := a.DSL != nil && a.DSL.StrictLexicalScope
	if strict != *in.Native.StrictLexical {
		t.Fatal("stage2_stop:lexical_parity")
	}
	reg := s2Registry(p)
	entity := reg.GetEntity(binding.Entity)
	if entity == nil {
		t.Fatal("stage2_stop:entity_metadata")
	}
	// Do not start a backend with a guessed or reduced form. Selection still
	// happens natively for the authenticated request; this only checks presence.
	matchedForms := 0
	for _, form := range entity.Forms {
		if form.Name == binding.Form && form.Kind == "object" {
			matchedForms++
		}
	}
	if matchedForms != 1 {
		t.Fatal("stage2_stop:form_metadata")
	}
	db, e := storage.ConnectSQLite(ctx, in.Copy.DB)
	if e != nil {
		t.Fatal("stage2_stop:copy_open")
	}
	defer db.Close()
	db.SetFilesDir(in.Copy.Media)
	// No Migrate/EnsureSchema/SyncRoles/seed/reconcile/start/workers.
	roles, e := auth.LoadRolesYAML(filepath.Join(in.Copy.Project, "roles"))
	if e != nil {
		t.Fatal("stage2_stop:roles_metadata")
	}
	if in.Mode == "strict_true" {
		guarded := access.GuardedEntitiesFromRoles(roles)
		db.SetStrictRLSGuard(func(name string) bool { return guarded[name] })
	} else if in.Mode != "strict_false" {
		t.Fatal("stage2_stop:explicit_mode")
	}
	repo := auth.NewRepo(db)
	storedRoles, e := repo.GetRolesForUser(ctx, binding.ActorID)
	if e != nil {
		t.Fatal("stage2_stop:actor_roles")
	}
	if !s2Same(s2RoleNames(storedRoles), binding.Roles) {
		t.Fatal("stage2_stop:role_names_parity")
	}
	policies := map[string]auth.Permission{}
	for _, r := range storedRoles {
		policies[r.Name] = r.Permissions
	}
	if !s2Witness(in.Native.RolePolicies, policies) {
		t.Fatal("stage2_stop:effective_policies_parity")
	}
	byName := map[string]auth.Permission{}
	for _, r := range roles {
		byName[r.Name] = r.Permissions
	}
	for name, perm := range policies {
		if other, ok := byName[name]; !ok || !s2Same(perm, other) {
			t.Fatal("stage2_stop:roles_file_db_parity")
		}
	}
	forms, layouts, e := extform.New(db).LoadEnabledPrintForms(ctx)
	if e != nil {
		t.Fatal("stage2_stop:external_print_metadata")
	}
	reg.SetExternalPrintForms(forms)
	reg.SetExternalLayoutForms(layouts)
	reports, e := extform.NewReports(db).LoadEnabledReports(ctx)
	if e != nil {
		t.Fatal("stage2_stop:external_report_metadata")
	}
	reg.SetExternalReports(reports)
	procs, programs, e := extform.NewProcessors(db).LoadEnabled(ctx)
	if e != nil {
		t.Fatal("stage2_stop:external_processor_metadata")
	}
	reg.SetExternalProcessors(procs, programs)
	interp := interpreter.New()
	interp.LookupProc = reg.GetModuleProc
	interp.LookupSiblingProc = reg.GetSiblingProc
	interp.LookupModuleProc = reg.GetModuleNamespacedProc
	interp.StrictLexicalScope = strict
	uiCfg, e := s2UI(a, in)
	if e != nil {
		t.Fatal("stage2_stop:ui_mapping")
	}
	if !s2Witness(in.Native.UI, s2ScalarMap(uiCfg, map[string]bool{"Bundle": true, "Mailer": true, "Webhooks": true, "LoginLimit": true, "Metrics": true, "JobQueue": true})) {
		t.Fatal("stage2_stop:ui_mapping_parity")
	}
	qcfg := s2Queue(a)
	// Native configuration/capacity is not an assertion that workers run.
	// Exact frozen app has no queue block: requested default4 -> SQLite1.
	// Preserve its native config and constructor; never Run/Start/reconcile.
	if !s2Witness(in.Native.Queue, qcfg) || in.Native.QueueConfiguredWorkers == nil || in.Native.QueueEffectiveWorkers == nil || *in.Native.QueueConfiguredWorkers != qcfg.Workers || qcfg.Workers != 4 || *in.Native.QueueEffectiveWorkers != 1 || a.Queue != nil {
		t.Fatal("stage2_stop:queue_configured_mapping")
	}
	sched := scheduler.New(db, reg, interp)
	// Native prepareServerRuntime loads the job REGISTRY before app.Build.
	// LoadJobs/replaceJobs on a NEW non-running scheduler only build cron in
	// memory; never call Start/Run/RunNow/queue.Start. Dormant LoadJobs was
	// explicitly accepted in review; empty/nil is not a parity substitute.
	if e = sched.LoadJobs(p.ScheduledJobs); e != nil {
		t.Fatal("stage2_stop:dormant_jobs_metadata")
	}
	defer func() {
		sc, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = sched.Shutdown(sc)
	}()
	queue := jobqueue.New(db, sched, qcfg)
	if queue.Workers() != *in.Native.QueueEffectiveWorkers || !queue.Degraded() || queue.InFlight() != 0 {
		t.Fatal("stage2_stop:queue_effective_mapping")
	}
	uiCfg.JobQueue = queue
	limiter := auth.NewLoginLimiter(5, time.Minute)
	uiCfg.LoginLimit = limiter
	front := New(reg, db, interp, repo, uiCfg, sched)
	defer func() {
		sc, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = front.Shutdown(sc)
	}()
	sched.SetVarsBuilder(front.BuildJobDSLVars) // dormant; never Start/RunNow
	apiCfg := api.Config{AppName: uiCfg.AppName, MaxFileSizeMB: uiCfg.MaxFileSizeMB, AllowedTypes: uiCfg.AllowedTypes, LoginLimit: limiter}
	if !s2Witness(in.Native.API, s2ScalarMap(apiCfg, map[string]bool{"Webhooks": true, "LoginLimit": true, "Metrics": true})) {
		t.Fatal("stage2_stop:api_mapping_parity")
	}
	capture := &s2Capture{records: map[int]*s2Record{}, front: front, binding: binding, input: in, entity: entity, stop: make(chan struct{})}
	interp.DebugSource = func() interpreter.DebugHook { return &s2Observer{c: capture} }
	app := api.New(reg, db, interp, repo, "127.0.0.1", *in.Supervisor.Port, apiCfg, front, sched)
	listener, e := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(*in.Supervisor.Port))
	if e != nil {
		t.Fatal("stage2_stop:own_port")
	}
	defer listener.Close()
	server := &http.Server{Handler: capture.wrap(app.Handler()), ReadHeaderTimeout: 5 * time.Second}
	defer func() {
		front.CloseEventStreams()
		sc, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = server.Shutdown(sc)
	}()
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	if e = s2WritePrivate(filepath.Join(in.Work, "ready.json"), map[string]any{"ready": true, "mode": in.Mode, "experiment_scope": in.ExperimentScope, "original_parity": "UNKNOWN", "login_target_bound": true}); e != nil {
		t.Fatal("stage2_stop:ready_receipt")
	}
	// First facts seal ONLY the observer. Keep native backend alive until the
	// owner closes the authenticated context/browser, including media followup.
	// No HTTP polling/debug route or browser launcher in this test.
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	facts := capture.stop
	browserComplete := false
WAIT:
	for {
		select {
		case <-facts:
			facts = nil
		case <-ticker.C:
			path := filepath.Join(in.Work, "browser.done")
			if _, se := os.Lstat(path); os.IsNotExist(se) {
				continue
			} else if se != nil {
				capture.gapStop("browser_receipt_unavailable")
				break WAIT
			}
			var done s2BrowserDone
			if s2JSON(path, &done) != nil || done.Schema != "onebase.audio.stage2.browser-done/v1" || done.Mode != in.Mode || done.CaptureSHA != in.Login.CaptureSHA || done.Stage != "CAPTURE_COMPLETED" || !s2True(done.ContextClosed) || !s2True(done.BrowserClosed) {
				capture.gapStop("browser_cleanup_or_capture_gap")
			} else {
				browserComplete = true
			}
			break WAIT
		case <-ctx.Done():
			capture.gapStop("runtime_bound")
			break WAIT
		case <-served:
			capture.gapStop("backend_ended")
			break WAIT
		}
	}
	front.CloseEventStreams()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if e = server.Shutdown(shutdownCtx); e != nil {
		capture.gapStop("http_cleanup_gap")
		_ = server.Close()
	}
	shutdownCancel()
	// Native browser cleanup belongs to reviewed coordinator. This test never
	// reads credentials or launches/imports the old harness or Chrome.
	capture.mu.Lock()
	flags := append([]s2Flags(nil), capture.completed...)
	globalGap := capture.globalGap
	pending := len(capture.records)
	capture.mu.Unlock()
	status := "NOT_REPRODUCED"
	found := false
	if globalGap != "" || pending != 0 || !browserComplete {
		status = "OBSERVATION_GAP"
	}
	for _, f := range flags {
		if f.Gap != "" {
			status = "OBSERVATION_GAP"
		}
		if f.Event == "Воспроизведение" && f.Element == binding.Player {
			found = true
			if f.ObservedDirty != nil && *f.ObservedDirty && status != "OBSERVATION_GAP" {
				status = "ACTUAL_RED_OBSERVED"
			}
		}
	}
	if !found && status != "OBSERVATION_GAP" {
		status = "OBSERVATION_GAP"
	}
	unchanged := s2VerifyTree(in.Copy.Project, m.Files, 256<<20) && s2VerifyTree(s2SourceRoot+"/config-after-b10", m.Files, 256<<20)
	if h, e := s2Hash(in.Copy.SourceDB); e != nil || h != in.Copy.SourceDBSHA {
		unchanged = false
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, e := os.Lstat(in.Copy.SourceDB + suffix); !os.IsNotExist(e) {
			unchanged = false
		}
	}
	if !unchanged {
		status = "OBSERVATION_GAP"
	}
	if queue.InFlight() != 0 || sched.ActiveRunCount() != 0 {
		status = "OBSERVATION_GAP"
		globalGap = "unexpected_active_job"
	}
	result := map[string]any{"schema": "onebase.audio.stage2.flags/v1", "status": status, "experiment_scope": in.ExperimentScope, "backend_kernel_net_deny": *in.Supervisor.BackendKernelNetDeny, "browser_network_boundary": in.Supervisor.BrowserNetworkBoundary, "browser_os_network_isolated": *in.Supervisor.BrowserOSNetworkIsolated, "browser_routing_cache_effect": in.Supervisor.BrowserRoutingCacheEffect, "browser_job_bound": *in.Supervisor.BrowserJobBound, "boundary_evidence": "reviewed_owner_supervisor_provenance", "explicit_mode": in.Mode, "original_strict_rls_parity": "UNKNOWN", "queue_configured_workers": qcfg.Workers, "queue_effective_workers": queue.Workers(), "pool_run_called": false, "records": flags, "global_observation_gap": globalGap, "browser_cleanup_confirmed": browserComplete, "pending_requests": pending, "source_unchanged": unchanged, "sdk_changed": false, "cause_claimed": false, "next": "STOP_REVIEW_NO_AUTOFIX"}
	if e = s2WritePrivate(filepath.Join(in.Work, "flags.json"), result); e != nil {
		t.Error("stage2_stop:flags_output")
	}
	t.Log("stage2_facts:" + status + ";originalParity=UNKNOWN;no_cause_claim")
}

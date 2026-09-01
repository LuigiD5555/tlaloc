package tlaloque

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifest(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadSwarmManifestBuildsBothTransports(t *testing.T) {
	path := writeManifest(t, `{
      "workers": [
        {
          "descriptor": {"id":"intent-http","capability":"detect_intent","scope":"general","engine":"MODEL","input_schema":"text","output_schema":"intent","parameter_count":12000000,"max_concurrency":4},
          "transport": "HTTP_JSON",
          "endpoint": "http://127.0.0.1:9101/infer",
          "timeout_ms": 5000
        },
        {
          "descriptor": {"id":"router","capability":"route","scope":"general","engine":"DETERMINISTIC","input_schema":"context","output_schema":"route","deterministic":true,"dependencies":["DETECT_INTENT"]},
          "transport": "PROCESS",
          "command": ["./workers/router"]
        }
      ],
      "plan": {
        "id": "micro-router",
        "max_parallel": 2,
        "nodes": [
          {"id":"intent","capability":"detect_intent"},
          {"id":"route","capability":"route","depends_on":["intent"]}
        ]
      }
    }`)

	manifest, err := LoadSwarmManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Schema != SwarmManifestSchemaR0 {
		t.Fatalf("schema=%s", manifest.Schema)
	}
	if manifest.Workers[0].Descriptor.Capability != "DETECT_INTENT" {
		t.Fatalf("descriptor not normalised: %+v", manifest.Workers[0].Descriptor)
	}
	if manifest.Plan.Nodes[0].Capability != "DETECT_INTENT" {
		t.Fatalf("plan not normalised: %+v", manifest.Plan.Nodes[0])
	}

	registry, err := manifest.Registry()
	if err != nil {
		t.Fatal(err)
	}
	descriptors := registry.Descriptors()
	if len(descriptors) != 2 {
		t.Fatalf("registered=%d", len(descriptors))
	}

	httpWorker, ok := registry.Get("intent-http")
	if !ok {
		t.Fatal("http worker not registered")
	}
	if _, isHTTP := httpWorker.(HTTPWorker); !isHTTP {
		t.Fatalf("intent-http built as %T, want HTTPWorker", httpWorker)
	}
	processWorker, ok := registry.Get("router")
	if !ok {
		t.Fatal("process worker not registered")
	}
	if _, isProcess := processWorker.(ProcessWorker); !isProcess {
		t.Fatalf("router built as %T, want ProcessWorker", processWorker)
	}
}

// An endpoint alone implies HTTP_JSON; nothing else does.
func TestLoadSwarmManifestInfersTransport(t *testing.T) {
	endpointOnly := writeManifest(t, `{"workers":[{"descriptor":{"id":"a","capability":"A","input_schema":"in","output_schema":"out"},"endpoint":"http://127.0.0.1:9000/infer"}]}`)
	manifest, err := LoadSwarmManifest(endpointOnly)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Workers[0].Transport != TransportHTTPJSON {
		t.Fatalf("transport=%s, want HTTP_JSON", manifest.Workers[0].Transport)
	}

	commandOnly := writeManifest(t, `{"workers":[{"descriptor":{"id":"a","capability":"A","input_schema":"in","output_schema":"out"},"command":["./worker"]}]}`)
	manifest, err = LoadSwarmManifest(commandOnly)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Workers[0].Transport != TransportProcess {
		t.Fatalf("transport=%s, want PROCESS", manifest.Workers[0].Transport)
	}
}

func TestLoadSwarmManifestRejectsMalformedCatalogs(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "unexpected schema",
			body: `{"schema":"tlaloc.other.r1","workers":[{"descriptor":{"id":"a","capability":"A","input_schema":"in","output_schema":"out"},"command":["./x"]}]}`,
			want: "unexpected swarm manifest schema",
		},
		{
			name: "no workers",
			body: `{"workers":[]}`,
			want: "requires workers",
		},
		{
			name: "process without command",
			body: `{"workers":[{"descriptor":{"id":"a","capability":"A","input_schema":"in","output_schema":"out"},"transport":"PROCESS"}]}`,
			want: "command is required",
		},
		{
			name: "http without endpoint",
			body: `{"workers":[{"descriptor":{"id":"a","capability":"A","input_schema":"in","output_schema":"out"},"transport":"HTTP_JSON"}]}`,
			want: "endpoint is required",
		},
		{
			name: "unsupported transport",
			body: `{"workers":[{"descriptor":{"id":"a","capability":"A","input_schema":"in","output_schema":"out"},"transport":"GRPC","command":["./x"]}]}`,
			want: "unsupported transport",
		},
		{
			name: "invalid descriptor",
			body: `{"workers":[{"descriptor":{"id":"a"},"command":["./x"]}]}`,
			want: "worker[0]",
		},
		{
			name: "unknown field",
			body: `{"workers":[{"descriptor":{"id":"a","capability":"A","input_schema":"in","output_schema":"out"},"command":["./x"],"replicas":4}]}`,
			want: "unknown field",
		},
		{
			name: "invalid plan",
			body: `{"workers":[{"descriptor":{"id":"a","capability":"A","input_schema":"in","output_schema":"out"},"command":["./x"]}],"plan":{"id":"p","nodes":[{"id":"n","capability":"A","depends_on":["ghost"]}]}}`,
			want: "unknown node",
		},
		{
			name: "malformed json",
			body: `{"workers":`,
			want: "swarm manifest",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := LoadSwarmManifest(writeManifest(t, testCase.body))
			if err == nil {
				t.Fatalf("expected rejection for %s", testCase.name)
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error %q does not mention %q", err, testCase.want)
			}
		})
	}
}

func TestLoadSwarmManifestReportsMissingFile(t *testing.T) {
	if _, err := LoadSwarmManifest(filepath.Join(t.TempDir(), "absent.json")); err == nil {
		t.Fatal("expected an error for a missing manifest")
	}
}

// Two workers claiming the same id must be refused: worker identity is what
// the report attributes latency and errors to.
func TestManifestRegistryRefusesDuplicateWorkerIDs(t *testing.T) {
	path := writeManifest(t, `{"workers":[
      {"descriptor":{"id":"twin","capability":"A","input_schema":"in","output_schema":"out"},"command":["./a"]},
      {"descriptor":{"id":"twin","capability":"B","input_schema":"in","output_schema":"out"},"command":["./b"]}
    ]}`)
	manifest, err := LoadSwarmManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manifest.Registry(); err == nil {
		t.Fatal("expected duplicate worker ids to be refused")
	}
}

// A manifest may ship a catalog with no plan; planning then comes from --goal.
func TestLoadSwarmManifestAllowsCatalogWithoutPlan(t *testing.T) {
	path := writeManifest(t, `{"workers":[{"descriptor":{"id":"a","capability":"A","input_schema":"in","output_schema":"out"},"command":["./x"]}]}`)
	manifest, err := LoadSwarmManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Plan.Nodes) != 0 {
		t.Fatalf("plan=%+v, want empty", manifest.Plan)
	}
}

package tlaloque

import "testing"

func TestNormalizedTransportPreservesR0Defaults(t *testing.T) {
	cases := []struct {
		name string
		spec WorkerSpec
		want string
	}{
		{name: "implicit process", spec: WorkerSpec{Command: []string{"worker"}}, want: TransportProcess},
		{name: "implicit http", spec: WorkerSpec{Endpoint: "http://127.0.0.1:9000/infer"}, want: TransportHTTPJSON},
		{name: "explicit process", spec: WorkerSpec{Transport: " process ", Command: []string{"worker"}}, want: TransportProcess},
		{name: "explicit http", spec: WorkerSpec{Transport: "http_json", Endpoint: "http://127.0.0.1:9000/infer"}, want: TransportHTTPJSON},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizedTransport(tc.spec); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestTransportStrategyRegistryBuildsExpectedAdapter(t *testing.T) {
	desc := CapabilityDescriptor{ID: "tiny-bert", Capability: "CLASSIFY", Scope: ScopeGeneral, Engine: EngineModel, InputSchema: "text", OutputSchema: "label"}
	processSpec := WorkerSpec{Descriptor: desc, Command: []string{"python", "worker.py"}}
	processStrategy, err := resolveTransportStrategy(normalizedTransport(processSpec))
	if err != nil {
		t.Fatal(err)
	}
	if err := processStrategy.Validate(processSpec, desc); err != nil {
		t.Fatal(err)
	}
	if _, ok := processStrategy.Build(processSpec, 0).(ProcessWorker); !ok {
		t.Fatal("expected ProcessWorker adapter")
	}

	httpSpec := WorkerSpec{Descriptor: desc, Endpoint: "http://127.0.0.1:9000/infer"}
	httpStrategy, err := resolveTransportStrategy(normalizedTransport(httpSpec))
	if err != nil {
		t.Fatal(err)
	}
	if err := httpStrategy.Validate(httpSpec, desc); err != nil {
		t.Fatal(err)
	}
	if _, ok := httpStrategy.Build(httpSpec, 0).(HTTPWorker); !ok {
		t.Fatal("expected HTTPWorker adapter")
	}
}

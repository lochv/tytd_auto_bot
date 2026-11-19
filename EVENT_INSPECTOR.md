# Event Inspector (read-only)

The inspector attaches to an already running Microsoft Edge instance through the local CDP endpoint, enables `Network`/`Page` observation, records WebSocket frames, redacts likely credentials, performs a schema-free protobuf wire decode when possible, and writes a JSON timeline. It does not call `Runtime.evaluate`, send game messages, or expose a game command API.

## Windows

Start Edge with a dedicated profile and local debugging port (close the existing Edge instance first if it reuses the profile):

```powershell
msedge.exe --remote-debugging-port=9222 --user-data-dir="$env:TEMP\tytd-edge-inspector"
```

Build and run from the repository root:

```powershell
go build -o .\build\event-inspector.exe .\cmd\event-inspector
.\build\event-inspector.exe -edge-cdp 127.0.0.1:9222 -duration 2m -out .\event-inspector.json
```

Open the game page in that Edge window, reproduce the event, then stop with Ctrl+C. Treat the JSON as sensitive even after redaction; the output is created with owner-only permissions where supported.

## Limitations

The protobuf decoder is schema-free and reports wire field numbers. To obtain named fields, add a descriptor mapping for the relevant generated message in `internal/proto`. Activity classification is intentionally conservative; future event types remain visible in the timeline and can be promoted to named types after a fixture is reviewed.

## Go packet analyzer

`Player` and `EventPlayer` expose an optional `Inspector *eventinspector.Analyzer`. When attached by an offline diagnostic/test harness, both websocket readers and writers record packet metadata. The analyzer recognizes 21001, 21036 and 21003, links a 21003 response to its request by `activityId`, and records JSON state paths that changed. Reports contain lengths and SHA-256 fingerprints, not raw packet bytes or credentials.

Generate a deliberately gated module scaffold only after reviewing an offline fixture:

```powershell
go run .\cmd\event-module-skeleton -name new_event -verified -out .\player_event_new_event.go
```

Without `-verified`, generation fails. The scaffold is inert and does not contain a sender or packet encoder.

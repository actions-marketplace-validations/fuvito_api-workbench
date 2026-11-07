# API Development Workbench
## Product Requirements, Architecture & Implementation Roadmap

**Status:** Proposed engineering baseline  
**Product type:** Open-source desktop API development and testing platform  
**Primary language:** Go  
**Frontend:** React + TypeScript  
**Desktop:** Wails  
**Storage:** Local filesystem + SQLite  
**Project formats:** YAML and JSON  
**API specification:** OpenAPI 3.0 / 3.1

---

# 1. Product Vision

Build an open-source, local-first API development and testing workbench that combines:

- Interactive API development.
- Request collections and environments.
- Automated scenario testing.
- OpenAPI specification integration.
- Contract and schema validation.
- CLI execution and CI/CD automation.

The platform should allow developers to move naturally from manually testing an endpoint to creating automated regression tests.

## Product principles

1. Local-first: no mandatory cloud account or hosted backend.
2. Git-native: project definitions are ordinary files.
3. CLI and GUI parity: both use the same execution engine.
4. Open standards: support OpenAPI, JSON Schema, YAML and JSON.
5. Extensible architecture: additional protocols and integrations can be added.
6. Community-driven: core functionality remains open source.

# 2. Target Users

Primary:
- Backend developers.
- Full-stack developers.
- QA automation engineers.

Secondary:
- DevOps engineers.
- API architects.
- Small engineering teams.

Initial API framework examples:
- Spring Boot.
- FastAPI.
- Express / Node.js.
- ASP.NET Core.

# 3. Functional Requirements

## 3.1 Workspace Management

**WS-001:** Create a new workspace.

**WS-002:** Open an existing workspace.

**WS-003:** Display collections, scenarios, environments and contracts in a project explorer.

**WS-004:** Create folders and organize API requests.

**WS-005:** Save workspace definitions to the filesystem.

**WS-006:** Support YAML and JSON project formats.

**WS-007:** Detect external file changes and reload or reconcile them safely.

## 3.2 API Client

**API-001:** Support GET, POST, PUT, PATCH, DELETE, HEAD and OPTIONS.

**API-002:** Support query parameters.

**API-003:** Support request headers.

**API-004:** Support JSON, raw text, form URL-encoded and multipart form-data bodies.

**API-005:** Support Bearer, Basic and API Key authentication.

**API-006:** Display HTTP status, response headers, response body and execution duration.

**API-007:** Pretty-print JSON responses.

**API-008:** Provide syntax highlighting and response search.

**API-009:** Support request cancellation and configurable timeouts.

**API-010:** Maintain request history.

**API-011:** Allow users to save a manually executed request as a collection item.

**API-012:** Support importing cURL commands.

## 3.3 Collections

**COL-001:** Create collections and folders.

**COL-002:** Save individual requests.

**COL-003:** Execute individual requests.

**COL-004:** Execute an entire collection.

**COL-005:** Support collection-level variables and authentication.

**COL-006:** Support collection-level default headers.

**COL-007:** Allow requests to inherit collection settings.

## 3.4 Environment Management

**ENV-001:** Create multiple environments.

**ENV-002:** Support local, development, staging and production environments.

**ENV-003:** Support environment variables.

**ENV-004:** Support global and collection-level variables.

**ENV-005:** Support environment-variable interpolation.

**ENV-006:** Support secret references through environment variables and the operating system credential store.

**ENV-007:** Mask secret values in UI and reports.

**ENV-008:** Allow switching environments without modifying request definitions.

## 3.5 Scenario Engine

**SCN-001:** Define scenarios using YAML or JSON.

**SCN-002:** Execute steps sequentially.

**SCN-003:** Reference requests from collections.

**SCN-004:** Extract response values into runtime variables.

**SCN-005:** Reuse extracted variables in subsequent requests.

**SCN-006:** Support status, header and JSON-body assertions.

**SCN-007:** Support conditional execution.

**SCN-008:** Support failure policies: stop, continue and skip.

**SCN-009:** Support configurable retry policies.

**SCN-010:** Provide detailed execution reports.

**SCN-011:** Support data-driven execution using multiple input datasets.

Initial scope should prioritize sequential execution. Parallel scenarios can be introduced later.

## 3.6 OpenAPI Integration

**OAI-001:** Import OpenAPI 3.0 and 3.1 documents.

**OAI-002:** Support YAML and JSON specifications.

**OAI-003:** Parse paths, HTTP methods, parameters, headers and request bodies.

**OAI-004:** Generate request collections from imported specifications.

**OAI-005:** Generate sample request bodies.

**OAI-006:** Extract response schemas.

**OAI-007:** Validate HTTP responses against declared response schemas.

**OAI-008:** Support authentication definitions in OpenAPI.

**OAI-009:** Display the relationship between generated requests and their source specification.

**OAI-010:** Detect specification changes when reimporting.

**OAI-011:** Provide basic breaking-change analysis.

OpenAPI import must not overwrite manually customized requests without user approval.

## 3.7 Reporting

**RPT-001:** Display individual request execution results.

**RPT-002:** Display scenario execution summaries.

**RPT-003:** Display failed assertions.

**RPT-004:** Capture execution duration.

**RPT-005:** Export JSON reports.

**RPT-006:** Export JUnit XML reports.

**RPT-007:** Return nonzero CLI exit codes for failed tests.

**RPT-008:** Redact secrets from reports.

# 4. Architecture

## 4.1 Main components

### Desktop frontend
- React
- TypeScript
- Vite
- Tailwind CSS
- shadcn/ui

Responsibilities:
- Request editor.
- Collection explorer.
- Environment selector.
- Response viewer.
- Scenario editor.
- OpenAPI management.
- Execution results.

### Desktop application
- Wails
- Go application bindings.

Responsibilities:
- Application lifecycle.
- Native file dialogs.
- Workspace selection.
- Secure local integration.
- Communication with the Go core.

### Go core

The Go core must be independent of Wails.

Suggested packages:

- domain
- workspace
- serialization
- request
- executor
- scenario
- variables
- assertions
- environment
- authentication
- openapi
- reporting
- history

### CLI

Use Cobra.

Commands:

- init
- request
- collection
- scenario
- import
- validate
- version

The CLI must use the same application services and execution engine as the desktop application.

## 4.2 Dependency direction

Frontend → Wails bindings → Application services → Domain and engine packages.

CLI → Application services → Domain and engine packages.

The domain and execution packages must not depend on Wails, React, or Cobra.

## 4.3 Data storage

Project definitions:
- YAML or JSON files.

Execution history:
- Local SQLite database.

Credentials:
- Operating system credential store where supported.
- Environment-variable references for CI/CD.

No mandatory remote database.

# 5. Project Structure

Initial proposed project layout:

```text
project/
├── cmd/
│   └── clrsig/
│       └── main.go
│
├── internal/
│   ├── domain/
│   ├── application/
│   ├── workspace/
│   ├── serialization/
│   ├── request/
│   ├── executor/
│   ├── scenario/
│   ├── variables/
│   ├── assertions/
│   ├── environment/
│   ├── authentication/
│   ├── openapi/
│   ├── reporting/
│   └── history/
│
├── pkg/
│   └── core/
│
├── cli/
│
├── desktop/
│   ├── frontend/
│   └── wails/
│
├── examples/
│
├── tests/
│
├── docs/
│
├── go.mod
└── README.md
```

The exact module boundaries can be adjusted after the first working vertical slice.

# 6. Project File Format

The application should maintain one canonical domain model.

YAML and JSON are serialization formats, not separate application models.

Example:

```yaml
schemaVersion: "1"

name: Customer API

request:
  method: GET
  url: "{{baseUrl}}/customers/{{customerId}}"

  headers:
    Accept: application/json

assertions:
  - type: status
    expected: 200

  - type: jsonpath
    path: "$.id"
    expected: 123
```

Equivalent JSON must be supported.

Requirements:
- Version every persisted project definition.
- Validate files before execution.
- Support format conversion.
- Preserve unknown supported extension fields where possible.
- Define migration rules for future schema versions.
- Use atomic writes when saving files.
- Avoid unnecessary formatting changes to existing files.

# 7. MVP Scope

The first public MVP must contain:

1. Desktop application.
2. Go execution engine.
3. Interactive HTTP request editor.
4. Request collections.
5. YAML and JSON support.
6. Environment management.
7. Basic authentication.
8. Response viewer.
9. Request history.
10. Sequential scenario execution.
11. Variable extraction.
12. Assertions.
13. OpenAPI import.
14. Response schema validation.
15. CLI execution.
16. JSON and JUnit reports.
17. Documentation and examples.

Explicitly excluded from MVP:
- Cloud accounts.
- Team synchronization.
- Hosted collaboration.
- GraphQL.
- WebSockets.
- gRPC.
- Load testing.
- Mock servers.
- Plugin marketplace.
- AI-generated tests.

# 8. Implementation Roadmap

## Phase 0 – Engineering Foundation

Deliverables:
- Repository structure.
- Go module.
- React and TypeScript application.
- Wails integration.
- CI/CD.
- Architecture documentation.
- Initial domain models.

Acceptance criteria:
- Desktop application launches.
- Go core can be invoked from the UI.
- Unit tests execute in CI.

## Phase 1 – HTTP Execution Engine

Deliverables:
- HTTP executor.
- Request and response models.
- Header and query parameter handling.
- Request body support.
- Timeouts and cancellation.
- Error handling.
- Authentication basics.

Acceptance criteria:
- Execute all supported HTTP methods.
- Correctly handle JSON responses.
- Display response status and duration.
- Test against a local mock HTTP server.

## Phase 2 – Workspace and API Client

Deliverables:
- Workspace management.
- Collection explorer.
- Request editor.
- Response viewer.
- YAML and JSON serialization.
- Environment management.
- Request history.

Acceptance criteria:
- Create and save collections.
- Reopen existing projects.
- Switch environments.
- Execute saved requests.
- Preserve data across application restarts.

## Phase 3 – Scenario Engine and CLI

Deliverables:
- Scenario definitions.
- Sequential execution.
- Runtime variable extraction.
- Assertions.
- Failure handling.
- Cobra CLI.
- JSON and JUnit reports.

Acceptance criteria:
- Execute a complete customer/order workflow.
- Reuse values extracted from earlier requests.
- Fail scenarios when assertions fail.
- Run the same scenario through GUI and CLI with consistent results.

## Phase 4 – OpenAPI Integration

Deliverables:
- OpenAPI 3.0/3.1 parsing.
- Collection generation.
- Request generation.
- Response schema validation.
- Specification reimport.
- Initial breaking-change detection.

Acceptance criteria:
- Import an existing OpenAPI document.
- Generate usable request collections.
- Execute generated requests.
- Validate responses against declared schemas.
- Detect representative breaking changes.

## Phase 5 – Public MVP

Deliverables:
- Cross-platform packaging.
- Windows, macOS and Linux release builds.
- Installation instructions.
- Example projects.
- GitHub Actions examples.
- User documentation.
- Contribution guidelines.
- Issue templates.

Acceptance criteria:
- New user can install and execute an API request within five minutes.
- New user can execute a multi-step scenario within ten minutes.
- Same project works on supported operating systems.
- CI/CD execution works without a graphical desktop session.

# 9. Quality Attributes

Performance:
- Low startup overhead.
- Efficient HTTP execution.
- Bounded response-body memory usage.
- Responsive UI during long-running scenarios.

Reliability:
- Cancellation support.
- Predictable error handling.
- Atomic project file writes.
- Stable execution behavior.

Security:
- Secret redaction.
- Safe credential storage.
- Restricted filesystem access.
- No unauthenticated local control server.
- Production environment warnings.

Portability:
- Windows.
- macOS.
- Linux.

Maintainability:
- Modular Go packages.
- Strong unit-test coverage for core execution.
- Integration tests using local HTTP servers.
- Stable serialization contracts.

# 10. Project Success Metrics

Initial community validation targets:

- 100 GitHub stars.
- 20 developers trying the application.
- 5 developers using it in real projects.
- 3 external contributors.
- Documented feedback from real API testing workflows.

Recurring usage and meaningful contributions matter more than GitHub stars.

# 11. Engineering Principles

1. Keep the Go core independent of the desktop framework.
2. Share execution logic between CLI and GUI.
3. Keep project definitions file-based.
4. Treat YAML and JSON as equal serialization formats.
5. Use OpenAPI as a first-class integration.
6. Avoid introducing remote infrastructure without demonstrated demand.
7. Build vertical slices rather than disconnected modules.
8. Test with real local HTTP servers.
9. Document public file formats.
10. Maintain backward compatibility for released project formats.

---

**Definition of MVP complete:** A developer can import an OpenAPI specification, explore and execute generated requests in the desktop application, create a multi-step scenario, save it as YAML or JSON, and execute the same scenario through the CLI and GitHub Actions.

We now have enough decisions to establish a concrete technical baseline. I would move from product exploration into engineering planning.

There are three important architectural decisions:
1. Desktop-first: Build a native desktop application using Wails, Go, and React.
2. Dual-format project files: Support YAML and JSON through one canonical internal data model.
3. OpenAPI in MVP: Make OpenAPI import and schema validation part of the initial release, not a future enhancement.
   I would also retain the CLI as a first-class component, even though desktop is the primary delivery method. This is important because automated execution is one of our strongest differentiators.
1. Final product architecture
   Product architecture
   Proposed baseline
   Desktop application
   React UI

TypeScript, Vite, Tailwind, shadcn/ui

Wails

Native window and Go bindings




Shared Go application core
Request engine
Scenario engine
Assertion engine
OpenAPI engine
Variable manager
Authentication
Environment manager
Report generator



Storage and project formats
YAML
JSON
OpenAPI
Local history

Filesystem-based, local-first, Git-compatible


CLI application

Same Go core, independent executable, CI/CD integration and machine-readable reports.


Critical architecture rule: The GUI and CLI must never implement their own HTTP execution, scenario evaluation, or assertion logic. They both call the same Go packages.
2. Technical decisions to lock
   Area	Decision
   Language	Go
   Desktop framework	Wails
   Frontend	React + TypeScript + Vite
   UI	Tailwind CSS + shadcn/ui
   CLI	Cobra
   Core HTTP	net/http
   Configuration	YAML + JSON
   OpenAPI	3.0 and 3.1
   JSON Schema	Draft 2020-12 support target
   Project storage	Local filesystem
   Local history	SQLite
   Unit testing	Go testing
   Frontend testing	Vitest + React Testing Library
   E2E testing	Playwright
   CI/CD	GitHub Actions
   Distribution	GitHub Releases

A few architectural refinements are important.
YAML and JSON strategy
I recommend supporting both formats natively, but maintaining a single internal representation.

For example:
request.yaml ──┐
├──> Domain Model ──> Execution Engine
request.json ──┘


The application should not have separate YAML and JSON execution implementations.

My preference:
- YAML is the default for newly created project files.
- JSON is supported for import, export, and direct editing.
- Users can choose their preferred format per project.
- The GUI operates on the canonical domain model.
- Serialization preserves supported fields and schema versions.
  Why SQLite?
  Collections, environments, and scenarios belong in ordinary files. However, request history, execution history, and recent project metadata are better suited to a local database.

SQLite is appropriate because it requires no server and has a small operational footprint.
Desktop security
Because the application executes arbitrary HTTP requests, security needs to be part of the architecture:
- Restrict Wails communication to trusted local application code.
- Never expose an unauthenticated local HTTP control server unnecessarily.
- Protect secrets using the operating system's credential store where available.
- Redact credentials in request history and reports.
- Restrict file operations to the selected project and explicitly authorized locations.
- Provide warnings for destructive requests against production environments.
3. Formal product requirements
   I would organize the product around six functional modules.
   Module	Responsibility
   Workspace	Projects, collections, folders and files
   API Client	Build and execute individual requests
   Scenario Runner	Execute multi-step workflows
   OpenAPI	Import specifications and validate requests/responses
   Environment	Variables, credentials and authentication
   Reporting	Execution results and CI/CD reports

The detailed requirements should distinguish MVP requirements from future functionality.
4. Engineering and product specification
   I would establish the following as the initial project specification. It is intentionally implementation-oriented so it can become the foundation for Claude Code development.
   (refer to Product Requirements, Architecture & Implementation Roadmap.md)
5. One important implementation recommendation
      I would not build all the modules independently and then integrate them.

Instead, build a complete vertical slice:
First end-to-end milestone
1. Launch the Wails desktop application.
2. Open a project directory.
3. Create a GET request.
4. Execute it through the Go engine.
5. Display the response in React.
6. Save the request as YAML.
7. Load the same request through the CLI.
   First demonstrable product milestone

This proves the most important architectural assumption before we invest in scenario editing, OpenAPI, and advanced UI.
6. Recommended initial repository organization
   I would use a monorepo, but keep the Go core as a separately testable module.
   clrsig/
   │
   ├── apps/
   │   ├── desktop/
   │   │   ├── frontend/
   │   │   └── wails/
   │   │
   │   └── cli/
   │
   ├── packages/
   │   └── core/
   │       ├── domain/
   │       ├── application/
   │       ├── executor/
   │       ├── scenario/
   │       ├── openapi/
   │       └── reporting/
   │
   ├── examples/
   ├── docs/
   ├── scripts/
   │
   ├── go.work
   ├── package.json
   ├── pnpm-workspace.yaml
   └── README.md


One refinement: I would keep the core as an independently compilable Go module, and use a Go workspace for local development. This avoids coupling the core to the desktop application's lifecycle.
7. The first implementation sprint
   I suggest we begin with five tasks:
   Task	Deliverable
   1	Initialize monorepo and Go workspace
   2	Create Wails + React application
   3	Define request/response domain models
   4	Implement HTTP executor
   5	Connect React request editor to Go executor

At the end of this sprint, we should have a working desktop API client—not just a collection of empty modules.
My final architectural advice
There is one thing I want to emphasize: the Go engine is the actual product foundation; the desktop application is its most visible interface.

If we get the engine right, we can eventually support additional interfaces, protocols, integrations, and automation workflows without rewriting the core.

I would make that the guiding principle for every engineering decision from this point forward.

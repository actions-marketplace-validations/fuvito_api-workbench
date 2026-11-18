# Please disregard this document, 2026-11-18-scenarios-screen-ux.md was built based on this document, and this was an early draft

## Scenarios Screen UX Report

### Executive summary

Your instinct is right: **the Scenarios screen is trying to be too many products at once**.

Right now it is functioning as:

1. A scenario editor.
2. A step editor.
3. A request builder.
4. A run console.
5. A result viewer.
6. A request/response inspector.
7. A variable manager.
8. A coverage/status summary.
9. A YAML/source escape hatch.

Each of those is useful, but when they all compete for the same screen, the user loses a clear mental model. The current design has a strong advantage — **edit and run are close together** — but it has crossed the line from “integrated workflow” into “crowded workspace”.

My honest recommendation: **do not split editing and running into completely separate pages**, but **introduce modes/perspectives inside the Scenarios screen** so the user can focus on one primary task at a time.

The best direction is:

> Keep one Scenarios screen, but reorganize it around **Build**, **Run**, and **Inspect** modes.

---

# 1. Current screen diagnosis

## What works well

### 1. Editing and running in one place is valuable

You were right to like this. For scenario tools, switching between an editor and a runner can become annoying because the workflow is naturally iterative:

- edit a request,
- run it,
- see failure,
- adjust assertion/header/body,
- run again,
- inspect response,
- repeat.

Keeping the run action close to the selected step is a good product decision.

### 2. Step list + selected step detail is a strong base pattern

The left-side step list and right-side selected-step detail is a familiar and effective pattern. It gives the user a sense of the scenario flow while letting them focus on one step.

### 3. Showing result status beside steps is useful

Seeing passed/failed/skipped state per step is very helpful. It turns the step list into a timeline, not just an editor list.

### 4. Request/response visibility is important

For an API scenario tool, request/response inspection is not optional. It is often the main reason someone runs a scenario manually.

---

## What is becoming problematic

### 1. The screen has too many vertical layers

The current hierarchy roughly becomes:

- Header/title/actions/environment/run/save
- Running/progress/result strip
- Spec/coverage warning
- Coverage banner
- Metadata section
- Variables section
- Run info section
- Step list and detail pane
- Step result strip
- Request/response exchange
- Step editor fields
- Advanced editor sections

This makes the screen feel compressed. Even if every individual section is reasonable, together they create **visual fatigue**.

### 2. Edit state and run result state are mixed too tightly

For a selected step, the user may see:

- result strip,
- request/response output,
- editable request fields,
- headers,
- query params,
- body,
- assertions,
- extractions,
- advanced options.

That means the same area is trying to answer two different questions:

- “What happened when I ran this?”
- “What do I want this step to be?”

Those are related, but they are not the same task.

### 3. Variables are globally important but locally distracting

Scenario variables matter, but they do not need to be permanently visible in the main vertical stack. They are usually configured occasionally, then referenced during editing.

When variables sit above the steps, they compete with the main workflow. The user has to scroll past or mentally ignore them.

### 4. Metadata/spec configuration is also not a daily focus

Title, description, and spec file are important, but they are not central during repeated edit/run cycles. Once set, they should recede.

### 5. Request/response content can easily dominate the screen

Request and response data are often large. If displayed inline above the editor, they can push the actual editing controls down. This makes the selected step pane feel like a long document rather than a focused tool.

### 6. The same step is being represented in too many ways

A step appears as:

- a row in the step list,
- a result row/strip,
- request/response exchange,
- editable step form,
- assertions/extractions form,
- possibly YAML representation elsewhere.

This is not inherently wrong, but without strong separation, it feels duplicated and heavy.

---

# 2. Main UX problem

The core problem is not that the Scenarios screen has too much functionality.

The core problem is:

> The screen does not clearly distinguish between **building**, **running**, and **inspecting**.

Those are three different user intents.

## Build intent

The user wants to author or modify the scenario:

- step name,
- method,
- URL,
- headers,
- query params,
- body,
- assertions,
- extractions,
- variables,
- retry/timeout behavior.

The interface should prioritize editable fields.

## Run intent

The user wants to execute:

- whole scenario,
- from selected step,
- possibly with environment selection,
- see progress,
- see pass/fail quickly,
- jump to failed step.

The interface should prioritize status and flow.

## Inspect intent

The user wants to understand what happened:

- request sent,
- response received,
- headers,
- body,
- assertion failures,
- extracted values,
- timing,
- attempts,
- errors.

The interface should prioritize evidence and debugging.

Currently all three intents are present at once.

---

# 3. Recommendation: keep one screen, introduce focused modes

I recommend keeping the Scenarios screen as a single destination, but adding an internal mode switch:

```plain text
Build | Run | Inspect
```


or possibly:

```plain text
Edit | Results
```


with a secondary inspector.

My preferred version is **three modes**, because it maps better to the actual workflow.

---

## Proposed high-level layout

### Global shell

Keep the same broad structure:

```plain text
┌───────────────────────────────────────────────┐
│ Scenario header: title, env, run, save, more   │
├───────────────────────────────────────────────┤
│ Build | Run | Inspect                          │
├───────────────┬───────────────────────────────┤
│ Step list      │ Mode-specific detail area     │
│ / timeline     │                               │
└───────────────┴───────────────────────────────┘
```


The left side remains stable. The right side changes based on mode.

This preserves the “same screen” benefit while reducing cognitive load.

---

# 4. Mode details

## A. Build mode

### Purpose

Build mode is for editing the scenario.

### What should be visible

- Step list.
- Selected step editor.
- Compact step status badges if a previous run exists.
- Save/discard.
- Run button still available.

### What should be hidden or minimized

- Full request/response exchange.
- Full run summary.
- Coverage details.
- Large result panels.
- Verbose metadata.
- Expanded variables by default.

### Selected step detail in Build mode

The selected step pane should primarily show:

```plain text
Step title / disabled / move/delete
Request
Headers
Query Params
Body
Assertions
Extractions
Advanced
```


If there is a latest result for the step, show only a compact summary:

```plain text
Last run: FAILED · 404 · 120 ms · 1 failed assertion
[Inspect result]
```


Clicking **Inspect result** switches to Inspect mode for that step.

### Why this helps

The user can focus on writing the scenario without being buried under output.

---

## B. Run mode

### Purpose

Run mode is for executing and watching scenario progress.

### What should be visible

- Scenario-level run summary.
- Timeline of steps.
- Current running step.
- Pass/fail/skipped counts.
- Duration.
- Failed assertion summaries.
- “Run all”, “Run from selected”, “Cancel”.

### What should be hidden or reduced

- Full editable request form.
- Full variables editor.
- Full metadata editor.
- Detailed request/response unless the user expands a step.

### Run mode layout

```plain text
┌─────────────────────────────────────┐
│ Run summary                         │
│ Passed 5 · Failed 1 · Skipped 2      │
│ Duration 2.4s · Environment staging  │
├─────────────────────────────────────┤
│ Step 1  PASSED  200  80ms            │
│ Step 2  PASSED  201  130ms           │
│ Step 3  FAILED  404  92ms            │
│   Assertion: expected status 200     │
│   [Inspect] [Run from here]          │
│ Step 4  SKIPPED                      │
└─────────────────────────────────────┘
```


### Why this helps

Run mode becomes a readable execution report rather than an editor with embedded run output.

---

## C. Inspect mode

### Purpose

Inspect mode is for debugging one selected step.

### What should be visible

- Selected step identity.
- Result status.
- Request sent.
- Response received.
- Assertions.
- Extractions.
- Error details.
- Attempts/timing.
- Option to jump back to edit the step.

### Main layout

```plain text
┌─────────────────────────────────────────────┐
│ Step 3: Create user                         │
│ FAILED · 404 · 92 ms                         │
├─────────────────────────────────────────────┤
│ Request | Response | Assertions | Extracted │
├─────────────────────────────────────────────┤
│ content area                                │
└─────────────────────────────────────────────┘
```


### Important note about request views

You said:

> request views content needs to be similar to each step's content display however does not need to be same

I agree with this direction.

The request view should reuse the same conceptual structure as the step editor, but it should be optimized for inspection, not editing.

For example:

### Step editor request display

```plain text
Method: [POST]
URL:    [{{baseUrl}}/users]

Headers:
[Content-Type] [application/json]

Body:
 editable JSON textarea
```


### Request inspection display

```plain text
POST https://api.example.com/users

Headers
Content-Type: application/json
Authorization: ***

Body
{
  "email": "user-abc@example.com"
}
```


The inspection view should be:

- read-only,
- formatted,
- copyable,
- redacted where needed,
- clearly showing resolved values,
- optionally showing “defined as” values when useful.

So yes: **similar content model, different presentation**.

---

# 5. Recommended information architecture

## Current issue

Metadata, variables, coverage, run results, and step editing all compete vertically.

## Recommended grouping

### Always visible

These should remain visible because they anchor the page:

- Scenario title.
- Environment selector.
- Run button.
- Save state.
- Mode switch.
- Step list/timeline.

### Usually collapsed or moved to side drawer

These should not consume primary vertical space:

- Description.
- Spec file.
- Scenario variables.
- Coverage banner.
- YAML/source view.

### Mode-specific

These should appear only when relevant:

| Content | Build mode | Run mode | Inspect mode |
|---|---:|---:|---:|
| Step editor | Primary | Hidden/minimized | Optional shortcut |
| Request/response | Hidden/minimized | Expandable | Primary |
| Assertions editor | Primary | Hidden | Assertion results primary |
| Variables editor | Drawer/collapsed | Hidden | Read-only resolved values optional |
| Run summary | Compact | Primary | Contextual |
| Coverage | Compact indicator | Summary card | Optional |
| Metadata | Collapsed | Hidden | Hidden |

---

# 6. Specific recommendations

## Recommendation 1: Add a mode switch

Add a top-level mode switch inside the scenario view:

```plain text
Build | Run | Inspect
```


Default behavior:

- Opening a scenario starts in **Build**.
- Clicking Run switches to **Run**.
- After run completes:
    - If failed, stay in Run mode and select first failed step.
    - If user clicks a step’s request/response, switch to Inspect mode.
- Clicking “Edit step” from Inspect returns to Build mode.

This creates a clean workflow:

```plain text
Build → Run → Inspect → Build
```


That is exactly how users debug scenarios.

---

## Recommendation 2: Keep the step list stable, but make it smarter

The step list should become the central navigation object.

In Build mode:

```plain text
1  GET /health
2  POST /users
3  GET /users/{{id}}
```


In Run/Inspect mode:

```plain text
1  PASS  GET /health          200  45ms
2  PASS  POST /users          201  120ms
3  FAIL  GET /users/{{id}}    404  90ms
```


Same list, different emphasis.

This avoids duplicating the step timeline elsewhere.

---

## Recommendation 3: Move metadata and variables into a “Scenario settings” drawer

The current metadata and variables sections are useful but too prominent.

I would move them under a button near the title:

```plain text
Scenario settings
```


Inside that drawer/panel:

```plain text
General
- Title
- Description
- Spec file

Variables
- name/value/secret
- dynamic helper text

Advanced
- YAML/source
```


This would immediately reduce visual clutter.

If variables are important for a selected step, show lightweight references in the step editor:

```plain text
Variables: baseUrl, userEmail, token
[Manage variables]
```


Or autocomplete can continue helping inside fields.

---

## Recommendation 4: Make request/response an inspector, not inline content above the editor

Currently, showing request/response directly inside the selected step detail risks pushing the editor down.

Instead:

In Build mode:

```plain text
Last response: 404 · 90 ms
[Inspect request/response]
```


In Inspect mode:

```plain text
Request | Response | Assertions | Extracted
```


This gives request/response the space it deserves without crowding the editing experience.

---

## Recommendation 5: Use progressive disclosure more aggressively

The step editor currently exposes many sections:

- Request,
- Headers,
- Query Params,
- Assertions,
- Extractions,
- Advanced.

This is valid, but should be made more scannable.

Recommended default expansion:

| Section | Default |
|---|---|
| Request | Open |
| Body | Open only if body exists |
| Headers | Collapsed unless values exist |
| Query Params | Collapsed unless values exist |
| Assertions | Open |
| Extractions | Collapsed unless values exist |
| Advanced | Collapsed unless non-default values exist |

This keeps common editing paths fast while avoiding empty-section clutter.

---

## Recommendation 6: Separate “definition” from “resolved runtime value”

This is especially important for request views.

In Build mode, the user sees the definition:

```plain text
URL: {{baseUrl}}/users/{{userId}}
Authorization: Bearer {{token}}
```


In Inspect mode, the user sees the resolved runtime request:

```plain text
GET https://staging.example.com/users/123
Authorization: ***
```


Optionally provide a toggle:

```plain text
Show: Resolved | Template
```


This would be very powerful and would make the request view meaningfully similar to the step editor while still serving a different purpose.

---

## Recommendation 7: Treat run results as a report, not decoration

The run result should not be scattered across the whole screen. It should have a dedicated Run mode with clear hierarchy:

1. Overall scenario status.
2. Failed step summary.
3. Step timeline.
4. Per-step details on demand.

This will make results easier to read and reduce visual noise in edit mode.

---

# 7. Suggested final layout

## Build mode

```plain text
┌────────────────────────────────────────────────────────────┐
│ Scenario title        Env: staging   Run   Save   Settings │
├────────────────────────────────────────────────────────────┤
│ Build | Run | Inspect                                      │
├──────────────────────┬─────────────────────────────────────┤
│ Steps                │ Step editor                         │
│                      │                                     │
│ 1 GET /health         │ Name: Create user                   │
│ 2 POST /users         │ Method: POST                        │
│ 3 GET /users/:id      │ URL: {{baseUrl}}/users              │
│                      │                                     │
│ + Add step            │ Assertions                          │
│                      │ Extractions                         │
│                      │ Advanced                            │
│                      │                                     │
│                      │ Last run: FAILED 404 [Inspect]      │
└──────────────────────┴─────────────────────────────────────┘
```


---

## Run mode

```plain text
┌────────────────────────────────────────────────────────────┐
│ Scenario title        Env: staging   Run again   Save      │
├────────────────────────────────────────────────────────────┤
│ Build | Run | Inspect                                      │
├──────────────────────┬─────────────────────────────────────┤
│ Steps                │ Run summary                         │
│                      │                                     │
│ ✓ 1 GET /health       │ FAILED                              │
│ ✓ 2 POST /users       │ 2 passed · 1 failed · 0 skipped      │
│ ✕ 3 GET /users/:id    │ Duration: 1.2s                       │
│                      │                                     │
│                      │ Failed step                          │
│                      │ Step 3 expected 200, got 404         │
│                      │                                     │
│                      │ [Inspect failed step] [Run from here]│
└──────────────────────┴─────────────────────────────────────┘
```


---

## Inspect mode

```plain text
┌────────────────────────────────────────────────────────────┐
│ Scenario title        Env: staging   Run from step          │
├────────────────────────────────────────────────────────────┤
│ Build | Run | Inspect                                      │
├──────────────────────┬─────────────────────────────────────┤
│ Steps                │ Step 3: Get created user             │
│                      │ FAILED · 404 · 90ms                  │
│ ✓ 1 GET /health       │                                     │
│ ✓ 2 POST /users       │ Request | Response | Assertions      │
│ ✕ 3 GET /users/:id    │                                     │
│                      │ GET https://.../users/123            │
│                      │ Headers                              │
│                      │ Body                                 │
│                      │                                     │
│                      │ [Edit this step]                     │
└──────────────────────┴─────────────────────────────────────┘
```


---

# 8. My honest opinion

I would **not** keep adding collapsible sections to the current single continuous screen. That will only delay the problem.

The screen needs a stronger conceptual split.

However, I would also **not** move scenario running to a completely separate page, because the edit-run-debug loop is one of the most valuable parts of the product.

The best solution is a middle path:

> One Scenarios screen, three focused working modes.

This keeps the product fast and integrated, while giving users breathing room.

---

# 9. Priority plan

## Phase 1: Reduce clutter without redesigning everything

Recommended first changes:

1. Move metadata and variables into a **Scenario settings** drawer.
2. Hide full request/response from the Build view.
3. Replace inline request/response with a compact result summary:
```plain text
Last run: FAILED · 404 · 90ms · Inspect
```

4. Make empty sections collapsed by default.

This alone will make the screen feel much lighter.

---

## Phase 2: Add Run and Inspect modes

Add:

```plain text
Build | Run | Inspect
```


Then make each mode prioritize one intent.

---

## Phase 3: Improve request/response inspection

Build a dedicated request/response viewer that is:

- similar in structure to step request content,
- read-only,
- formatted,
- resolved,
- copyable,
- redacted,
- tabbed by Request / Response / Assertions / Extracted.

---

# 10. Final recommendation

My recommendation is:

> Keep scenarios as one integrated workspace, but stop showing editing, execution, results, variables, metadata, coverage, and request/response details all at the same time.

The ideal Scenarios screen should feel like:

- **Build mode:** “I am authoring the scenario.”
- **Run mode:** “I am watching the scenario execute.”
- **Inspect mode:** “I am debugging what happened.”

That structure will preserve what you liked about the current design — editing and running together — while solving the crowding problem in a scalable way.

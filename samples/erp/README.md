# ERP Sample — BitCode Engine

Sample ERP application built entirely with JSON definitions. Demonstrates all engine features.

## New Features

- **WebSocket** — Real-time updates at `ws://localhost:8080/ws`
- **Admin UI** — Built-in admin panel at `http://localhost:8080/admin`
- **Multi-tenancy** — Set `TENANT_ENABLED=true` for tenant isolation
- **Python plugins** — `.py` scripts auto-detected alongside TypeScript
- **gRPC protocol** — Proto definition at `pkg/plugin/proto/plugin.proto`

## Modules

### CRM (depends: base)
- **Models**: contact, lead, activity, tag
- **Features used**: many2one, one2many, many2many, computed fields, record_rules, file upload
- **Workflow**: lead pipeline (new → contacted → qualified → proposal → won/lost)
- **Processes**: qualify_lead, convert_lead, win_lead, lose_lead, log_activity (uses validate, update, if, emit, assign, log, script)
- **Agents**: event triggers (lead.qualified, lead.won, lead.lost) + cron (weekly report, stale lead reminder)
- **Views**: list, form, kanban, custom dashboard
- **Templates**: dashboard.html with partials
- **Scripts**: TypeScript plugins (on_deal_won, notify_manager, weekly_report)
- **i18n**: Indonesian translations

### HRM (depends: base)
- **Models**: department, job_position, employee, leave_request
- **Features used**: many2one, one2many, self-referential FK (manager_id → employee), record_rules
- **Workflows**: employee lifecycle (probation → active → on_leave → terminated), leave approval (draft → submitted → approved/rejected)
- **Processes**: submit_leave, approve_leave, reject_leave, promote_employee (uses validate, query, if, update, emit, log, script)
- **Agents**: event triggers (leave.submitted, leave.approved, employee.promoted) + cron (weekly attendance)
- **Views**: list, form, custom dashboard
- **Scripts**: TypeScript plugins (on_leave_approved, on_promotion)
- **i18n**: Indonesian translations

## Feature Coverage

| Feature | CRM | HRM |
|---------|-----|-----|
| Model (fields, relationships) | ✅ | ✅ |
| many2one (FK) | ✅ contact_id, assigned_to | ✅ department_id, manager_id |
| one2many (reverse FK) | ✅ contact.leads | ✅ department.employees |
| many2many | ✅ contact.tags, lead.tags | - |
| computed fields | ✅ weighted_revenue | - |
| file upload | ✅ contact.photo | ✅ employee.photo |
| record_rules (RLS) | ✅ per user/manager | ✅ per user/officer |
| auto_crud API | ✅ | ✅ |
| auth (JWT + RLS) | ✅ | ✅ |
| workflow (state machine) | ✅ lead pipeline | ✅ employee + leave |
| process (business logic) | ✅ 5 processes | ✅ 4 processes |
| step: validate | ✅ | ✅ |
| step: update | ✅ | ✅ |
| step: if (conditional) | ✅ | ✅ |
| step: emit (events) | ✅ | ✅ |
| step: assign | ✅ | ✅ |
| step: log (audit) | ✅ | ✅ |
| step: query | - | ✅ |
| step: script (TS plugin) | ✅ | ✅ |
| step: script (Python plugin) | ✅ analyze_pipeline.py | ✅ generate_onboard_checklist.py |
| step: http (external API) | ✅ enrich_leads | ✅ onboard_employee |
| step: loop | ✅ enrich_leads | - |
| step: switch | ✅ enrich_leads | ✅ onboard_employee |
| step: call (sub-process) | ✅ | ✅ |
| agent (event triggers) | ✅ 3 triggers | ✅ 3 triggers |
| agent (cron jobs) | ✅ 2 cron | ✅ 1 cron |
| views: list | ✅ | ✅ |
| views: form | ✅ | ✅ |
| views: kanban | ✅ lead_kanban | - |
| views: calendar | - | ✅ leave_calendar |
| views: chart | ✅ pipeline_chart | - |
| views: custom (template) | ✅ dashboard | ✅ dashboard |
| templates (Go html) | ✅ | ✅ |
| partials | ✅ lead_card | - |
| i18n (Indonesian) | ✅ | ✅ |
| demo data | ✅ | ✅ |
| settings | ✅ | ✅ |
| permissions (RBAC) | ✅ 9 perms | ✅ 13 perms |
| groups (hierarchy) | ✅ user → manager | ✅ user → officer → manager |
| menu structure | ✅ | ✅ |
| WebSocket (real-time) | ✅ events broadcast | ✅ events broadcast |
| Admin UI | ✅ model/data browser | ✅ model/data browser |
| Multi-tenancy | ✅ tenant isolation | ✅ tenant isolation |
| Python plugin | ✅ analyze_pipeline.py | ✅ generate_onboard_checklist.py, calculate_leave_balance.py |
| Model inheritance | ✅ vip_contact inherits contact | - |
| Search (?q=) | ✅ name, email, company | ✅ name, email, employee_id |
| gRPC protocol | ✅ proto defined | ✅ proto defined |

## Quick Start

```bash
cd samples/erp

# Run with SQLite (zero config, default)
MODULE_DIR=modules go run ../../engine/cmd/engine/main.go

# Or with the CLI
MODULE_DIR=modules go run ../../engine/cmd/bitcode/main.go dev
```

## Test the API

```bash
# Health check
curl http://localhost:8080/health

# List contacts (no auth for demo)
curl http://localhost:8080/api/contact

# Create a contact
curl -X POST http://localhost:8080/api/contact \
  -H "Content-Type: application/json" \
  -d '{"name": "John Doe", "email": "john@example.com", "company": "Acme Inc"}'

# List leads
curl http://localhost:8080/api/lead

# Create a lead
curl -X POST http://localhost:8080/api/lead \
  -H "Content-Type: application/json" \
  -d '{"name": "Big Deal", "company": "MegaCorp", "expected_revenue": 100000, "source": "referral"}'

# Workflow action: qualify a lead
curl -X POST http://localhost:8080/api/lead/{id}/qualify

# List employees
curl http://localhost:8080/api/employee

# List departments
curl http://localhost:8080/api/department

# Create leave request
curl -X POST http://localhost:8080/api/leave_request \
  -H "Content-Type: application/json" \
  -d '{"employee_id": "...", "leave_type": "annual", "start_date": "2026-05-01", "end_date": "2026-05-03", "days": 3}'

# Approve leave
curl -X POST http://localhost:8080/api/leave_request/{id}/approve

# View SSR pages
curl http://localhost:8080/views/crm_dashboard
curl http://localhost:8080/views/hrm_dashboard
curl http://localhost:8080/views/contact_list
curl http://localhost:8080/views/lead_list
curl http://localhost:8080/views/employee_list

# Validate all definitions
MODULE_DIR=modules go run ../../engine/cmd/bitcode/main.go validate

# List modules
MODULE_DIR=modules go run ../../engine/cmd/bitcode/main.go module list
```

## Admin UI

Open `http://localhost:8080/admin` in your browser to see:
- Dashboard with model/module/view counts
- Model browser (fields, record rules, data)
- Module list with versions and dependencies
- View list with preview links

## WebSocket (Real-time)

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
ws.onmessage = (e) => console.log(JSON.parse(e.data));
ws.onopen = () => {
  ws.send(JSON.stringify({ type: 'subscribe', channel: 'lead.won' }));
  ws.send(JSON.stringify({ type: 'subscribe', channel: 'leave.approved' }));
};
```

Events are broadcast automatically when processes emit domain events.

## Multi-tenancy

```bash
TENANT_ENABLED=true TENANT_STRATEGY=header \
  MODULE_DIR=modules go run ../../engine/cmd/engine/main.go

curl -H "X-Tenant-ID: company-a" http://localhost:8080/api/contacts
curl -H "X-Tenant-ID: company-b" http://localhost:8080/api/contacts
```

Strategies: `header` (X-Tenant-ID), `subdomain` (company-a.app.com), `path` (/tenant/:id/...)

## With PostgreSQL

```bash
DB_DRIVER=postgres DB_HOST=localhost DB_USER=bitcode DB_PASSWORD=bitcode DB_NAME=erp_sample \
  MODULE_DIR=modules go run ../../engine/cmd/engine/main.go
```

## With Redis Cache

```bash
CACHE_DRIVER=redis REDIS_URL=redis://localhost:6379 \
  MODULE_DIR=modules go run ../../engine/cmd/engine/main.go
```

## Project Structure

```
samples/erp/
├── bitcode.yaml                    # Project config
├── README.md
└── modules/
    ├── crm/                        # CRM Module
    │   ├── module.json             # Module definition (deps, perms, groups, menu, settings)
    │   ├── models/
    │   │   ├── tag.json            # many2many target
    │   │   ├── contact.json        # many2one, one2many, many2many, file, record_rules
    │   │   ├── lead.json           # computed field, workflow field, record_rules
    │   │   └── activity.json       # many2one to lead + contact
    │   ├── apis/
    │   │   ├── tag_api.json        # auto_crud (public, no auth)
    │   │   ├── contact_api.json    # auto_crud + auth (RLS automatic)
    │   │   ├── lead_api.json       # auto_crud + auth + workflow + actions
    │   │   └── activity_api.json   # auto_crud + auth
    │   ├── processes/
    │   │   ├── lead_workflow.json   # State machine definition
    │   │   ├── qualify_lead.json    # validate → update → log → emit
    │   │   ├── convert_lead.json    # validate → if → assign → update → emit
    │   │   ├── win_lead.json        # validate → update → if → log → emit → script
    │   │   ├── lose_lead.json       # validate → update → log → emit
    │   │   └── log_activity.json    # validate → create → if → emit
    │   ├── agents/
    │   │   └── lead_agent.json      # 3 event triggers + 2 cron jobs
    │   ├── views/
    │   │   ├── contact_list.json    # list view
    │   │   ├── contact_form.json    # form view with tabs
    │   │   ├── lead_list.json       # list view with workflow actions
    │   │   ├── lead_form.json       # form view with layout + actions
    │   │   ├── lead_kanban.json     # kanban view grouped by status
    │   │   └── crm_dashboard.json   # custom view with data_sources
    │   ├── scripts/                 # TypeScript plugins
    │   ├── templates/               # Go html/template
    │   ├── data/demo.json           # Demo data
    │   └── i18n/id.json             # Indonesian translations
    └── hrm/                         # HRM Module
        ├── module.json
        ├── models/
        │   ├── department.json      # Self-referential FK (parent_id)
        │   ├── job_position.json    # one2many to employees
        │   ├── employee.json        # Self-ref FK (manager_id), record_rules
        │   └── leave_request.json   # Workflow + approval flow
        ├── apis/
        │   ├── department_api.json
        │   ├── position_api.json
        │   ├── employee_api.json    # auto_crud + workflow + actions
        │   └── leave_api.json       # auto_crud + workflow + approval actions
        ├── processes/
        │   ├── employee_workflow.json
        │   ├── leave_workflow.json
        │   ├── submit_leave.json    # validate → query → if → update → log → emit
        │   ├── approve_leave.json   # validate → update → log → emit → script
        │   ├── reject_leave.json    # validate → update → log → emit
        │   └── promote_employee.json # validate → assign → update → log → emit → script
        ├── agents/
        │   └── hr_agent.json        # 3 event triggers + 1 cron
        ├── views/
        ├── scripts/
        ├── templates/
        ├── data/demo.json
        └── i18n/id.json
```

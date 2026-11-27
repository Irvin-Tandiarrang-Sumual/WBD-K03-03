# Security: SAST with CodeQL

Static Application Security Testing (SAST) analyzes application source code without executing the program to identify security vulnerabilities early in the software development lifecycle.

The repository integrates **CodeQL** to parse source code into relational database graphs (Abstract Syntax Trees, Control Flow Graphs, and Data Flow Graphs) and execute standard queries to track data flow from untrusted input sources to sensitive execution sinks. Security findings are aligned with **OWASP Top 10** and **CWE Top 25** standards.

---

## 1. Workflow Configuration and Execution

SAST analysis is automated via GitHub Actions in [`.github/workflows/codeql.yml`](../.github/workflows/codeql.yml):
- **Automatic Triggers**: Runs on every `push` and `pull_request` targeting `main` and `develop`.
- **Manual Trigger**: Can be run on-demand via **Actions** > **CodeQL SAST** > **Run workflow** (`workflow_dispatch`).
- **Language Matrix**:
  - **Go (Backend)**: Uses `build-mode: manual` with compilation command `go build -v ./...` in the `backend/` directory.
  - **JavaScript/TypeScript (Frontend)**: Uses `build-mode: none` for static AST parsing and data-flow analysis of the React SPA.

Scan results are published to **Security** > **Code scanning alerts** on GitHub and uploaded as SARIF files in the Actions workflow run summary.

---

## 2. Code Coverage Scope

| Component | Language / Framework | Target Directories | Analysis Scope |
|---|---|---|---|
| **Backend** | Go 1.23+ / Gin, GORM | `backend/cmd/...`, `backend/internal/...`, `backend/pkg/...` | HTTP handlers, RBAC middleware, token authentication, business domain services, database drivers, queue workers, and migration engines. |
| **Frontend** | TypeScript / React 19 | `frontend-spa/src/...` | UI components, form inputs, client routing, and API HTTP client services. |

---

## 3. Vulnerability Findings and Security Evaluation

Evaluation of static analysis findings and risk decisions:

| ID | Rule / CWE | Severity | Location | Potential Impact (OWASP Top 10) | Decision | Rationale and Mitigations |
|---|---|---|---|---|---|---|
| **F-01** | `go/sql-injection`<br>(CWE-89) | **High** | `backend/internal/adapters/driver/postgresql/executor_repository.go:34`<br>`backend/internal/adapters/driver/mysql/executor_repository.go:34` | **A03:2021 – Injection**:<br>Arbitrary SQL execution against target databases. | **Accepted Risk**<br>(By Design) | Core platform purpose is database migration management which executes user-submitted DDL/DML. Protected by defense-in-depth: (1) Session cookie and CSRF validation; (2) Strict RBAC (`view_only` blocked from execution); (3) Multi-leader unanimous consensus approval required prior to execution; (4) Execution isolated to target databases, separated from platform metadata. |
| **F-02** | `js/xss`<br>(CWE-79) | **Medium** | `frontend-spa/src/**/*.tsx` | **A03:2021 – Injection**:<br>Cross-site scripting via unescaped input. | **Clean**<br>(0 Findings) | React 19 automatically escapes JSX expression bindings. No usage of `dangerouslySetInnerHTML`. Session cookies use `HttpOnly` and `SameSite=Lax`. |
| **F-03** | `go/csrf-protection`<br>(CWE-352) | **Medium** | `backend/internal/adapters/http/middleware/csrf.go` | **A01:2021 – Broken Access Control**:<br>Cross-site request forgery. | **Clean**<br>(Protected) | All state-mutating HTTP requests (`POST`, `PUT`, `PATCH`, `DELETE`) require a valid `X-CSRF-Token` header validated against the encrypted session cookie. |
| **F-04** | `go/path-injection`<br>(CWE-22) | **High** | Filesystem access | **A01:2021 – Broken Access Control**:<br>Unauthorized file access on server host. | **Clean**<br>(0 Findings) | Application exposes no endpoints that read or write arbitrary host filesystem paths from user input. |

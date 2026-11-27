# Demo Target Databases and Scenarios

The platform provides pre-seeded demo databases and test user accounts initialized on Docker startup when `SEED_TEST_DATA=true`.

---

## Demo User Accounts

Use these pre-configured accounts to test authentication, group-based Role-Based Access Control (RBAC), and multi-leader consensus approval:

| Username | Password | Global Role | Group & Database Access | Usage Scenario |
|---|---|---|---|---|
| `admin` | `admin123` | `admin` | `admin` (`view_edit`) | User management, group configuration, database registration, and executing approved drafts. |
| `leader` | `leader123` | `user` | `engineering` (Leader) | Leader with voting rights (`approve` / `reject`) on engineering group proposals. |
| `user` | `user123` | `user` | `engineering` (Member) | Regular developer: submits draft proposals and queries team databases. |
| `leader_qa` | `qa123` | `user` | `qa-team` (Leader) | Second leader to test multi-group consensus (unanimous 100% approval across groups). |
| `auditor_lead` | `auditor123` | `user` | `auditors` (Leader) | `view_only` access: inspects schemas and query execution plans (`EXPLAIN`), blocked from mutating data. |

---

## Target Database Profiles

Each demo database represents a specific migration use case:

| Database | Engine | Initial Seed State | Scenario & Behavior |
|---|---|---|---|
| `db1_pg` | PostgreSQL 15 | `v3_applied` (Normal) | **Reversible Migration History**: All applied versions have valid SQL Down definitions. Ideal for testing Rollback draft creation, multi-leader voting, and safe rollback execution. |
| `db2_mysql` | MySQL 8.4 | `v3_applied` (Normal) | **Irreversible Migration**: Contains a destructive column drop (`DROP COLUMN`). Tests automatic hazard detection where rollback actions are blocked to protect schema integrity. |
| `db3_analytics` | PostgreSQL 15 | `v1_applied` (Post-Rollback) | **Active Forward Pointer**: Previously rolled back, leaving an active forward pointer. Used to test Forward draft creation and re-applying migrations to the latest version. |
| `db4_billing` | PostgreSQL 15 | **Quarantined (HTTP 423 Locked)** | **Partial Batch Failure Recovery**: Simulates an interrupted batch DML job. The database is quarantined. Tests `Resume` (continuing from failure checkpoint) and `Revert` (rolling back committed chunks). |

---

## Interactive Walkthrough Scenarios

### 1. Safe Rollback (`db1_pg`)
1. Log in as `user` (`user123`).
2. Navigate to `db1_pg` > **Migrations** tab.
3. Select the latest migration version and submit a **Rollback Proposal**. The system automatically generates the reverse DDL (SQL Down).
4. Log in as `leader` (`leader123`) and vote **Approve**.
5. Log in as `leader_qa` (`qa123`) and vote **Approve** to satisfy 100% multi-leader consensus.
6. Execute the approved proposal. The database rolls back to the previous version.

### 2. Batch Quarantine Recovery (`db4_billing`)
1. Log in as `leader` (`leader123`).
2. Open `db4_billing`. An alert indicates the database is in **Quarantine** (`HTTP 423 Locked`).
3. Open the quarantined draft details.
4. Review the checkpoint log to see processed rows and failure location.
5. Choose a recovery action:
   - Click **Resume**: Background workers continue remaining chunks from the checkpoint.
   - Click **Revert**: Background workers execute reverse SQL only for rows already committed.

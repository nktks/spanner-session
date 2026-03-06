# spanner-session

Creates a Cloud Spanner session and outputs the session name. Optionally accepts a FGAC (Fine-Grained Access Control) database role to restrict session privileges.

## Why this tool is needed

[Spanner MCP Server](https://docs.cloud.google.com/spanner/docs/use-spanner-mcp) provides remote MCP tools (`create_session`, `execute_sql`, `get_database_ddl`, etc.) for AI agents like Claude Code. However, its `create_session` tool does not support the `creatorRole` parameter, meaning sessions are created without FGAC database roles. This gives the caller full access based on their IAM permissions.

This tool fills that gap by creating sessions with a specified FGAC role via the Spanner API directly. The output session name can then be passed to Spanner MCP's `execute_sql` tool, enforcing FGAC access control through MCP.

For example, with the built-in `spanner_sys_reader` role, sessions are restricted to read-only access on `INFORMATION_SCHEMA` and `SPANNER_SYS` tables, preventing access to application data.

## Prerequisites

- Go 1.21+
- ADC configured:
  ```bash
  # Normal IAM-based session
  gcloud auth application-default login

  # FGAC session with service account impersonation
  gcloud auth application-default login \
    --impersonate-service-account=<service-account>@<project>.iam.gserviceaccount.com
  ```
- For FGAC, the caller (or impersonated service account) needs:
  - `roles/spanner.fineGrainedAccessUser`
  - `roles/spanner.databaseRoleUser` (with condition: `resource.name.endsWith("/databaseRoles/<role>")`)

## Usage

```bash
go run github.com/nktks/spanner-session@latest \
  --project <project-id> \
  --instance <instance-id> \
  --database <database-name> \
  [--database-role <fgac-role>]
```

Outputs the session name to stdout.

- Without `--database-role`: creates a normal IAM-based session
- With `--database-role`: creates a FGAC session restricted to the specified database role

## Example

Given the following schema:

```sql
CREATE TABLE Users (
    Id STRING(20) NOT NULL,
    FirstName STRING(50),
    LastName STRING(50),
    Age INT64 NOT NULL,
    FullName STRING(100) AS (FirstName || ' ' || LastName) STORED,
) PRIMARY KEY (Id);
```

### Create a FGAC session

```bash
$ go run github.com/nktks/spanner-session@latest \
    --project my-project \
    --instance my-instance \
    --database my-database \
    --database-role spanner_sys_reader

projects/my-project/instances/my-instance/databases/my-database/sessions/AN-wRBd...
```

### Create a normal IAM-based session

```bash
$ spanner-session \
    --project my-project \
    --instance my-instance \
    --database my-database

projects/my-project/instances/my-instance/databases/my-database/sessions/BX-yQCe...
```

### Use the session with Spanner MCP

Pass the session to Spanner MCP's `execute_sql`. The FGAC role restricts what the session can access.

**Application table with FGAC session — DENIED:**

```
> execute_sql(session="projects/.../sessions/AN-wRBd...", sql="SELECT * FROM Users")

{
  "error": {
    "code": 403,
    "message": "Role spanner_sys_reader does not have required privileges on table Users.",
    "status": "PERMISSION_DENIED"
  }
}
```

**INFORMATION_SCHEMA with FGAC session — ALLOWED:**

```
> execute_sql(session="projects/.../sessions/AN-wRBd...", sql="SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES LIMIT 5")

{
  "rows": [
    ["CHANGE_STREAM_COLUMNS"],
    ["CHANGE_STREAM_OPTIONS"],
    ["COLUMNS"],
    ["COLUMN_OPTIONS"],
    ["INDEXES"]
  ]
}
```

**Application table with normal session — ALLOWED:**

```
> execute_sql(session="projects/.../sessions/BX-yQCe...", sql="SELECT * FROM Users LIMIT 3")

{
  "rows": [
    ["u001", "John", "Doe", "30", "John Doe"],
    ["u002", "Jane", "Smith", "25", "Jane Smith"],
    ["u003", "Bob", "Wilson", "35", "Bob Wilson"]
  ]
}
```

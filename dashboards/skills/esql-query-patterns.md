# ES|QL Query Patterns

Common patterns for generating ES|QL queries from natural language requests.

## Pattern Recognition Guide

When translating natural language to ES|QL, identify these key elements:

| User Says                               | ES\|QL Element                            |
| --------------------------------------- | ----------------------------------------- |
| "show", "list", "get", "find"           | `FROM` + `KEEP` (select fields)           |
| "from", "in" (index)                    | `FROM index-pattern`                      |
| "where", "with", "that have", "filter"  | `WHERE condition`                         |
| "last X hours/days", "since", "between" | `WHERE @timestamp > NOW() - X time`       |
| "count", "how many"                     | `STATS count = COUNT(*)`                  |
| "average", "mean"                       | `STATS avg = AVG(field)`                  |
| "total", "sum"                          | `STATS total = SUM(field)`                |
| "maximum", "highest", "top value"       | `STATS max = MAX(field)`                  |
| "minimum", "lowest"                     | `STATS min = MIN(field)`                  |
| "by", "per", "grouped by", "for each"   | `... BY field`                            |
| "top N", "first N", "limit"             | `LIMIT N`                                 |
| "sorted by", "order by"                 | `SORT field [DESC/ASC]`                   |
| "unique", "distinct"                    | `COUNT_DISTINCT(field)`                   |
| "contains", "includes"                  | `WHERE field LIKE "*value*"` or `MATCH()` |
| "starts with"                           | `WHERE STARTS_WITH(field, "prefix")`      |
| "ends with"                             | `WHERE ENDS_WITH(field, "suffix")`        |

---

## Time-Based Queries

### Recent Data

```
"show errors from the last hour"
→
FROM logs-*
| WHERE @timestamp > NOW() - 1 hour
| WHERE level == "error"
| SORT @timestamp DESC
| LIMIT 100
```

### Time Range

```
"events between January 1 and January 15, 2024"
→
FROM events-*
| WHERE @timestamp >= "2024-01-01" AND @timestamp < "2024-01-16"
```

### Time Bucketing

```
"count events per hour for today"
→
FROM events-*
| WHERE @timestamp > NOW() - 24 hours
| STATS count = COUNT(*) BY bucket = DATE_TRUNC(1 hour, @timestamp)
| SORT bucket DESC
```

### Time Comparisons

```
"requests slower than 5 seconds"
→
FROM api-logs
| WHERE response_time > 5000
| SORT response_time DESC
| LIMIT 100
```

---

## Aggregation Patterns

### Simple Count

```
"how many errors are there"
→
FROM logs-*
| WHERE level == "error"
| STATS total_errors = COUNT(*)
```

### Count by Category

```
"count of events by status code"
→
FROM web-logs
| STATS count = COUNT(*) BY status_code
| SORT count DESC
```

### Multiple Aggregations

```
"show min, max, and average response time"
→
FROM api-logs
| STATS
    min_time = MIN(response_time),
    max_time = MAX(response_time),
    avg_time = AVG(response_time)
```

### Grouped Multiple Aggregations

```
"average and max CPU per host"
→
FROM metrics-*
| STATS
    avg_cpu = AVG(system.cpu.percent),
    max_cpu = MAX(system.cpu.percent)
  BY host.name
| SORT avg_cpu DESC
```

### Top N Pattern

```
"top 10 hosts by error count"
→
FROM logs-*
| WHERE level == "error"
| STATS error_count = COUNT(*) BY host.name
| SORT error_count DESC
| LIMIT 10
```

### Percentiles

```
"p50, p95, p99 response times by endpoint"
→
FROM api-logs
| STATS
    p50 = PERCENTILE(response_time, 50),
    p95 = PERCENTILE(response_time, 95),
    p99 = PERCENTILE(response_time, 99)
  BY endpoint
| SORT p99 DESC
```

### Unique Counts

```
"count of unique users per day"
→
FROM user-events
| STATS unique_users = COUNT_DISTINCT(user_id) BY day = DATE_TRUNC(1 day, @timestamp)
| SORT day DESC
```

---

## Filtering Patterns

### Exact Match

```
"errors from production"
→
FROM logs-*
| WHERE level == "error" AND environment == "production"
```

### Multiple Values (IN)

```
"events with status 400, 401, or 403"
→
FROM web-logs
| WHERE status_code IN (400, 401, 403)
```

### Pattern Matching

```
"requests to /api endpoints"
→
FROM web-logs
| WHERE url LIKE "/api/*"
```

### Full-Text Search (8.17+)

```
"documents containing 'connection timeout'"
→
FROM logs-* METADATA _score
| WHERE MATCH(message, "connection timeout")
| SORT _score DESC
| LIMIT 100
```

### Null Handling

```
"records where error field exists"
→
FROM logs-*
| WHERE error IS NOT NULL
```

### Negation

```
"all events except from test environment"
→
FROM events-*
| WHERE environment != "test"
```

---

## Transformation Patterns

### Computed Fields

```
"show response time in seconds"
→
FROM api-logs
| EVAL response_time_sec = response_time_ms / 1000
| KEEP endpoint, response_time_sec
```

### String Manipulation

```
"extract domain from email addresses"
→
FROM users
| EVAL domain = SUBSTRING(email, LOCATE("@", email) + 1, LENGTH(email))
| KEEP email, domain
```

### Conditional Values

```
"categorize response times as fast/medium/slow"
→
FROM api-logs
| EVAL speed_category = CASE(
    response_time < 100, "fast",
    response_time < 500, "medium",
    "slow"
  )
| STATS count = COUNT(*) BY speed_category
```

### Rate Calculation

```
"error rate percentage by service"
→
FROM logs-*
| STATS
    total = COUNT(*),
    errors = COUNT(CASE(level == "error", 1, null))
  BY service.name
| EVAL error_rate = ROUND(errors * 100.0 / total, 2)
| SORT error_rate DESC
```

---

## Log Parsing Patterns

### GROK for Structured Extraction

```
"parse Apache access logs"
→
FROM raw-logs
| GROK message "%{IP:client_ip} - - \\[%{HTTPDATE:timestamp}\\] \"%{WORD:method} %{URIPATHPARAM:path} HTTP/%{NUMBER:http_version}\" %{NUMBER:status:int} %{NUMBER:bytes:int}"
| KEEP client_ip, method, path, status, bytes
```

### DISSECT for Simple Patterns

```
"extract user and action from 'User X performed Y'"
→
FROM audit-logs
| DISSECT message "User %{user} performed %{action}"
| STATS count = COUNT(*) BY user, action
```

---

## Advanced Patterns

### Multi-Index Query

```
"combine data from logs and metrics"
→
FROM logs-*, metrics-*
| WHERE @timestamp > NOW() - 1 hour
| KEEP @timestamp, host.name, message, cpu.percent
```

### Data Enrichment

```
"add geo info to IP addresses"
→
FROM web-logs
| ENRICH geoip-policy ON client.ip WITH country_name, city_name
| STATS requests = COUNT(*) BY country_name
| SORT requests DESC
```

### Multivalue Handling

```
"count occurrences of each tag"
→
FROM documents
| MV_EXPAND tags
| STATS count = COUNT(*) BY tags
| SORT count DESC
```

### Chained Aggregations

```
"average daily count per week"
→
FROM events
| STATS daily_count = COUNT(*) BY day = DATE_TRUNC(1 day, @timestamp)
| STATS avg_daily = AVG(daily_count) BY week = DATE_TRUNC(1 week, day)
| SORT week DESC
```

---

## Common Mistakes to Avoid

1. **Forgetting LIMIT** - Always add `LIMIT` to prevent returning too many rows

2. **Wrong time field** - Common names: `@timestamp`, `timestamp`, `time`, `date`

3. **Case sensitivity** - Field names are case-sensitive: `host.Name` ≠ `host.name`

4. **String vs Keyword** - Use `.keyword` suffix for exact matches on text fields:

   ```esql
   WHERE status.keyword == "active"
   ```

5. **Type mismatches** - Convert types when needed:

   ```esql
   | EVAL num = TO_INTEGER(string_field)
   ```

6. **STATS without aggregation** - STATS requires aggregate functions:

   ```esql
   // Wrong: | STATS BY host
   // Right: | STATS count = COUNT(*) BY host
   ```

7. **Missing FROM** - Every query must start with a source command

8. **Pipe placement** - Each command needs a pipe before it (except FROM)

---

# ES|QL Query Generation Tips

Guidelines for generating accurate ES|QL queries from natural language.

## Step-by-Step Generation Process

### 1. Identify the Data Source

**Question:** What index or data should be queried?

- Look for index names, data types, or subject areas mentioned
- Common patterns: `logs-*`, `metrics-*`, `events-*`, `apm-*`
- If unclear, use wildcards or ask for clarification

```esql
FROM logs-*           // Generic logs
FROM metrics-*        // Metrics data
FROM .ds-*            // Data streams
FROM my-index-2024.*  // Dated indices
```

### 2. Determine Time Range

**Question:** What time period should be covered?

| User Expression | ES\|QL                                                             |
| --------------- | ------------------------------------------------------------------ |
| "last hour"     | `@timestamp > NOW() - 1 hour`                                      |
| "last 24 hours" | `@timestamp > NOW() - 24 hours`                                    |
| "last 7 days"   | `@timestamp > NOW() - 7 days`                                      |
| "today"         | `@timestamp > NOW() - 24 hours`                                    |
| "yesterday"     | `@timestamp >= NOW() - 48 hours AND @timestamp < NOW() - 24 hours` |
| "this week"     | `@timestamp > NOW() - 7 days`                                      |
| "this month"    | `@timestamp > NOW() - 30 days`                                     |

**Default:** If no time range specified, consider adding a reasonable default (e.g., last 24 hours) to avoid scanning too much data.

### 3. Identify Filters

**Question:** What conditions should narrow the results?

Look for:

- Status/level: "errors", "warnings", "successful"
- Environment: "production", "staging", "dev"
- Source/host: specific servers, services, applications
- Values: specific codes, IDs, names

```esql
// Multiple filters
| WHERE level == "error"
| WHERE environment == "production"
| WHERE service.name == "api-gateway"
```

Or combined:

```esql
| WHERE level == "error" AND environment == "production" AND service.name == "api-gateway"
```

### 4. Determine Output Type

**Question:** Does the user want raw data or aggregated results?

| User Intent                   | Approach                        |
| ----------------------------- | ------------------------------- |
| "show me", "list", "find"     | Raw data with KEEP, SORT, LIMIT |
| "count", "how many"           | STATS with COUNT                |
| "average", "total", "sum"     | STATS with aggregation function |
| "by X", "per X", "grouped by" | STATS ... BY grouping           |
| "top N", "most common"        | STATS + SORT DESC + LIMIT       |
| "distribution", "breakdown"   | STATS COUNT BY category         |
| "over time", "trend"          | STATS BY DATE_TRUNC             |

### 5. Select Fields

**Question:** What fields should be shown?

For raw data queries, use KEEP to select relevant fields:

```esql
| KEEP @timestamp, host.name, message, level
```

For aggregations, the output fields are defined by STATS:

```esql
| STATS count = COUNT(*), avg_time = AVG(response_time) BY endpoint
```

### 6. Apply Ordering and Limits

**Question:** How should results be ordered and limited?

- Time-based: `SORT @timestamp DESC`
- By count/value: `SORT count DESC`
- Alphabetical: `SORT name ASC`

**Always add LIMIT** unless the user specifically wants all results:

```esql
| LIMIT 100  // Reasonable default
| LIMIT 1000 // Maximum before considering pagination
```

---

## Field Name Conventions

When generating queries, use common field naming conventions:

### Elastic Common Schema (ECS)

| Category    | Common Fields                                                  |
| ----------- | -------------------------------------------------------------- |
| Timestamp   | `@timestamp`                                                   |
| Message     | `message`                                                      |
| Log level   | `log.level`, `level`                                           |
| Host        | `host.name`, `host.ip`                                         |
| Service     | `service.name`, `service.type`                                 |
| HTTP        | `http.request.method`, `http.response.status_code`, `url.path` |
| User        | `user.name`, `user.id`                                         |
| Source      | `source.ip`, `source.port`                                     |
| Destination | `destination.ip`, `destination.port`                           |
| Error       | `error.message`, `error.type`                                  |
| Event       | `event.action`, `event.category`, `event.outcome`              |

### Legacy/Custom Fields

Some indices may use non-ECS field names:

- `status_code` instead of `http.response.status_code`
- `hostname` instead of `host.name`
- `timestamp` instead of `@timestamp`

**Recommendation:** When unsure, use `./esql.js schema <index>` to discover actual field names.

---

## Query Optimization Tips

### 1. Filter Early

Put WHERE clauses as early as possible:

```esql
// Good - filter first
FROM logs-*
| WHERE @timestamp > NOW() - 1 hour
| WHERE level == "error"
| STATS count = COUNT(*) BY host.name

// Less efficient - filtering after processing
FROM logs-*
| STATS count = COUNT(*) BY host.name, level
| WHERE level == "error"
```

### 2. Use Appropriate Time Ranges

Smaller time ranges = faster queries:

```esql
// Specific range is faster
| WHERE @timestamp > NOW() - 1 hour

// Than scanning all data
// (no time filter)
```

### 3. Limit Fields

Only keep fields you need:

```esql
// Good - specific fields
| KEEP @timestamp, message, host.name

// Less efficient - all fields
// (no KEEP command)
```

### 4. Use LIMIT

Prevent returning excessive rows:

```esql
| LIMIT 100  // Always include for raw data queries
```

---

## Common Query Templates

### Error Investigation

```esql
FROM logs-*
| WHERE @timestamp > NOW() - 1 hour
| WHERE level == "error"
| KEEP @timestamp, message, host.name, service.name, error.message
| SORT @timestamp DESC
| LIMIT 100
```

### Service Health Overview

```esql
FROM metrics-*
| WHERE @timestamp > NOW() - 15 minutes
| STATS
    avg_cpu = AVG(system.cpu.percent),
    avg_mem = AVG(system.memory.used.pct),
    host_count = COUNT_DISTINCT(host.name)
  BY service.name
| SORT avg_cpu DESC
```

### API Performance Analysis

```esql
FROM apm-*
| WHERE @timestamp > NOW() - 1 hour
| STATS
    count = COUNT(*),
    avg_duration = AVG(transaction.duration.us),
    p95_duration = PERCENTILE(transaction.duration.us, 95),
    error_count = COUNT(CASE(transaction.result != "success", 1, null))
  BY transaction.name
| EVAL error_rate = ROUND(error_count * 100.0 / count, 2)
| SORT count DESC
| LIMIT 20
```

### Traffic Analysis

```esql
FROM web-logs
| WHERE @timestamp > NOW() - 24 hours
| STATS
    requests = COUNT(*),
    unique_ips = COUNT_DISTINCT(client.ip)
  BY hour = DATE_TRUNC(1 hour, @timestamp)
| SORT hour DESC
```

### Security Event Review

```esql
FROM security-*
| WHERE @timestamp > NOW() - 24 hours
| WHERE event.category == "authentication"
| WHERE event.outcome == "failure"
| STATS
    failures = COUNT(*)
  BY user.name, source.ip
| WHERE failures > 5
| SORT failures DESC
```

---

## Handling Ambiguity

When the user request is ambiguous:

### Missing Index

If no index specified, make a reasonable assumption:

- "show errors" → `FROM logs-*`
- "show CPU usage" → `FROM metrics-*`
- "show requests" → `FROM web-logs` or `FROM access-*`

Or output the query with a placeholder and note:

```esql
FROM <index-pattern>  // Specify your index
| WHERE ...
```

### Missing Time Range

Add a sensible default:

```esql
| WHERE @timestamp > NOW() - 24 hours  // Default: last 24 hours
```

### Unclear Aggregation

When "show X" could mean list or count:

- If followed by "by Y" → aggregation
- If asking for specifics → raw data
- If asking "how many" → count
- Default to raw data with limit

### Unknown Field Names

If field names are uncertain:

1. Use common ECS names as first guess
2. Suggest running schema discovery
3. Note the assumption in output

---

## Output Formatting Suggestions

When presenting generated queries:

```
=== ES|QL Query ===

FROM logs-*
| WHERE @timestamp > NOW() - 1 hour
| WHERE level == "error"
| STATS count = COUNT(*) BY host.name
| SORT count DESC
| LIMIT 10

=== Explanation ===
- Queries all log indices
- Filters to the last hour
- Counts errors per host
- Returns top 10 hosts by error count

=== To Execute ===
./esql.js raw "FROM logs-* | WHERE @timestamp > NOW() - 1 hour | WHERE level == \"error\" | STATS count = COUNT(*) BY host.name | SORT count DESC | LIMIT 10"
```

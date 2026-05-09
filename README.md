# Gator

Gator is a CLI RSS feed aggregator written in Go.

It allows users to:

- Register/login users
- Add RSS feeds
- Follow/unfollow feeds
- Aggregate posts from feeds
- Browse posts directly from the terminal

---

# Requirements

You must have these installed:

- Go
- PostgreSQL

## Install Go

Download from:

https://go.dev/dl/

## Install PostgreSQL

Download from:

https://www.postgresql.org/download/

---

# Installation

Install the CLI using:

```bash
go install github.com/shoaibaziz-afk/Blog_aggregator@latest
```

After installation, the `gator` binary can be run directly from your terminal.

---

# Database Setup

Create a PostgreSQL database:

```sql
CREATE DATABASE gator;
```

---

# Config File

Create a config file at:

```text
~/.gatorconfig.json
```

Example config:

```json
{
  "db_url": "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable",
  "current_user_name": ""
}
```

Update the username/password if your PostgreSQL credentials are different.

---

# Running Migrations

Run migrations with Goose:

```bash
goose -dir sql/schema postgres "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable" up
```

---

# Build

Build the project:

```bash
go build
```

This creates a compiled binary.

Run the program:

```bash
./Blog_aggregator
```

---

# Commands

## Register User

```bash
./Blog_aggregator register shoaib
```

## Login

```bash
./Blog_aggregator login shoaib
```

## Add Feed

```bash
./Blog_aggregator addfeed "TechCrunch" "https://techcrunch.com/feed/"
```

## Follow Feed

```bash
./Blog_aggregator follow "https://techcrunch.com/feed/"
```

## Unfollow Feed

```bash
./Blog_aggregator unfollow "https://techcrunch.com/feed/"
```

## List Feeds

```bash
./Blog_aggregator feeds
```

## Following

```bash
./Blog_aggregator following
```

## Aggregate Feeds

```bash
./Blog_aggregator agg 30s
```

## Browse Posts

```bash
./Blog_aggregator browse
```

Browse with custom limit:

```bash
./Blog_aggregator browse 10
```

---

# Notes

- `go run .` is mainly for development.
- After building/installing, use the compiled binary directly.
- Stop the aggregator using `Ctrl + C`.

---

# Example RSS Feeds

TechCrunch:

```text
https://techcrunch.com/feed/
```

Hacker News:

```text
https://news.ycombinator.com/rss
```

Boot.dev Blog:

```text
https://www.boot.dev/blog/index.xml
```

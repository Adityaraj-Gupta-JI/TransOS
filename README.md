```markdown
# TransOS: Cross-Platform Environment State & Configuration Migration Engine

TransOS is a zero-agent, high-performance command-line interface (CLI) tool designed to automate the cross-platform extraction and injection of user-space environment configurations, system paths, and shell parameters between heterogeneous operating systems (Windows and Linux).

---

## Technical Architecture

TransOS decouples state extraction from state injection to eliminate hypervisor dependencies and persistent background daemons:

1. **Extraction Phase (`transos extract`):** Gathers host-level environment variables, user configuration paths, and theme parameters, serializing them into a strictly typed, schema-validated JSON payload (`migration_profile.json`).
2. **Transformation Engine:** Parses shell path formats (normalizing Windows `\` paths to POSIX `/` standard equivalents) and verifies data integrity via JSON Schema v7 specifications.
3. **Injection Phase (`transos inject`):** Translates the payload parameters directly into Linux session structures (such as environment configuration files and desktop properties) backed by a Write-Ahead Log (WAL) transactional rollback safety layer.

---

## Project Structure

```text
transos/
├── cmd/
│   └── transos/
│       └── main.go          # CLI entry point, banner, and interactive TUI control panel
├── internal/
│   ├── extractor/
│   │   └── windows.go       # Real host environment & configuration extraction module
│   ├── injector/
│   │   └── linux.go         # POSIX environment & dotfile injection module
│   ├── schema/
│   │   └── model.go         # Schema v7 payload structures and serialization logic
│   └── wal/
│       └── logger.go        # Transactional Write-Ahead Log safety tracker
├── go.mod
├── go.sum
└── README.md
```

## Installation & Setup

Ensure you have Go (version 1.22 or higher) installed on your local machine.

### Clone the repository

```bash
git clone [https://github.com/Adityaraj-Gupta-JI/TransOS.git](https://github.com/Adityaraj-Gupta-JI/TransOS.git)
cd TransOS
```

### Initialize and tidy Go modules

```bash
go mod tidy
```

## Usage Guide

TransOS can be operated via direct CLI arguments or through its built-in interactive control panel.

### 1. Interactive Control Panel (TUI)

Launch the interactive terminal interface:

```bash
go run cmd/transos/main.go interactive
```

### 2. Direct CLI Commands

#### Extract Host State

```bash
go run cmd/transos/main.go extract
```

#### Inject & Sync State

```bash
go run cmd/transos/main.go inject
```

## Academic Project Context

Developed as an advanced Systems Engineering and Operating Systems project, TransOS explores user-space virtualization alternatives to traditional full-system disk or memory snapshot migration tools (such as ISR and Zap).

## License

All the rights are reserved to the owner @2026.

---
```
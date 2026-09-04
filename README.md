<div align="center">

# TransOS

### Cross-Platform Environment State & Configuration Migration Engine

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux-blue?style=for-the-badge)](#)
[![Build](https://img.shields.io/badge/Build-Passing-brightgreen?style=for-the-badge)](#)
[![Schema Version](https://img.shields.io/badge/Schema-v7-orange?style=for-the-badge)](#)

</div>

---

**TransOS** is a zero-agent, high-performance command-line interface (CLI) tool designed to automate the cross-platform extraction and injection of user-space environment configurations, system paths, and shell parameters between heterogeneous operating systems (Windows and Linux).

---

## Table of Contents

- [Technical Architecture](#technical-architecture)
- [Project Structure](#project-structure)
- [Installation & Setup](#installation--setup)
- [Usage Guide](#usage-guide)
- [Academic Project Context](#academic-project-context)
- [License](#license)

---

## Technical Architecture

TransOS decouples state extraction from state injection to eliminate hypervisor dependencies and persistent background daemons:

<table>
<tr>
<td width="25%" valign="top">

### 🔍 Extraction Phase
`transos extract`

Gathers host-level environment variables, user configuration paths, HKCU Win32 registry keys, and installed application inventories, serializing them into a strictly typed, schema-validated JSON payload (`migration_profile.json`).

</td>
<td width="25%" valign="top">

### 🌳 AST Path Translation Engine
`internal/translator`

Parses Windows path primitives (`C:\Users\...`, `%APPDATA%`, multi-path string variables separated by `;`) into Abstract Syntax Trees (AST) and normalizes them into POSIX-compliant Linux paths (`~/.config/...`, path separators `:`) dynamically.

</td>
<td width="25%" valign="top">

### 📦 Dependency Script Generator
`internal/translator/package_mapper.go`

Maps Windows installed software against a Linux software catalog to generate an automated bash dependency downloader script (`install_dependencies.sh`).

</td>
<td width="25%" valign="top">

### 🧾 WAL Transaction Engine
`internal/wal`

Protects target system state by logging atomic transactions to `transos.wal` before executing mutations, allowing complete system state rollback via `transos rollback`.

</td>
</tr>
</table>

> **Note:** All transformations are performed entirely in user-space — no hypervisor, kernel module, or background daemon is required at any stage of the pipeline.

---

## Project Structure

```text
transos/
├── cmd/
│   └── transos/
│       └── main.go          # CLI entry point, banner, and interactive TUI control panel
├── internal/
│   ├── extractor/
│   │   └── windows.go       # Win32 Registry & host environment extraction module
│   ├── injector/
│   │   └── linux.go         # POSIX environment & dotfile injection engine
│   ├── schema/
│   │   └── model.go         # Schema v7 payload structures and serialization logic
│   ├── translator/
│   │   ├── ast.go           # AST Path Translation Engine (Win-to-POSIX primitive mapping)
│   │   └── package_mapper.go # Windows-to-Linux package mapper for installer generation
│   └── wal/
│       └── logger.go        # Transactional Write-Ahead Log (WAL) engine & rollback handler
├── target_output/           # Generated migration deliverables and transaction logs
│   ├── transos_env.conf
│   ├── install_dependencies.sh
│   └── transos.wal
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

---

## Installation & Setup

### Prerequisites

| Requirement | Version |
|---|---|
| Go | `1.22+` |
| OS (source) | Windows |
| OS (target) | Linux |

### 1. Clone the Repository

```bash
git clone https://github.com/Adityaraj-Gupta-JI/TransOS.git
cd TransOS
```

### 2. Initialize and Tidy Go Modules

```bash
go mod tidy
```

---

## Usage Guide

TransOS can be operated via direct CLI arguments or through its built-in interactive control panel.

### 1. Direct CLI Commands

**Extract Host State**

```bash
go run cmd/transos/main.go extract
```

**Inject & Sync State**

```bash
go run cmd/transos/main.go inject
```

**Rollback Target State**

```bash
go run cmd/transos/main.go rollback
```

### 2. Interactive Control Panel (TUI)

Launch the interactive terminal interface:

```bash
go run cmd/transos/main.go interactive
```

---

## Academic Project Context

Developed as an advanced **Systems Engineering and Operating Systems** project, TransOS explores user-space virtualization alternatives to traditional full-system disk or memory snapshot migration tools (such as ISR and Zap).

---

## License

This project is proprietary software. All rights are reserved by the owner.

No part of this project, including its source code, documentation, or associated files, may be copied, modified, distributed, sublicensed, published, or used for commercial purposes without prior written permission from the owner.

For permission requests, licensing inquiries, or other usage-related questions, please contact the project owner through the repository or the official contact information provided by the owner.

Copyright (c) 2026 TransOS Owner. All rights reserved.
# Feature: Fix Distribution - Binary Releases + Homebrew Tap

## Overview

Set up automated binary releases via GitHub Actions + GoReleaser, plus a Homebrew tap for macOS users. This gives users three installation options:

1. `brew install gerunddev/tap/jjazy` (macOS)
2. `curl -fsSL .../install.sh | bash` (macOS/Linux)
3. `git clone && make install` (developers)

---

## Manual Steps (You Must Do These)

### Step 1: Create Homebrew Tap Repository

**When:** Before or during implementation

1. Go to https://github.com/new
2. Create repository named `homebrew-tap` under your account
   - Owner: `gerunddev`
   - Name: `homebrew-tap`
   - Public repository
   - Initialize with README (optional)
3. Clone it locally (the developing agent will need to create the formula file):
   ```bash
   cd ~/Developer
   git clone https://github.com/gerunddev/homebrew-tap.git
   ```

### Step 2: Verify GitHub Actions Permissions

**When:** Before first release

1. Go to https://github.com/gerunddev/jjazy/settings/actions
2. Under "Workflow permissions", ensure:
   - "Read and write permissions" is selected
   - "Allow GitHub Actions to create and approve pull requests" is checked (optional)

### Step 3: Create a GitHub Personal Access Token (for Homebrew auto-update)

**When:** If you want releases to auto-update the Homebrew formula

1. Go to https://github.com/settings/tokens
2. Generate new token (classic) with scopes:
   - `repo` (full control of private repositories)
   - Or just `public_repo` if homebrew-tap is public
3. Copy the token
4. Go to https://github.com/gerunddev/jjazy/settings/secrets/actions
5. Add new secret:
   - Name: `HOMEBREW_TAP_TOKEN`
   - Value: (paste your token)

### Step 4: First Release (After Implementation)

**When:** After all code is merged

1. Ensure you're on main with all changes:
   ```bash
   jj git push
   ```

2. Create and push a version tag:
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

3. Watch the release workflow:
   - Go to https://github.com/gerunddev/jjazy/actions
   - Watch the "Release" workflow run
   - Verify it creates a GitHub Release with binaries

4. Test installation methods:
   ```bash
   # Test install script
   curl -fsSL https://raw.githubusercontent.com/gerunddev/jjazy/main/install.sh | bash

   # Test Homebrew (after formula is pushed)
   brew install gerunddev/tap/jjazy
   ```

### Step 5: (Optional) Install GoReleaser Locally for Testing

```bash
brew install goreleaser

# Test the config without publishing
goreleaser build --snapshot --clean
```

---

## Files to Create/Modify (Developing Agent)

### Task 1: GoReleaser Configuration

**File:** `.goreleaser.yaml`

**Purpose:** Configure how binaries are built and packaged for each platform.

**Key requirements:**
- Use `before.hooks` to build Rust library before Go compilation
- Build for: darwin-arm64, darwin-amd64, linux-amd64, linux-arm64
- CGO_ENABLED=1 for all builds
- Package as `.tar.gz` with LICENSE and README
- Generate checksums file

**Acceptance criteria:**
- `goreleaser build --snapshot --clean --single-target` produces working binary locally

**Dependencies:** none

---

### Task 2: GitHub Actions Release Workflow

**File:** `.github/workflows/release.yml`

**Purpose:** Automated pipeline that triggers on tag push and builds all platform binaries.

**Key requirements:**
- Trigger on `v*` tags
- Matrix strategy for platforms:
  - `macos-latest` (arm64) - Apple Silicon
  - `macos-13` (amd64) - Intel Mac
  - `ubuntu-latest` (amd64)
  - `ubuntu-24.04-arm` (arm64) - ARM Linux
- Install Rust toolchain
- Install Go toolchain
- Run GoReleaser with `--split` and `--merge` for matrix builds
- Upload all artifacts to GitHub Release

**Note on architecture:** GitHub's `macos-latest` runners are now ARM64 (M1). Use `macos-13` for Intel builds.

**Acceptance criteria:**
- Pushing a tag creates GitHub Release with 4 platform binaries + checksums
- Release notes are auto-generated from commits

**Dependencies:** Task 1

---

### Task 3: Install Script

**File:** `install.sh`

**Purpose:** One-liner installation for users who don't use Homebrew.

**Key requirements:**
- Detect OS: `uname -s` (Darwin/Linux)
- Detect arch: `uname -m` (arm64/x86_64)
- Map to release asset names
- Download from GitHub Releases API (latest or specified version)
- Verify SHA256 checksum
- Extract to `~/.local/bin` or `/usr/local/bin`
- Make executable
- Print success message with next steps

**Usage:**
```bash
# Latest version
curl -fsSL https://raw.githubusercontent.com/gerunddev/jjazy/main/install.sh | bash

# Specific version
curl -fsSL https://raw.githubusercontent.com/gerunddev/jjazy/main/install.sh | bash -s v0.1.0
```

**Acceptance criteria:**
- Works on macOS ARM64 and x86_64
- Works on Linux ARM64 and x86_64
- Verifies checksums
- Clear error messages on failure

**Dependencies:** Task 2 (needs release asset URL pattern)

---

### Task 4: Homebrew Formula

**File:** `~/Developer/homebrew-tap/Formula/jjazy.rb` (in the tap repo)

**Purpose:** Homebrew formula that downloads pre-built binaries.

**Key requirements:**
- Download correct binary based on Hardware::CPU (arm64 vs x86_64)
- Verify SHA256 checksum (per-architecture)
- Install binary to bin
- Include test block
- Support both Apple Silicon and Intel Macs

**Example structure:**
```ruby
class Jjazy < Formula
  desc "Terminal UI for jj (Jujutsu) version control"
  homepage "https://github.com/gerunddev/jjazy"
  version "0.1.0"
  license "MIT"  # or appropriate license

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/gerunddev/jjazy/releases/download/v#{version}/jjazy_darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER_ARM64_SHA"
    else
      url "https://github.com/gerunddev/jjazy/releases/download/v#{version}/jjazy_darwin_amd64.tar.gz"
      sha256 "PLACEHOLDER_AMD64_SHA"
    end
  end

  def install
    bin.install "jjazy"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/jjazy --version", 2)
  end
end
```

**Note:** SHA256 values must be updated after first release. Consider automating this.

**Acceptance criteria:**
- `brew install gerunddev/tap/jjazy` works on Apple Silicon
- `brew install gerunddev/tap/jjazy` works on Intel Mac
- `brew test jjazy` passes

**Dependencies:** Task 2 (needs actual release to get SHA256)

---

### Task 5: Homebrew Formula Auto-Update (Optional)

**File:** Modify `.github/workflows/release.yml`

**Purpose:** Automatically update Homebrew formula SHA256 values when new release is created.

**Key requirements:**
- After GoReleaser completes, calculate SHA256 of release assets
- Update formula in homebrew-tap repo via GitHub API or git push
- Use `HOMEBREW_TAP_TOKEN` secret for authentication

**Acceptance criteria:**
- New release automatically updates formula with correct SHA256 values
- No manual formula updates needed

**Dependencies:** Task 2, Task 4, Manual Step 3 (token)

---

### Task 6: Update README

**File:** `README.md`

**Purpose:** Document installation methods.

**Content to add:**
```markdown
## Installation

### Homebrew (macOS)

```bash
brew install gerunddev/tap/jjazy
```

### Install Script (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/gerunddev/jjazy/main/install.sh | bash
```

### Download Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/gerunddev/jjazy/releases).

### Build from Source

Requires Go 1.23+ and Rust 1.70+:

```bash
git clone https://github.com/gerunddev/jjazy.git
cd jjazy
make install
```

> **Note:** `go install github.com/gerunddev/jjazy@latest` does not work because this project includes a Rust FFI component that must be compiled locally.
```

**Acceptance criteria:**
- All installation methods documented
- Clear note about `go install` not working

**Dependencies:** Task 3, Task 4

---

### Task 7: Add Version Flag

**File:** `main.go`

**Purpose:** Support `--version` flag for Homebrew test block and general use.

**Key requirements:**
- Add `-version` / `--version` flag
- Print version set by ldflags during build
- Exit cleanly (Homebrew tests often use this)

**Acceptance criteria:**
- `jjazy --version` prints version and exits 0
- Works with version injected via `-ldflags "-X main.Version=..."`

**Dependencies:** none (can be done in parallel)

---

## Implementation Order

```
Task 7 (version flag) ─────────────────────────────────┐
                                                        │
Task 1 (goreleaser) ──► Task 2 (GH Actions) ──────────┼──► Task 5 (auto-update)
                                │                       │
                                ├──► Task 3 (install.sh)┤
                                │                       │
                                └──► Task 4 (formula) ──┘
                                            │
                                            └──► Task 6 (README)
```

**Parallel work:**
- Task 7 can be done anytime
- Task 3 and Task 4 can be done in parallel after Task 2
- Task 5 and Task 6 come last

**Critical path:** Task 1 → Task 2 → Task 4 → Task 6

---

## Testing Checklist

Before first release:
- [ ] `goreleaser build --snapshot --clean` works locally
- [ ] Version flag works: `./jjazy --version`

After first release:
- [ ] GitHub Release contains all 4 platform binaries
- [ ] GitHub Release contains checksums.txt
- [ ] Install script works on macOS ARM64
- [ ] Install script works on Linux (if you have access)
- [ ] `brew install gerunddev/tap/jjazy` works
- [ ] `brew test jjazy` passes
- [ ] Installed binary runs correctly in a jj repo

---

## File Summary

| File | Repository | Action |
|------|------------|--------|
| `.goreleaser.yaml` | jjazy | Create |
| `.github/workflows/release.yml` | jjazy | Create |
| `install.sh` | jjazy | Create |
| `main.go` | jjazy | Modify (add version flag) |
| `README.md` | jjazy | Modify |
| `Formula/jjazy.rb` | homebrew-tap | Create |

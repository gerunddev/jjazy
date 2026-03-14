#!/usr/bin/env python3
"""Generate Homebrew formula for jjazy with correct version and SHA256 sums."""

import os
import sys

version = os.environ["VERSION"]
sha_arm64 = os.environ["SHA256_ARM64"]
sha_amd64 = os.environ["SHA256_AMD64"]
sha_linux_arm64 = os.environ["SHA256_LINUX_ARM64"]
sha_linux_amd64 = os.environ["SHA256_LINUX_AMD64"]

formula = f"""class Jjazy < Formula
  desc "Terminal UI for jj (Jujutsu) version control"
  homepage "https://github.com/gerunddev/jjazy"
  version "{version}"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/gerunddev/jjazy/releases/download/v{version}/jjazy_darwin_arm64.tar.gz"
      sha256 "{sha_arm64}"
    else
      url "https://github.com/gerunddev/jjazy/releases/download/v{version}/jjazy_darwin_amd64.tar.gz"
      sha256 "{sha_amd64}"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/gerunddev/jjazy/releases/download/v{version}/jjazy_linux_arm64.tar.gz"
      sha256 "{sha_linux_arm64}"
    else
      url "https://github.com/gerunddev/jjazy/releases/download/v{version}/jjazy_linux_amd64.tar.gz"
      sha256 "{sha_linux_amd64}"
    end
  end

  def install
    bin.install "jjazy"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/jjazy --version")
  end
end
"""

output_path = sys.argv[1] if len(sys.argv) > 1 else "jjazy.rb"
with open(output_path, "w") as f:
    f.write(formula)

print(f"Generated formula for jjazy {version} at {output_path}")

# Canonical copy lives in orieken/homebrew-tap/Formula/assay.rb
# Keep this file in sync with that one.
#
# To update after a release:
#   1. Tag: git tag v2.x.x && git push origin v2.x.x
#   2. Compute sha256:
#        curl -sL https://github.com/orieken/assay/archive/refs/tags/v2.x.x.tar.gz | shasum -a 256
#   3. Set version and sha256 in both copies, then commit to homebrew-tap.

class Assay < Formula
  desc "Language-agnostic test scaffold generator"
  homepage "https://github.com/orieken/assay"
  url "https://github.com/orieken/assay/archive/refs/tags/v#{version}.tar.gz"
  version "2.0.0"
  sha256 "REPLACE_WITH_SHA256"
  license "MIT"
  head "https://github.com/orieken/assay.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = %W[
      -s -w
      -X github.com/orieken/assay/cmd/assay.Version=#{version}
    ]
    system "go", "build", *std_go_args(ldflags: ldflags.join(" ")), "./cmd/assay"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/assay version")
  end
end

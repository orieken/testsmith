# Formula for https://github.com/orieken/homebrew-tap
# Copy this file to orieken/homebrew-tap/Formula/assay.rb and update
# the url/sha256 fields after each release.
#
# SHA256 values are published in assay_VERSION_checksums.txt on the release page.

class Assay < Formula
  desc "Language-agnostic test scaffold generator"
  homepage "https://github.com/orieken/assay"
  version "2.0.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/orieken/assay/releases/download/v#{version}/assay_#{version}_darwin_arm64.tar.gz"
      sha256 "REPLACE_WITH_SHA256_DARWIN_ARM64"
    end

    on_intel do
      url "https://github.com/orieken/assay/releases/download/v#{version}/assay_#{version}_darwin_amd64.tar.gz"
      sha256 "REPLACE_WITH_SHA256_DARWIN_AMD64"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/orieken/assay/releases/download/v#{version}/assay_#{version}_linux_amd64.tar.gz"
      sha256 "REPLACE_WITH_SHA256_LINUX_AMD64"
    end
  end

  def install
    bin.install "assay"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/assay version")
  end
end

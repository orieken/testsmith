# Formula for https://github.com/orieken/homebrew-tap
# Copy this file to orieken/homebrew-tap/Formula/assay.rb and update
# the url/sha256 fields after each release.
#
# To generate sha256 for a new release:
#   curl -sL <url> | sha256sum

class Testsmith < Formula
  desc "Language-agnostic test scaffold generator"
  homepage "https://github.com/orieken/assay"
  version "2.0.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/orieken/assay/releases/download/v#{version}/assay-darwin-arm64"
      sha256 "REPLACE_WITH_SHA256_DARWIN_ARM64"

      def install
        bin.install "assay-darwin-arm64" => "assay"
      end
    end

    on_intel do
      url "https://github.com/orieken/assay/releases/download/v#{version}/assay-darwin-amd64"
      sha256 "REPLACE_WITH_SHA256_DARWIN_AMD64"

      def install
        bin.install "assay-darwin-amd64" => "assay"
      end
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/orieken/assay/releases/download/v#{version}/assay-linux-amd64"
      sha256 "REPLACE_WITH_SHA256_LINUX_AMD64"

      def install
        bin.install "assay-linux-amd64" => "assay"
      end
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/assay version")
  end
end

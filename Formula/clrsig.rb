class Clrsig < Formula
  desc "API Contract Validation and Spec Conformance Testing CLI"
  homepage "https://github.com/fuvito/api-workbench"
  version "0.8.5"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/fuvito/api-workbench/releases/download/v#{version}/clrsig-darwin-arm64"
      sha256 "48294c0350a0553fc642a89d73d955f3c1a69a6ea7a527c3caeafd83286d8c90"
    else
      url "https://github.com/fuvito/api-workbench/releases/download/v#{version}/clrsig-darwin-amd64"
      sha256 "cdf6ce3169791737390e81bf2d932c319db861dd332a4d1fe7bc7f742a62b516"
    end
  end

  on_linux do
    url "https://github.com/fuvito/api-workbench/releases/download/v#{version}/clrsig-linux-amd64"
    sha256 "f9cf2e690ef70aa0ae61edeefa1d8d25739a544e9c44346715e1fdfda98c951f"
  end

  def install
    if OS.mac?
      arch = Hardware::CPU.arm? ? "arm64" : "amd64"
      bin.install "clrsig-darwin-#{arch}" => "clrsig"
    elsif OS.linux?
      bin.install "clrsig-linux-amd64" => "clrsig"
    end
  end

  test do
    system "#{bin}/clrsig", "--help"
  end
end

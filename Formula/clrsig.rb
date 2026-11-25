class Clrsig < Formula
  desc "API Contract Validation and Spec Conformance Testing CLI"
  homepage "https://github.com/fuvito/api-workbench"
  version "0.13.2"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/fuvito/api-workbench/releases/download/v#{version}/clrsig-darwin-arm64"
      sha256 "a211e9b0b272c5f90218a3b82ad45b8ed3e546b704055c113ff05dd0444c7635"
    else
      url "https://github.com/fuvito/api-workbench/releases/download/v#{version}/clrsig-darwin-amd64"
      sha256 "ffbd665601de21f0f60ea403e6e66387817dd93b338dee1d18dcef6a3dce16d3"
    end
  end

  on_linux do
    url "https://github.com/fuvito/api-workbench/releases/download/v#{version}/clrsig-linux-amd64"
    sha256 "57fc5d6660e6658d99de8def59b8675cc7063127de2a14e95a2c218619d03e0b"
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

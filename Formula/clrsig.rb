class Clrsig < Formula
  desc "API Contract Validation and Spec Conformance Testing CLI"
  homepage "https://github.com/fuvito/api-workbench"
  version "0.13.5"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/fuvito/api-workbench/releases/download/v#{version}/clrsig-darwin-arm64"
      sha256 "437460273636e7af822c386b6cb43f08d1f4b080d612a32a018b5b6249ff5a01"
    else
      url "https://github.com/fuvito/api-workbench/releases/download/v#{version}/clrsig-darwin-amd64"
      sha256 "a9d22812de3b7343dc31ec13fe7ce1f6374ef6c114fde5acdf9578ae0726e311"
    end
  end

  on_linux do
    url "https://github.com/fuvito/api-workbench/releases/download/v#{version}/clrsig-linux-amd64"
    sha256 "dd05646d40d760588d6134ff3606e505a9cb676747c44fa438833b20336a6efd"
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

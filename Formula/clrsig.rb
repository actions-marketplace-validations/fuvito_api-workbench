class Clrsig < Formula
  desc "API Contract Validation and Spec Conformance Testing CLI"
  homepage "https://github.com/fuvito/api-workbench"
  version "0.11.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/fuvito/api-workbench/releases/download/v#{version}/clrsig-darwin-arm64"
      sha256 "02abc8e954dedd6a46414518045e951f6440fa15fe7c3ed987bd7d46ebaf3708"
    else
      url "https://github.com/fuvito/api-workbench/releases/download/v#{version}/clrsig-darwin-amd64"
      sha256 "59671c275043f9157cc2755556178e9d0115e9794724e5a01f20d3979acf2ec5"
    end
  end

  on_linux do
    url "https://github.com/fuvito/api-workbench/releases/download/v#{version}/clrsig-linux-amd64"
    sha256 "82105159910ed7536ea28ade25f7e7572f5900c00e86e985bb41add7d0cb2624"
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

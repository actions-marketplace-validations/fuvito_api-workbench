class Clrsig < Formula
  desc "API Contract Validation and Spec Conformance Testing CLI"
  homepage "https://github.com/fuvito/api-workbench"
  version "0.8.8"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/fuvito/api-workbench/releases/download/v#{version}/clrsig-darwin-arm64"
      sha256 "7d194fceb032af0bb87725106854e50bf0ec4bb38cc806344f25ce0ed083bb8a"
    else
      url "https://github.com/fuvito/api-workbench/releases/download/v#{version}/clrsig-darwin-amd64"
      sha256 "d35fa31c986d267030f8dca3f980cea8d0ee234ab85f861fe3e26ce6a12a5a08"
    end
  end

  on_linux do
    url "https://github.com/fuvito/api-workbench/releases/download/v#{version}/clrsig-linux-amd64"
    sha256 "4a94e9036b29440ec9ce05f652853a6343387572825f85c9580041c3e4189595"
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

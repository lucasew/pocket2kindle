let
  pkgs = import <nixpkgs> {};
in
pkgs.buildGoModule rec {
  name = "p2k";
  version = "0.0.1";
  vendorSha256 = "0z5flbrii5dqylv5pap3svbm652jgqdsmnlzfa8nx86h3drk2fjn";
  src = ./.;
  meta = with pkgs.lib; {
    description = "Pocket2Kindle: Fetch articles from pocket and send as ebook to kindle";
    homepage = "https://github.com/lucasew/pocket2kindle";
    platforms = platforms.linux;
  };
}

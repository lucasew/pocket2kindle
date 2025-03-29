{ buildGoModule
, calibre
, lib
}:

buildGoModule {
  name = "p2k";
  version = "0.0.1";

  src = ./.;

  vendorHash = "sha256-rSahRMBEj788k6BY2EujcxQ6LmFb7WSpMOr+bFE7sDk=";

  buildInputs = [
    calibre
  ];

  meta = with lib; {
    description = "Pocket2Kindle: Fetch articles from pocket and send as ebook to kindle";
    homepage = "https://github.com/lucasew/pocket2kindle";
    platforms = platforms.linux;
  };
}

with import <nixpkgs> {};

myEnvFun {
  name = "protobuf";
  buildInputs = [
    pkgconfig autoconf automake libtool autoreconfHook zlib
  ];
}

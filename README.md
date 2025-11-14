# Fosilo

An Ace of Spades v0.75 dedicated server written in Go.

## Features

- Supports the entire protocol (including community extensions)
- Supports the following game modes: CTF, TDM, Babel, Arena, TC
- Add server to the BuildAndShoot and aos.coffee masterservers
- Plugin system for commands and gamemodes in Lua

## Installation

1. Ensure you have ENet installed on your system from your favorite package manager
2. Grab pre-compiled binary or build from source
3. Edit config file of game mode you wish to run, change passowrds, setup map pool, enable registration in the masterserver etc.
4. Launch the server, for example we will be using CTF
`./fosilo start --config config/config-ctf.toml`

### Building from Source

1. Clone the repository:
```bash
git clone https://github.com/siohaza/fosilo.git
cd fosilo
```

2. Install ENet from your favorite package manager and then Golang dependencies:
```bash
go mod download
```

3. Build the server:
```bash
make build
# OR
go build -o fosilo ./cmd/fosilo
```

The compiled binary will be created as `fosilo` in the project root.

For detailed Lua API documentation see [this](docs/lua.md) file.

## License

[GPLv3](LICENSE)

## Credits

- [libspades](https://codeberg.org/totallynotaburner/libspades) and [piqueserver](https://piqueserver.github.io/aosprotocol/) - 0.75 protocol documentation with extensions
- [SpadesX](https://github.com/SpadesX/SpadesX) - reference for physics code, vxl map parser
- [libvxl](https://github.com/xtreme8000/libvxl) - reference for vxl map parser

- [NotABurner](https://codeberg.org/totallynotaburner) - major help with server debugging, design suggestions
- [sByte](https://github.com/DryByte) - design suggestions
